package slackbot

import (
	"context"
	"testing"
)

func TestSSMLoaderMock_Get(t *testing.T) {
	loader := NewSSMLoaderMock("test-key", "test-value")
	got, err := loader.Get(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != "test-value" {
		t.Errorf("got %q, want %q", got, "test-value")
	}
}

func TestSSMLoader_Get_CachedValue(t *testing.T) {
	loader := &SSMLoader{
		Key: "some-key",
	}
	loader.SetValue("cached-value")

	got, err := loader.Get(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != "cached-value" {
		t.Errorf("got %q, want %q", got, "cached-value")
	}
}

func TestSSMLoader_MustGet_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()

	loader := &SSMLoader{
		Key: "some-key",
		// SSMCli is nil, Value is nil → will panic
	}
	loader.MustGet(context.Background())
}
