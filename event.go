package slackbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/aws/aws-lambda-go/events"
	"github.com/slack-go/slack/slackevents"
)

type EventHandler struct {
	callbackEvent          *callbackEvent
	slackVerificationToken *SSMLoader
}

type EventHandlerFunc[T any] func(ctx context.Context, ev T) error

func RegisterHandler[T any](eh *EventHandler, handler EventHandlerFunc[T]) {
	eventType := reflect.TypeFor[T]()
	eh.callbackEvent.handlers[eventType] = handler
}

func NewEventHandler(slackVerificationToken *SSMLoader) *EventHandler {
	return &EventHandler{
		callbackEvent:          newCallbackEvent(),
		slackVerificationToken: slackVerificationToken,
	}
}

func (eh *EventHandler) HandleEvent(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(request.Body),
		slackevents.OptionVerifyToken(&slackevents.TokenComparator{
			VerificationToken: eh.slackVerificationToken.MustGet(ctx),
		}),
	)
	if err != nil {
		return NewResponse(http.StatusUnauthorized), fmt.Errorf("failed to parse event: %s, %w", request.Body, err)
	}

	switch eventsAPIEvent.Type {
	case slackevents.URLVerification:
		res, err := HandleURLVerification(request.Body)
		if err != nil {
			return NewResponse(http.StatusUnauthorized), fmt.Errorf("failed to handle url verification: %w", err)
		}
		resp := NewResponse(http.StatusOK)
		resp.Body = res.Challenge
		return resp, nil
	case slackevents.CallbackEvent:
		if err := eh.callbackEvent.Handle(ctx, eventsAPIEvent); err != nil {
			return NewResponse(http.StatusOK), fmt.Errorf("failed to handle callback event: %w", err)
		}
	}
	return NewResponse(http.StatusOK), nil
}

func HandleURLVerification(body string) (*slackevents.ChallengeResponse, error) {
	var res *slackevents.ChallengeResponse
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal body: %w", err)
	}
	return res, nil
}

var ErrPermissionDenied = errors.New("permission denied")

type callbackEvent struct {
	handlers map[reflect.Type]any
}

func newCallbackEvent() *callbackEvent {
	return &callbackEvent{
		handlers: make(map[reflect.Type]any),
	}
}

func (ce *callbackEvent) Handle(ctx context.Context, ev slackevents.EventsAPIEvent) error {
	innerEvent := ev.InnerEvent
	eventType := reflect.TypeOf(innerEvent.Data)

	handler, exists := ce.handlers[eventType]
	if !exists {
		return fmt.Errorf("no handler registered for event type: %s", eventType)
	}

	handlerValue := reflect.ValueOf(handler)
	args := []reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(innerEvent.Data),
	}

	results := handlerValue.Call(args)
	if len(results) > 0 && !results[0].IsNil() {
		return results[0].Interface().(error)
	}

	return nil
}
