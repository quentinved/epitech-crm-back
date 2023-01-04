package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator"
)

type CreateArticle struct {
	Title   string `json:"Title" validate:"required"`
	Content string `json:"Content" validate:"required"`
	Tag     string `json:"Tag"`
}

type UpdateArticle struct {
	Title   string `json:"Title" validate:"required"`
	Content string `json:"Content" validate:"required"`
	Tag     string `json:"Tag"`
}

var validate *validator.Validate = validator.New()

func router(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received req %#v", req)
	switch req.HTTPMethod {
	case "GET":
		return processGet(ctx, req)
	case "POST":
		return processPost(ctx, req)
	case "DELETE":
		return processDelete(ctx, req)
	case "PUT":
		return processPut(ctx, req)
	default:
		return clientError(http.StatusMethodNotAllowed)
	}
}

func processGet(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	tag, okTag := req.QueryStringParameters["tag"]

	if okTag {
		return processGetArticlesByTag(ctx, tag)
	} else if !ok {
		return processGetArticles(ctx)
	} else {
		return processGetArticle(ctx, id)
	}
}

func clientError(status int) (events.APIGatewayProxyResponse, error) {

	return events.APIGatewayProxyResponse{
		Body:       http.StatusText(status),
		StatusCode: status,
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	log.Println(err.Error())

	return events.APIGatewayProxyResponse{
		Body:       http.StatusText(http.StatusInternalServerError),
		StatusCode: http.StatusInternalServerError,
	}, nil
}

func processGetArticlesByTag(ctx context.Context, tag string) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received GET article request with Tag = %s", tag)

	article, err := getArticleByTag(ctx, tag)
	if err != nil {
		return serverError(err)
	}

	if article == nil {
		return clientError(http.StatusNotFound)
	}

	json, err := json.Marshal(article)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully fetched article item %s", json)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET",
			"Access-Control-Allow-Headers": "Content-Type",
		},
		Body: string(json),
	}, nil
}

func processGetArticle(ctx context.Context, id string) (events.APIGatewayProxyResponse, error) {
	log.Printf("Received GET article request with ID = %s", id)

	article, err := getArticle(ctx, id)
	if err != nil {
		return serverError(err)
	}

	if article == nil {
		return clientError(http.StatusNotFound)
	}

	json, err := json.Marshal(article)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully fetched article item %s", json)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET",
			"Access-Control-Allow-Headers": "Content-Type",
		},
		Body: string(json),
	}, nil
}

func processGetArticles(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	log.Print("Received GET articles request")

	articles, err := listArticles(ctx)
	if err != nil {
		return serverError(err)
	}

	json, err := json.Marshal(articles)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully fetched articles: %s", json)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET",
			"Access-Control-Allow-Headers": "Content-Type",
		},
		Body: string(json),
	}, nil
}

func processPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token := req.Headers["Authorization"]
	unautorized := checkGroupCognito(token, "admin")
	if unautorized.StatusCode != 0 {
		return unautorized, nil
	}
	var createArticle CreateArticle
	err := json.Unmarshal([]byte(req.Body), &createArticle)
	if err != nil {
		log.Printf("Can't unmarshal body: %v", err)
		return clientError(http.StatusUnprocessableEntity)
	}

	err = validate.Struct(&createArticle)
	if err != nil {
		log.Printf("Invalid bodytest: %v", err)
		return clientError(http.StatusBadRequest)
	}
	log.Printf("Received POST request with item: %+v", createArticle)

	res, err := insertArticle(ctx, createArticle)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Inserted new article: %+v", res)

	json, err := json.Marshal(res)
	if err != nil {
		return serverError(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Body:       string(json),
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "POST",
			"Access-Control-Allow-Headers": "Content-Type",
			"Location":                     fmt.Sprintf("/article/%s", res.Id),
		},
	}, nil
}

func processDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token := req.Headers["Authorization"]
	unautorized := checkGroupCognito(token, "admin")
	if unautorized.StatusCode != 0 {
		return unautorized, nil
	}
	id, ok := req.PathParameters["id"]
	if !ok {
		return clientError(http.StatusBadRequest)
	}
	log.Printf("Received DELETE request with id = %s", id)

	article, err := deleteArticle(ctx, id)
	if err != nil {
		return serverError(err)
	}

	if article == nil {
		return clientError(http.StatusNotFound)
	}

	json, err := json.Marshal(article)
	if err != nil {
		return serverError(err)
	}
	log.Printf("Successfully deleted article item %+v", article)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "DELETE",
			"Access-Control-Allow-Headers": "Content-Type",
		},
	}, nil
}

func processPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return clientError(http.StatusBadRequest)
	}

	var updateArticle UpdateArticle
	err := json.Unmarshal([]byte(req.Body), &updateArticle)
	if err != nil {
		log.Printf("Can't unmarshal body: %v", err)
		return clientError(http.StatusUnprocessableEntity)
	}

	err = validate.Struct(&updateArticle)
	if err != nil {
		log.Printf("Invalid body: %v", err)
		return clientError(http.StatusBadRequest)
	}
	log.Printf("Received PUT request with item: %+v", updateArticle)

	res, err := updateItem(ctx, id, updateArticle)
	if err != nil {
		return serverError(err)
	}

	if res == nil {
		return clientError(http.StatusNotFound)
	}

	log.Printf("Updated todo: %+v", res)

	json, err := json.Marshal(res)
	if err != nil {
		return serverError(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(json),
		Headers: map[string]string{
			"Location":                     fmt.Sprintf("/todo/%s", res.Id),
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "PUT",
			"Access-Control-Allow-Headers": "Content-Type",
		},
	}, nil
}

// id_token=eyJraWQiOiJzK211UE9FWkFKWlE5blwvN1JxOHpPNnJPWWtMNVJjbUdzSFwvS3RJMVNTd3M9IiwiYWxnIjoiUlMyNTYifQ.eyJhdF9oYXNoIjoiX1FVRzJVRVRzNi1Rak9YdEdOSEpfUSIsInN1YiI6IjNhNTk1NTU1LTdjNDQtNGY2NS1hZmYxLTQ0ODBlYjM2MzFiNyIsImNvZ25pdG86Z3JvdXBzIjpbIkFkbWluIl0sImVtYWlsX3ZlcmlmaWVkIjp0cnVlLCJpc3MiOiJodHRwczpcL1wvY29nbml0by1pZHAuZXUtd2VzdC0zLmFtYXpvbmF3cy5jb21cL2V1LXdlc3QtM19HWnFkaXRQaDYiLCJjb2duaXRvOnVzZXJuYW1lIjoicXVlbnRpbiIsImNvZ25pdG86cm9sZXMiOlsiYXJuOmF3czppYW06OjQxNTM4NjE0MjYwMjpyb2xlXC9lcGl0ZWNoLWNybS1BcnRpY2xlRnVuY3Rpb25Sb2xlLUNaMEhTNlgzQTdJWiJdLCJhdWQiOiIxdjZ1bGZoaTcwNWwzbWxmZ3Y0bzdodGRnYiIsInRva2VuX3VzZSI6ImlkIiwiYXV0aF90aW1lIjoxNjcxMDc0OTE5LCJleHAiOjE2NzEwNzg1MTksImlhdCI6MTY3MTA3NDkxOSwianRpIjoiOTdjMjJhYmEtMzM4MS00M2Y5LTgyYjAtYWVkY2U4ODE0MDdlIiwiZW1haWwiOiJxdWVudGluLnZlZHJlbm5lQG91dGxvb2suZnIifQ.ENWBt9JyPLjMhqOzpMAEV8f7jUzo3Myu0L40X2CWd9OBeHCPYmKGG-8mvhtuXIGZuDGy_K5afo2v4GrPW6hp_0Zs3Uqt8bEvVYQY3w5Xn9j2PbEuu_z1kIOvzAQ-IzY5ieomdL1wr3dyfwRLZohM0DkpbbXNbDd4oOjedrugya3vBgueuF3z6yEboONe_OFJEOA_-zM_ERu06m8zOVWONISb920GeAakg48kcyp6_wB4twQaxRqCnz4_jG6yESr94Zr60tD_wTYMaPf5177IyS8Th9mv_IxkO3j8GXuOAcXS1-cTqCOUyJyvvQZWnWOvSjKjucHqZQ9EbgfQS9lucQ
// &access_token=eyJraWQiOiJaenRsblFWemxndStPeG5DNDhBcVJlelhZdWQ5ZVp3VjhjQ2FVTThuRHQwPSIsImFsZyI6IlJTMjU2In0.eyJzdWIiOiIzYTU5NTU1NS03YzQ0LTRmNjUtYWZmMS00NDgwZWIzNjMxYjciLCJjb2duaXRvOmdyb3VwcyI6WyJBZG1pbiJdLCJpc3MiOiJodHRwczpcL1wvY29nbml0by1pZHAuZXUtd2VzdC0zLmFtYXpvbmF3cy5jb21cL2V1LXdlc3QtM19HWnFkaXRQaDYiLCJ2ZXJzaW9uIjoyLCJjbGllbnRfaWQiOiIxdjZ1bGZoaTcwNWwzbWxmZ3Y0bzdodGRnYiIsInRva2VuX3VzZSI6ImFjY2VzcyIsInNjb3BlIjoiYXdzLmNvZ25pdG8uc2lnbmluLnVzZXIuYWRtaW4gcGhvbmUgb3BlbmlkIHByb2ZpbGUgZW1haWwiLCJhdXRoX3RpbWUiOjE2NzEwNzQ5MTksImV4cCI6MTY3MTA3ODUxOSwiaWF0IjoxNjcxMDc0OTE5LCJqdGkiOiJjMTE0MzcxZi03MDYzLTQxMmQtOGQ3YS01ODk5YWFjYmJlMGQiLCJ1c2VybmFtZSI6InF1ZW50aW4ifQ.lVnw4sEC3TvhwSshnEZYyKDxwqiFnJhOZWcMDjj1JFwBd3zO6eJfvFLdcMnMkdaX6mLN9ZcL9neHYUqbDAvrxJrdTequEFRLRqdA6ixewtMHmlw3GTAO7VserMn1VVDXYhMkiRMjJm2MMhIiPFaF-SNGRlXoMuFZjUk3PkdK2n1dSEay5PzU1Du38LOQ2pV1trq0v7zy8dI-SD3yFVAAnqV4URvq0rIICGh5x6MVboff18LP5NqYCPqwGtJ58LaW4IvnT05vTctBLF92HKFEqbkM5RFNVwwbqeo44XS2OqYRzxDUZR_VKSSpsz3I_NHlZOovxHY8Jv6ceyemC55rAA
// &expires_in=3600&token_type=Bearer
