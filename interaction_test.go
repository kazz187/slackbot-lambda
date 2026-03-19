package slackbot

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/slack-go/slack"
)

func TestEventRoutes_Register(t *testing.T) {
	t.Run("new registration", func(t *testing.T) {
		er := EventRoutes{}
		handler := func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
			return nil
		}
		er.Register(slack.InteractionTypeBlockActions, "action1", handler)
		if _, ok := er[slack.InteractionTypeBlockActions]["action1"]; !ok {
			t.Error("handler not registered")
		}
	})

	t.Run("same type different action", func(t *testing.T) {
		er := EventRoutes{}
		er.Register(slack.InteractionTypeBlockActions, "action1", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error { return nil })
		er.Register(slack.InteractionTypeBlockActions, "action2", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error { return nil })
		if len(er[slack.InteractionTypeBlockActions]) != 2 {
			t.Errorf("expected 2 actions, got %d", len(er[slack.InteractionTypeBlockActions]))
		}
	})

	t.Run("overwrite", func(t *testing.T) {
		er := EventRoutes{}
		called := ""
		er.Register(slack.InteractionTypeBlockActions, "action1", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
			called = "first"
			return nil
		})
		er.Register(slack.InteractionTypeBlockActions, "action1", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
			called = "second"
			return nil
		})
		_ = er.Run(context.Background(), slack.InteractionTypeBlockActions, &slack.BlockAction{ActionID: "action1"}, slack.InteractionCallback{})
		if called != "second" {
			t.Errorf("expected second handler, got %q", called)
		}
	})
}

func TestEventRoutes_Run(t *testing.T) {
	t.Run("normal routing", func(t *testing.T) {
		er := EventRoutes{}
		called := false
		er.Register(slack.InteractionTypeBlockActions, "action1", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
			called = true
			return nil
		})
		err := er.Run(context.Background(), slack.InteractionTypeBlockActions, &slack.BlockAction{ActionID: "action1"}, slack.InteractionCallback{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !called {
			t.Error("handler not called")
		}
	})

	t.Run("unregistered type", func(t *testing.T) {
		er := EventRoutes{}
		err := er.Run(context.Background(), slack.InteractionTypeBlockActions, &slack.BlockAction{ActionID: "action1"}, slack.InteractionCallback{})
		if !errors.Is(err, ErrNotFoundInteractionType) {
			t.Errorf("expected ErrNotFoundInteractionType, got %v", err)
		}
	})

	t.Run("unregistered action", func(t *testing.T) {
		er := EventRoutes{}
		er.Register(slack.InteractionTypeBlockActions, "action1", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error { return nil })
		err := er.Run(context.Background(), slack.InteractionTypeBlockActions, &slack.BlockAction{ActionID: "unknown"}, slack.InteractionCallback{})
		if !errors.Is(err, ErrNotFoundActionValue) {
			t.Errorf("expected ErrNotFoundActionValue, got %v", err)
		}
	})
}

func makeInteractionBody(t *testing.T, callback slack.InteractionCallback) string {
	t.Helper()
	b, err := json.Marshal(callback)
	if err != nil {
		t.Fatalf("failed to marshal callback: %v", err)
	}
	return "payload=" + url.QueryEscape(string(b))
}

func TestInteractionHandler_Handle(t *testing.T) {
	t.Run("normal block actions", func(t *testing.T) {
		ih := NewInteractionHandler()
		called := false
		ih.EventRoutes.Register(slack.InteractionTypeBlockActions, "btn1", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
			called = true
			return nil
		})

		body := makeInteractionBody(t, slack.InteractionCallback{
			Type: slack.InteractionTypeBlockActions,
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{{ActionID: "btn1"}},
			},
		})

		resp, err := ih.Handle(context.Background(), events.APIGatewayProxyRequest{Body: body})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
		if !called {
			t.Error("handler not called")
		}
	})

	t.Run("handler error", func(t *testing.T) {
		ih := NewInteractionHandler()
		ih.EventRoutes.Register(slack.InteractionTypeBlockActions, "btn1", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
			return errors.New("handler failed")
		})

		body := makeInteractionBody(t, slack.InteractionCallback{
			Type: slack.InteractionTypeBlockActions,
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{{ActionID: "btn1"}},
			},
		})

		_, err := ih.Handle(context.Background(), events.APIGatewayProxyRequest{Body: body})
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("multiple actions one fails", func(t *testing.T) {
		ih := NewInteractionHandler()
		ih.EventRoutes.Register(slack.InteractionTypeBlockActions, "ok", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
			return nil
		})
		ih.EventRoutes.Register(slack.InteractionTypeBlockActions, "fail", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
			return errors.New("fail")
		})

		body := makeInteractionBody(t, slack.InteractionCallback{
			Type: slack.InteractionTypeBlockActions,
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{
					{ActionID: "ok"},
					{ActionID: "fail"},
				},
			},
		})

		_, err := ih.Handle(context.Background(), events.APIGatewayProxyRequest{Body: body})
		if err == nil {
			t.Error("expected error from failing handler")
		}
	})

	t.Run("multiple actions both fail", func(t *testing.T) {
		ih := NewInteractionHandler()
		ih.EventRoutes.Register(slack.InteractionTypeBlockActions, "fail1", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
			return errors.New("fail1")
		})
		ih.EventRoutes.Register(slack.InteractionTypeBlockActions, "fail2", func(ctx context.Context, action *slack.BlockAction, callback slack.InteractionCallback) error {
			return errors.New("fail2")
		})

		body := makeInteractionBody(t, slack.InteractionCallback{
			Type: slack.InteractionTypeBlockActions,
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{
					{ActionID: "fail1"},
					{ActionID: "fail2"},
				},
			},
		})

		_, err := ih.Handle(context.Background(), events.APIGatewayProxyRequest{Body: body})
		if err == nil {
			t.Error("expected joined error")
		}
	})

	t.Run("unregistered route is ignored", func(t *testing.T) {
		ih := NewInteractionHandler()

		body := makeInteractionBody(t, slack.InteractionCallback{
			Type: slack.InteractionTypeBlockActions,
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{{ActionID: "unknown"}},
			},
		})

		_, err := ih.Handle(context.Background(), events.APIGatewayProxyRequest{Body: body})
		if err != nil {
			t.Errorf("unregistered routes should be ignored, got: %v", err)
		}
	})

	t.Run("invalid url encoding", func(t *testing.T) {
		_, err := ih.Handle(context.Background(), events.APIGatewayProxyRequest{Body: "%zz"})
		if err == nil {
			t.Error("expected error for invalid URL encoding")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		ih := NewInteractionHandler()
		_, err := ih.Handle(context.Background(), events.APIGatewayProxyRequest{Body: "payload=not-json"})
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("unsupported interaction type", func(t *testing.T) {
		ih := NewInteractionHandler()
		body := makeInteractionBody(t, slack.InteractionCallback{
			Type: slack.InteractionTypeShortcut,
		})

		resp, err := ih.Handle(context.Background(), events.APIGatewayProxyRequest{Body: body})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("empty actions array", func(t *testing.T) {
		ih := NewInteractionHandler()
		body := makeInteractionBody(t, slack.InteractionCallback{
			Type: slack.InteractionTypeBlockActions,
			ActionCallback: slack.ActionCallbacks{
				BlockActions: []*slack.BlockAction{},
			},
		})

		resp, err := ih.Handle(context.Background(), events.APIGatewayProxyRequest{Body: body})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})
}

var ih = NewInteractionHandler()
