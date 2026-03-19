package slackbot

import "fmt"

type Code string

const (
	CodeInternal        Code = "Internal"
	CodeUnauthorized    Code = "Unauthorized"
	CodeInvalidArgument Code = "InvalidArgument"
)

type BotError struct {
	Code    Code
	Message string
	Err     error
}

func NewBotError(code Code, message string, err error) *BotError {
	return &BotError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func (be *BotError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", be.Code, be.Message, be.Err)
}

func (be *BotError) Unwrap() error {
	return be.Err
}
