package slackbot

import (
	"net/http"
	"testing"
)

func TestNewResponse(t *testing.T) {
	resp := NewResponse(http.StatusOK)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if resp.Headers["x-slack-no-retry"] != "1" {
		t.Errorf("expected x-slack-no-retry header to be '1', got %q", resp.Headers["x-slack-no-retry"])
	}
}
