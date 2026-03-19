package slackbot

import (
	"context"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestLambdaRouter_Handle(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectEvent    bool
		expectInteract bool
	}{
		{
			name:        "event path",
			path:        "/slack/event",
			expectEvent: true,
		},
		{
			name:           "interaction path",
			path:           "/slack/interaction",
			expectInteract: true,
		},
		{
			name: "unknown path returns 200",
			path: "/slack/other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventCalled := false
			interactionCalled := false

			router := NewRouter(
				func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
					eventCalled = true
					return NewResponse(200), nil
				},
				func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
					interactionCalled = true
					return NewResponse(200), nil
				},
			)

			resp, err := router.Handle(context.Background(), events.APIGatewayProxyRequest{Path: tt.path})
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if resp.StatusCode != 200 {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}
			if eventCalled != tt.expectEvent {
				t.Errorf("eventCalled = %v, want %v", eventCalled, tt.expectEvent)
			}
			if interactionCalled != tt.expectInteract {
				t.Errorf("interactionCalled = %v, want %v", interactionCalled, tt.expectInteract)
			}
		})
	}
}
