package slackbot

import (
	"context"
	"testing"

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
