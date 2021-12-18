package util

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"github.com/forstmeier/askpaulgraham/pkg/db"
)

// Log provides a basic wrapper to format log output.
func Log(key string, value interface{}) {
	logMessage(key, value)
}

// SendResponse is a helper function for sendin g responses
// to API Gateway.
func SendResponse(statusCode int, payload interface{}, message string) (events.APIGatewayProxyResponse, error) {
	var body interface{}

	switch payloadValue := payload.(type) {
	case error:
		body = struct {
			Error string `json:"error"`
		}{
			Error: payload.(error).Error(),
		}

	case []db.Data:
		body = struct {
			Message   string    `json:"message"`
			Summaries []db.Data `json:"summaries"`
		}{
			Message:   "success",
			Summaries: payloadValue,
		}

	case string:
		body = struct {
			Message string `json:"message"`
			Answer  string `json:"answer"`
		}{
			Message: "success",
			Answer:  payloadValue,
		}

	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		logMessage("MARSHAL_RESPONSE_PAYLOAD_ERROR", err.Error())
		return events.APIGatewayProxyResponse{
			StatusCode:      http.StatusInternalServerError,
			Body:            fmt.Sprintf(`{"error": %q}`, err),
			IsBase64Encoded: false,
		}, err
	}

	logMessage("RESPONSE_BODY", string(bodyBytes))
	return events.APIGatewayProxyResponse{
		StatusCode:      statusCode,
		Body:            string(bodyBytes),
		IsBase64Encoded: false,
	}, nil
}

func logMessage(key string, value interface{}) {
	log.Printf(`{"%s": "%+v"}`, key, value)
}

// GetIDFromURL returns the ID from the essay URL path.
func GetIDFromURL(url string) string {
	_, file := path.Split(url)
	id := strings.Replace(file, ".html", "", -1)
	return id
}
