package slackbot

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/slack-go/slack/slackevents"
)

func TestRegisterHandler(t *testing.T) {
	eh := NewEventHandler(nil)

	// Test registering AppMentionEvent handler
	appMentionCalled := false
	RegisterHandler(eh, func(ctx context.Context, ev *slackevents.AppMentionEvent) error {
		appMentionCalled = true
		if ev.Text != "test message" {
			t.Errorf("expected text 'test message', got '%s'", ev.Text)
		}
		return nil
	})

	// Test registering AppHomeOpenedEvent handler
	appHomeOpenedCalled := false
	RegisterHandler(eh, func(ctx context.Context, ev *slackevents.AppHomeOpenedEvent) error {
		appHomeOpenedCalled = true
		if ev.Tab != "home" {
			t.Errorf("expected tab 'home', got '%s'", ev.Tab)
		}
		return nil
	})

	// Test handling AppMentionEvent
	appMentionEvent := slackevents.EventsAPIEvent{
		InnerEvent: slackevents.EventsAPIInnerEvent{
			Data: &slackevents.AppMentionEvent{
				Text: "test message",
			},
		},
	}
	err := eh.callbackEvent.Handle(context.Background(), appMentionEvent)
	if err != nil {
		t.Errorf("unexpected error handling AppMentionEvent: %v", err)
	}
	if !appMentionCalled {
		t.Error("AppMentionEvent handler was not called")
	}

	// Test handling AppHomeOpenedEvent
	appHomeOpenedEvent := slackevents.EventsAPIEvent{
		InnerEvent: slackevents.EventsAPIInnerEvent{
			Data: &slackevents.AppHomeOpenedEvent{
				Tab: "home",
			},
		},
	}
	err = eh.callbackEvent.Handle(context.Background(), appHomeOpenedEvent)
	if err != nil {
		t.Errorf("unexpected error handling AppHomeOpenedEvent: %v", err)
	}
	if !appHomeOpenedCalled {
		t.Error("AppHomeOpenedEvent handler was not called")
	}

	// Test handling unregistered event type
	unregisteredEvent := slackevents.EventsAPIEvent{
		InnerEvent: slackevents.EventsAPIInnerEvent{
			Data: &slackevents.MessageEvent{},
		},
	}
	err = eh.callbackEvent.Handle(context.Background(), unregisteredEvent)
	if err == nil {
		t.Error("expected error for unregistered event type")
	}
}

func TestHandleURLVerification(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		body := `{"token":"tok","challenge":"chall","type":"url_verification"}`
		res, err := HandleURLVerification(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Challenge != "chall" {
			t.Errorf("expected challenge 'chall', got %q", res.Challenge)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := HandleURLVerification("{invalid}")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := HandleURLVerification("")
		if err == nil {
			t.Error("expected error for empty string")
		}
	})
}

func TestEventHandler_HandleEvent(t *testing.T) {
	verificationToken := "test-verification-token"

	t.Run("url verification", func(t *testing.T) {
		eh := NewEventHandler(NewSSMLoaderMock("token", verificationToken))
		body := `{"token":"` + verificationToken + `","challenge":"my-challenge","type":"url_verification"}`

		resp, err := eh.HandleEvent(context.Background(), events.APIGatewayProxyRequest{Body: body})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		if resp.Body != "my-challenge" {
			t.Errorf("expected body 'my-challenge', got %q", resp.Body)
		}
	})

	t.Run("callback event with registered handler", func(t *testing.T) {
		eh := NewEventHandler(NewSSMLoaderMock("token", verificationToken))
		called := false
		RegisterHandler(eh, func(ctx context.Context, ev *slackevents.AppMentionEvent) error {
			called = true
			return nil
		})

		body := `{"token":"` + verificationToken + `","type":"event_callback","event":{"type":"app_mention","text":"hi"}}`
		resp, err := eh.HandleEvent(context.Background(), events.APIGatewayProxyRequest{Body: body})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		if !called {
			t.Error("handler should be called")
		}
	})

	t.Run("callback event with unregistered handler", func(t *testing.T) {
		eh := NewEventHandler(NewSSMLoaderMock("token", verificationToken))

		body := `{"token":"` + verificationToken + `","type":"event_callback","event":{"type":"app_mention","text":"hi"}}`
		_, err := eh.HandleEvent(context.Background(), events.APIGatewayProxyRequest{Body: body})
		if err == nil {
			t.Error("expected error for unregistered handler")
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		eh := NewEventHandler(NewSSMLoaderMock("token", verificationToken))
		body := `{"token":"wrong-token","type":"event_callback","event":{"type":"app_mention","text":"hi"}}`

		resp, err := eh.HandleEvent(context.Background(), events.APIGatewayProxyRequest{Body: body})
		if err == nil {
			t.Error("expected error for invalid token")
		}
		if resp.StatusCode != 401 {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		eh := NewEventHandler(NewSSMLoaderMock("token", verificationToken))
		_, err := eh.HandleEvent(context.Background(), events.APIGatewayProxyRequest{Body: "{invalid}"})
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestCallbackEvent_Handle_HandlerReturnsError(t *testing.T) {
	eh := NewEventHandler(nil)
	handlerErr := errors.New("handler error")
	RegisterHandler(eh, func(ctx context.Context, ev *slackevents.AppMentionEvent) error {
		return handlerErr
	})

	ev := slackevents.EventsAPIEvent{
		InnerEvent: slackevents.EventsAPIInnerEvent{
			Data: &slackevents.AppMentionEvent{Text: "test"},
		},
	}
	err := eh.callbackEvent.Handle(context.Background(), ev)
	if !errors.Is(err, handlerErr) {
		t.Errorf("expected handler error to propagate, got: %v", err)
	}
}
