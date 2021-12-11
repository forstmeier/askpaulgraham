package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"

	"github.com/forstmeier/askpaulgraham/pkg/db"
	"github.com/forstmeier/askpaulgraham/pkg/nlp"
	"github.com/forstmeier/askpaulgraham/util"
)

type requestPayload struct {
	Question string `json:"question"`
}

func handler(dbClient db.Databaser, nlpClient nlp.NLPer) func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		util.Log("REQUEST", request)

		switch request.HTTPMethod {
		case "GET":
			data, err := dbClient.GetData(ctx)
			if err != nil {
				return util.SendResponse(
					http.StatusInternalServerError,
					err,
					"GET_DATA_ERROR",
				)
			}

			return util.SendResponse(
				http.StatusOK,
				data,
				"RESPONSE_BODY",
			)

		case "POST":
			payload := requestPayload{}
			if err := json.Unmarshal([]byte(request.Body), &payload); err != nil {
				return util.SendResponse(
					http.StatusBadRequest,
					err,
					"UNMARSHAL_BODY_ERROR",
				)
			}

			if err := dbClient.StoreQuestion(ctx, payload.Question); err != nil {
				return util.SendResponse(
					http.StatusInternalServerError,
					err,
					"STORE_QUESTION_ERROR",
				)
			}

			answers, err := nlpClient.GetAnswers(ctx, payload.Question)
			if err != nil {
				return util.SendResponse(
					http.StatusInternalServerError,
					err,
					"GET_ANSWERS_ERROR",
				)
			}

			return util.SendResponse(
				http.StatusOK,
				answers[0],
				"RESPONSE_BODY",
			)

		default:
			return util.SendResponse(
				http.StatusMethodNotAllowed,
				fmt.Errorf("method '%s' not allowed", request.HTTPMethod),
				"METHOD_NOT_ALLOWED_ERROR",
			)
		}
	}
}
