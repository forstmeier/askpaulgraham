package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"

	"github.com/forstmeier/askpaulgraham/pkg/cnt"
	"github.com/forstmeier/askpaulgraham/pkg/db"
	"github.com/forstmeier/askpaulgraham/pkg/nlp"
)

type samConfigTOML struct {
	DataBucket   string `toml:"data_bucket"`
	OpenAIAPIKey string `toml:"open_ai_api_key"`
}

func main() {
	ctx := context.Background()

	postID := flag.String("id", "", "blog post id")
	summaryCheck := flag.Bool("summary", false, "get and upload summary")
	answersCheck := flag.Bool("answers", false, "get and upload answers")

	flag.Parse()

	if *postID == "" {
		log.Fatal("error missing argument: argument 'id' is required")
	}

	if !*summaryCheck && !*answersCheck {
		log.Fatal("error missing argument: either argument 'summary' or 'answers' is required")
	}

	config := samConfigTOML{}
	configContent, err := ioutil.ReadFile("samconfig.toml")
	if err := toml.Unmarshal(configContent, &config); err != nil {
		log.Fatalf("error reading config file: %v", err)
	}

	newSession, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})
	if err != nil {
		log.Fatalf("error creating aws session: %v", err)
	}

	dynamoDBClient := dynamodb.New(newSession)

	summariesTableName := ""
	tables, err := dynamoDBClient.ListTables(&dynamodb.ListTablesInput{})
	for _, tableName := range tables.TableNames {
		if strings.Contains(*tableName, "summariesTable") {
			summariesTableName = *tableName
		}
	}

	cntClient := cnt.New()
	dbClient := db.New(newSession, config.DataBucket, "", summariesTableName)
	nlpClient := nlp.New(newSession, config.OpenAIAPIKey, config.DataBucket)

	postURL := fmt.Sprintf("http://www.paulgraham.com/%s.html", *postID)

	items, err := cntClient.GetItems(ctx, "http://www.aaronsw.com/2002/feeds/pgessays.rss")
	if err != nil {
		log.Fatalf("error getting items: %v", err)
	}

	postTitle := ""
	for _, item := range items {
		if item.Link == postURL {
			postTitle = item.Title
		}
	}

	text, err := cntClient.GetText(ctx, postURL)
	if err != nil {
		log.Fatalf("error getting text: %v", err)
	}

	if summaryCheck != nil && *summaryCheck {
		summary, err := nlpClient.GetSummary(ctx, *text)
		if err != nil {
			log.Fatalf("error getting summary: %v", err)
		}

		if err := dbClient.StoreData(ctx, *postID, postURL, postTitle, *summary, *text); err != nil {
			log.Fatalf("error storing data: %v", err)
		}

		log.Printf("successfully added %q summary", *postID)

	} else if answersCheck != nil && *answersCheck {
		if err := nlpClient.SetAnswer(ctx, *postID, *text); err != nil {
			log.Fatalf("error setting answer: %v", err)
		}

		log.Printf("successfully added %q answer", *postID)

	}
}
