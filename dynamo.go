package main

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
)

type Article struct {
	Id      string `json:"id" dynamodbav:"id"`
	Title   string `json:"title" dynamodbav:"title"`
	Content string `json:"content" dynamodbav:"content"`
}

const TableName = "Articles"

var db dynamodb.Client

func init() {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	db = *dynamodb.NewFromConfig(sdkConfig)
}

func getArticle(ctx context.Context, id string) (*Article, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"id": key,
		},
	}

	log.Printf("Calling Dynamodb with input: %v", input)
	result, err := db.GetItem(ctx, input)
	if err != nil {
		return nil, err
	}
	log.Printf("Executed GetItem DynamoDb successfully. Result: %#v", result)

	if result.Item == nil {
		return nil, nil
	}

	article := new(Article)
	err = attributevalue.UnmarshalMap(result.Item, article)
	if err != nil {
		return nil, err
	}

	return article, nil
}

func insertArticle(ctx context.Context, createArticle CreateArticle) (*Article, error) {
	article := Article{
		Title:   createArticle.Title,
		Content: createArticle.Content,
		Id:      uuid.NewString(),
	}

	item, err := attributevalue.MarshalMap(article)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
	}

	res, err := db.PutItem(ctx, input)
	if err != nil {
		return nil, err
	}

	err = attributevalue.UnmarshalMap(res.Attributes, &article)
	if err != nil {
		return nil, err
	}

	return &article, nil
}

func deleteArticle(ctx context.Context, id string) (*Article, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"id": key,
		},
		ReturnValues: types.ReturnValue(*aws.String("ALL_OLD")),
	}

	res, err := db.DeleteItem(ctx, input)
	if err != nil {
		return nil, err
	}

	if res.Attributes == nil {
		return nil, nil
	}

	article := new(Article)
	err = attributevalue.UnmarshalMap(res.Attributes, article)
	if err != nil {
		return nil, err
	}

	return article, nil
}

func listArticles(ctx context.Context) ([]Article, error) {
	articles := make([]Article, 0)
	var token map[string]types.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName:         aws.String(TableName),
			ExclusiveStartKey: token,
		}

		result, err := db.Scan(ctx, input)
		if err != nil {
			return nil, err
		}

		var fetchedArticles []Article
		err = attributevalue.UnmarshalListOfMaps(result.Items, &fetchedArticles)
		if err != nil {
			return nil, err
		}

		articles = append(articles, fetchedArticles...)
		token = result.LastEvaluatedKey
		if token == nil {
			break
		}
	}

	return articles, nil
}

func updateItem(ctx context.Context, id string, updateArticle UpdateArticle) (*Article, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	expr, err := expression.NewBuilder().WithUpdate(
		expression.Set(
			expression.Name("title"),
			expression.Value(updateArticle.Title),
		).Set(
			expression.Name("content"),
			expression.Value(updateArticle.Content),
		),
	).WithCondition(
		expression.Equal(
			expression.Name("id"),
			expression.Value(id),
		),
	).Build()
	if err != nil {
		return nil, err
	}

	input := &dynamodb.UpdateItemInput{
		Key: map[string]types.AttributeValue{
			"id": key,
		},
		TableName:                 aws.String(TableName),
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		ConditionExpression:       expr.Condition(),
		ReturnValues:              types.ReturnValue(*aws.String("ALL_NEW")),
	}

	res, err := db.UpdateItem(ctx, input)
	if err != nil {
		var smErr *smithy.OperationError
		if errors.As(err, &smErr) {
			var condCheckFailed *types.ConditionalCheckFailedException
			if errors.As(err, &condCheckFailed) {
				return nil, nil
			}
		}

		return nil, err
	}

	if res.Attributes == nil {
		return nil, nil
	}

	article := new(Article)
	err = attributevalue.UnmarshalMap(res.Attributes, article)
	if err != nil {
		return nil, err
	}

	return article, nil
}
