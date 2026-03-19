package slackbot

import (
	"bytes"
	"io"
	"net/http"
)

// SlackVerificationMiddleware validates Slack signature for webhook requests
func SlackVerificationMiddleware(getSigningSecret func(r *http.Request) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Block retry requests
			if err := BlockRetryRequest(r.Header); err != nil {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Read request body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}

			// Get signing secret
			signingSecret, err := getSigningSecret(r)
			if err != nil {
				http.Error(w, "Failed to get signing secret", http.StatusInternalServerError)
				return
			}

			// Verify Slack signature
			if err := Verify(r.Header, body, signingSecret); err != nil {
				http.Error(w, "Signature verification failed", http.StatusUnauthorized)
				return
			}

			// Restore body for next handler
			r.Body = io.NopCloser(bytes.NewReader(body))

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}
