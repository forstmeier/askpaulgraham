package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"

	"github.com/forstmeier/askpaulgraham/pkg/db"
	"github.com/forstmeier/askpaulgraham/pkg/nlp"
	"github.com/forstmeier/askpaulgraham/util"
)

type requestPayload struct {
	Question string `json:"question"`
	UserID   string `json:"user_id"`
}

func handler(dbClient db.Databaser, nlpClient nlp.NLPer, jwtSigningKey string) func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		util.Log("REQUEST", request)

		// NOTE: this will be added back in once the UI is completed
		// if err := util.ValidateToken(request.Headers["Token"], jwtSigningKey); err != nil {
		// 	return util.SendResponse(
		// 		http.StatusUnauthorized,
		// 		err,
		// 		"VALIDATE_TOKEN_ERROR",
		// 	)
		// }

		switch request.HTTPMethod {
		case "GET":
			summaries, err := dbClient.GetSummaries(ctx)
			if err != nil {
				return util.SendResponse(
					http.StatusInternalServerError,
					err,
					"GET_SUMMARIES_ERROR",
				)
			}

			return util.SendResponse(
				http.StatusOK,
				summaries,
				"SUCCESSFUL_GET_RESPONSE",
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

			id := uuid.NewString()

			if err := dbClient.StoreQuestion(ctx, id, payload.Question); err != nil {
				return util.SendResponse(
					http.StatusInternalServerError,
					err,
					"STORE_QUESTION_ERROR",
				)
			}

			answer, err := nlpClient.GetAnswer(ctx, payload.Question, payload.UserID)
			if err != nil {
				return util.SendResponse(
					http.StatusInternalServerError,
					err,
					"GET_ANSWERS_ERROR",
				)
			}

			if err := dbClient.StoreAnswer(ctx, id, *answer); err != nil {
				return util.SendResponse(
					http.StatusInternalServerError,
					err,
					"STORE_ANSWER_ERROR",
				)
			}

			return util.SendResponse(
				http.StatusOK,
				*answer,
				"SUCCESSFUL_POST_RESPONSE",
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
