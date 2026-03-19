package slackbot

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func computeSlackSignature(secret, timestamp, body string) string {
	sigBaseString := fmt.Sprintf("v0:%s:%s", timestamp, body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sigBaseString))
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}
