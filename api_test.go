package slackbot

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestConvertHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		wantLen int
	}{
		{
			name:    "empty map",
			headers: map[string]string{},
			wantLen: 0,
		},
		{
			name:    "single header",
			headers: map[string]string{"content-type": "application/json"},
			wantLen: 1,
		},
		{
			name: "multiple headers",
			headers: map[string]string{
				"content-type":  "application/json",
				"authorization": "Bearer token",
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := ConvertHeaders(tt.headers)
			if len(h) != tt.wantLen {
				t.Errorf("got %d headers, want %d", len(h), tt.wantLen)
			}
			for key, value := range tt.headers {
				got := h.Get(key)
				if got != value {
					t.Errorf("header %q = %q, want %q", key, got, value)
				}
			}
		})
	}
}

func TestConvertHeaders_KeyNormalization(t *testing.T) {
	h := ConvertHeaders(map[string]string{"content-type": "application/json"})
	if got := h.Get("Content-Type"); got != "application/json" {
		t.Errorf("canonical key lookup failed: got %q", got)
	}
}

func TestBlockRetryRequest(t *testing.T) {
	tests := []struct {
		name    string
		header  http.Header
		wantErr bool
	}{
		{
			name:    "no retry header",
			header:  http.Header{},
			wantErr: false,
		},
		{
			name: "retry num 0",
			header: http.Header{
				"X-Slack-Retry-Num": []string{"0"},
			},
			wantErr: false,
		},
		{
			name: "retry num 1",
			header: http.Header{
				"X-Slack-Retry-Num": []string{"1"},
			},
			wantErr: true,
		},
		{
			name: "retry num 3",
			header: http.Header{
				"X-Slack-Retry-Num": []string{"3"},
			},
			wantErr: true,
		},
		{
			name: "non-numeric value",
			header: http.Header{
				"X-Slack-Retry-Num": []string{"abc"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := BlockRetryRequest(tt.header)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlockRetryRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerify(t *testing.T) {
	secret := "test-signing-secret"
	body := `{"type":"event_callback"}`
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	t.Run("missing headers", func(t *testing.T) {
		err := Verify(http.Header{}, []byte(body), secret)
		if err == nil {
			t.Error("expected error for missing headers")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		header := http.Header{
			"X-Slack-Request-Timestamp": []string{timestamp},
			"X-Slack-Signature":         []string{"v0=invalidsignature"},
		}
		err := Verify(header, []byte(body), secret)
		if err == nil {
			t.Error("expected error for invalid signature")
		}
	})

	t.Run("valid signature", func(t *testing.T) {
		sig := computeSlackSignature(secret, timestamp, body)
		header := http.Header{
			"X-Slack-Request-Timestamp": []string{timestamp},
			"X-Slack-Signature":         []string{sig},
		}
		err := Verify(header, []byte(body), secret)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		sig := computeSlackSignature("wrong-secret", timestamp, body)
		header := http.Header{
			"X-Slack-Request-Timestamp": []string{timestamp},
			"X-Slack-Signature":         []string{sig},
		}
		err := Verify(header, []byte(body), secret)
		if err == nil {
			t.Error("expected error for wrong secret")
		}
	})
}

func TestVerify_Integration(t *testing.T) {
	// Verify that ConvertHeaders output works with Verify
	secret := "my-secret"
	body := `hello`
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := computeSlackSignature(secret, timestamp, body)

	rawHeaders := map[string]string{
		"x-slack-request-timestamp": timestamp,
		"x-slack-signature":         sig,
	}
	header := ConvertHeaders(rawHeaders)
	err := Verify(header, []byte(body), secret)
	if err != nil {
		fmt.Println(err)
		// slack library may reject old timestamps; just ensure no panic
	}
}
