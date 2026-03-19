package slackbot

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSlackVerificationMiddleware(t *testing.T) {
	secret := "test-secret"
	body := `{"text":"hello"}`
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	makeValidRequest := func() *http.Request {
		sig := computeSlackSignature(secret, timestamp, body)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
		req.Header.Set("X-Slack-Request-Timestamp", timestamp)
		req.Header.Set("X-Slack-Signature", sig)
		return req
	}

	t.Run("retry request returns 200 without calling next", func(t *testing.T) {
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
		})

		middleware := SlackVerificationMiddleware(func(r *http.Request) (string, error) {
			return secret, nil
		})(next)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
		req.Header.Set("X-Slack-Retry-Num", "1")

		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		if nextCalled {
			t.Error("next handler should not be called for retry requests")
		}
	})

	t.Run("body read failure returns 400", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		middleware := SlackVerificationMiddleware(func(r *http.Request) (string, error) {
			return secret, nil
		})(next)

		req := httptest.NewRequest(http.MethodPost, "/", errReader{})
		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("secret retrieval failure returns 500", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		middleware := SlackVerificationMiddleware(func(r *http.Request) (string, error) {
			return "", errors.New("ssm error")
		})(next)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rr.Code)
		}
	})

	t.Run("invalid signature returns 401", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		middleware := SlackVerificationMiddleware(func(r *http.Request) (string, error) {
			return secret, nil
		})(next)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
		req.Header.Set("X-Slack-Request-Timestamp", timestamp)
		req.Header.Set("X-Slack-Signature", "v0=invalidsig")

		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("valid request calls next handler and body is re-readable", func(t *testing.T) {
		nextCalled := false
		var bodyFromNext []byte
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			var err error
			bodyFromNext, err = io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("failed to read body in next handler: %v", err)
			}
			w.WriteHeader(http.StatusOK)
		})

		middleware := SlackVerificationMiddleware(func(r *http.Request) (string, error) {
			return secret, nil
		})(next)

		req := makeValidRequest()
		rr := httptest.NewRecorder()
		middleware.ServeHTTP(rr, req)

		if !nextCalled {
			t.Error("next handler should be called")
		}
		if string(bodyFromNext) != body {
			t.Errorf("body not re-readable: got %q, want %q", bodyFromNext, body)
		}
	})
}

// errReader is an io.Reader that always returns an error
type errReader struct{}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}
