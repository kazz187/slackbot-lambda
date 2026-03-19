package slackbot

import (
	"errors"
	"fmt"
	"testing"
)

func TestBotError_Error(t *testing.T) {
	inner := fmt.Errorf("something broke")
	be := NewBotError(CodeInternal, "test message", inner)
	got := be.Error()
	want := "[Internal] test message: something broke"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBotError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("root cause")
	be := NewBotError(CodeUnauthorized, "auth failed", inner)

	t.Run("errors.Is", func(t *testing.T) {
		if !errors.Is(be, inner) {
			t.Error("errors.Is should match inner error")
		}
	})

	t.Run("errors.As", func(t *testing.T) {
		var target *BotError
		if !errors.As(be, &target) {
			t.Error("errors.As should match BotError")
		}
		if target.Code != CodeUnauthorized {
			t.Errorf("expected CodeUnauthorized, got %s", target.Code)
		}
	})

	t.Run("wrapped chain", func(t *testing.T) {
		wrapped := fmt.Errorf("outer: %w", be)
		var target *BotError
		if !errors.As(wrapped, &target) {
			t.Error("errors.As should find BotError in chain")
		}
	})
}
