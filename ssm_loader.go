package slackbot

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type SSMLoader struct {
	SSMCli *ssm.Client
	Key    string
	Value  *string
}

func NewSSMLoader(ssmClient *ssm.Client, key string) *SSMLoader {
	return &SSMLoader{
		SSMCli: ssmClient,
		Key:    key,
	}
}

func NewSSMLoaderMock(key string, value string) *SSMLoader {
	return &SSMLoader{
		Key:   key,
		Value: aws.String(value),
	}
}

func (l *SSMLoader) Get(ctx context.Context) (string, error) {
	if l.Value != nil {
		return *l.Value, nil
	}

	res, err := l.SSMCli.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &l.Key,
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", err
	}
	l.Value = res.Parameter.Value
	return *l.Value, nil
}

func (l *SSMLoader) MustGet(ctx context.Context) string {
	v, err := l.Get(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// SetValue sets the value of the SSMLoader for testing purposes.
func (l *SSMLoader) SetValue(value string) {
	l.Value = aws.String(value)
}
