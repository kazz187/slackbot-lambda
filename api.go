package slackbot

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/slack-go/slack"
)

func ConvertHeaders(headers map[string]string) http.Header {
	h := http.Header{}
	for key, value := range headers {
		h.Set(key, value)
	}
	return h
}

func Verify(header http.Header, body []byte, signingSecret string) error {
	verifier, err := slack.NewSecretsVerifier(header, signingSecret)
	if err != nil {
		return fmt.Errorf("failed to create secrets verifier: %w", err)
	}

	if _, err := verifier.Write(body); err != nil {
		return fmt.Errorf("failed to write body to verifier: %w", err)
	}
	if err := verifier.Ensure(); err != nil {
		return fmt.Errorf("failed to verify request: %w", err)
	}
	return nil
}

func BlockRetryRequest(header http.Header) error {
	retryNumStr := header.Get("x-slack-retry-num")
	if retryNumStr != "" {
		retryNum, err := strconv.Atoi(retryNumStr)
		if err != nil {
			return fmt.Errorf("failed to parse retry num: %w", err)
		}
		if retryNum > 0 {
			return fmt.Errorf("retry request")
		}
	}
	return nil
}
