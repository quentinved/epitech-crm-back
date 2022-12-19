package main

import (
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/golang-jwt/jwt"
)

func parseGroup(token string) string {
	token2, err := jwt.Parse(token, func(token2 *jwt.Token) (interface{}, error) {
		return token2, nil
	})
	if err != nil {
		// TODO: Convert to []byte to fix error
		fmt.Println(err)
	}

	if elem, find := token2.Claims.(jwt.MapClaims)["cognito:groups"]; find {
		for _, v := range elem.([]interface{}) {
			if v == "User" {
				return "user"
			}
			if v == "Admin" {
				return "admin"
			}
		}
	}
	return "error"
}

func checkGroupCognito(token string, roleExpected string) events.APIGatewayProxyResponse {
	// TODO: check if token is valid
	role := parseGroup(token)
	if role != roleExpected {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Body:       string("Unauthorized"),
		}
	}
	return events.APIGatewayProxyResponse{
		StatusCode: 0,
	}
}
