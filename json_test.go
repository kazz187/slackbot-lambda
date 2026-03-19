package slackbot

import (
	"math"
	"testing"
)

func TestMustJSONMarshal(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		got := MustJSONMarshal(map[string]string{"key": "value"})
		want := `{"key":"value"}`
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("panic on unmarshalable value", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		MustJSONMarshal(math.Inf(1))
	})
}

func TestMustJSONUnmarshal(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		got := MustJSONUnmarshal[map[string]string](`{"key":"value"}`)
		if got["key"] != "value" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("panic on invalid JSON", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		MustJSONUnmarshal[map[string]string](`{invalid}`)
	})
}
