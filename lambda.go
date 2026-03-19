package slackbot

import (
	"context"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type LambdaHandler func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

type LambdaRouter struct {
	EventHandler       LambdaHandler
	InteractionHandler LambdaHandler
}

func NewRouter(eventHandler, interactionHandler LambdaHandler) *LambdaRouter {
	return &LambdaRouter{
		EventHandler:       eventHandler,
		InteractionHandler: interactionHandler,
	}
}

func (r *LambdaRouter) Handle(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if strings.HasSuffix(request.Path, "event") {
		return r.EventHandler(ctx, request)
	} else if strings.HasSuffix(request.Path, "interaction") {
		return r.InteractionHandler(ctx, request)
	}
	return NewResponse(http.StatusOK), nil
}
