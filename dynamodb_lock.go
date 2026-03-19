package slackbot

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBLock struct {
	DynamoDBCli *dynamodb.Client
	TableName   string
}

func NewDynamoDBLock(dynamoDBCli *dynamodb.Client, tableName string) *DynamoDBLock {
	return &DynamoDBLock{
		DynamoDBCli: dynamoDBCli,
		TableName:   tableName,
	}
}

func (dl *DynamoDBLock) AcquireLock(ctx context.Context, userID string, ttl time.Duration) error {
	now := time.Now()
	expiresAt := now.Add(ttl).Unix()

	input := &dynamodb.PutItemInput{
		TableName: &dl.TableName,
		Item: map[string]types.AttributeValue{
			"UserID":    &types.AttributeValueMemberS{Value: userID},
			"ExpiresAt": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", expiresAt)},
		},
		ConditionExpression: aws.String("attribute_not_exists(UserID)"),
	}

	if _, err := dl.DynamoDBCli.PutItem(ctx, input); err != nil {
		return fmt.Errorf("failed to put item: %w", err)
	}

	return nil
}

func (dl *DynamoDBLock) ReleaseLock(ctx context.Context, userID string) error {
	input := &dynamodb.DeleteItemInput{
		TableName: &dl.TableName,
		Key: map[string]types.AttributeValue{
			"UserID": &types.AttributeValueMemberS{Value: userID},
		},
	}

	if _, err := dl.DynamoDBCli.DeleteItem(ctx, input); err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	return nil
}
