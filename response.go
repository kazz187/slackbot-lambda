package slackbot

import "github.com/aws/aws-lambda-go/events"

func NewResponse(code int) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Headers: map[string]string{
			"x-slack-no-retry": "1",
		},
	}
}
