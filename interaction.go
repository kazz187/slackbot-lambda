package slackbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/slack-go/slack"
)

type Payload struct {
	Message Message `json:"message"`
}
type Message struct {
	Root Root `json:"root"`
}
type Root struct {
	User string `json:"user"`
}

type InteractionHandler struct {
	EventRoutes EventRoutes
}

type InteractionCallbackHandler func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error

type EventRoutes map[slack.InteractionType]map[string]InteractionCallbackHandler

var (
	ErrNotFoundInteractionType = fmt.Errorf("not found interaction type")
	ErrNotFoundActionValue     = fmt.Errorf("not found action value")
)

func (er EventRoutes) Register(kind slack.InteractionType, actionID string, handler InteractionCallbackHandler) {
	if _, ok := er[kind]; !ok {
		er[kind] = map[string]InteractionCallbackHandler{}
	}
	er[kind][actionID] = handler
}

func (er EventRoutes) Run(ctx context.Context, kind slack.InteractionType, action *slack.BlockAction, callback slack.InteractionCallback) error {
	routes, ok := er[kind]
	if !ok {
		return ErrNotFoundInteractionType
	}

	fn, ok := routes[action.ActionID]
	if !ok {
		return ErrNotFoundActionValue
	}

	return fn(ctx, action, callback)
}

func NewInteractionHandler() *InteractionHandler {
	return &InteractionHandler{
		EventRoutes: EventRoutes{},
	}
}

func (ih *InteractionHandler) Handle(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	raw, err := url.QueryUnescape(request.Body)
	if err != nil {
		return NewResponse(http.StatusOK), fmt.Errorf("failed to unescape payload: %w", err)
	}
	raw = strings.Replace(raw, "payload=", "", 1)
	var payload slack.InteractionCallback
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return NewResponse(http.StatusOK), fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	var retErr error
	switch payload.Type {
	// For BlockKit
	case slack.InteractionTypeBlockActions, slack.InteractionTypeBlockSuggestion, slack.InteractionTypeViewSubmission, slack.InteractionTypeViewClosed:
		for _, action := range payload.ActionCallback.BlockActions {
			if err := ih.EventRoutes.Run(ctx, payload.Type, action, payload); err != nil {
				if !(errors.Is(err, ErrNotFoundInteractionType) || errors.Is(err, ErrNotFoundActionValue)) {
					if retErr == nil {
						retErr = err
					} else {
						retErr = errors.Join(retErr, err)
					}
				}
			}
		}
	}
	if retErr != nil {
		return NewResponse(http.StatusOK), retErr
	}

	return NewResponse(http.StatusOK), nil
}
