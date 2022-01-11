//+build !test

package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/forstmeier/askpaulgraham/pkg/cnt"
	"github.com/forstmeier/askpaulgraham/pkg/db"
	"github.com/forstmeier/askpaulgraham/pkg/nlp"
	"github.com/forstmeier/askpaulgraham/util"
)

const (
	summaryFilename   = "etc/data/summary.json"
	summariesFilename = "etc/data/summaries.json"
)

const (
	getAction  = "get"
	setAction  = "set"
	singleSize = "single"
	bulkSize   = "bulk"
)

type summariesJSON struct {
	Items []summaryJSON `json:"items"`
}

type summaryJSON struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Number  int    `json:"number"`
}

// The summaries CLI is used to generate and upload essay summaries
// to be presented via the DynamoDB table.
func main() {
	ctx := context.Background()

	action := flag.String("action", "get", `action to perform ("get" or "set")`)
	size := flag.String("size", "single", `size of the action ("single" or "bulk")`)
	postID := flag.String("id", "", "blog post id")

	flag.Parse()

	if *action != getAction && *action != setAction {
		log.Fatalf("error invalid action: %s", *action)
	}

	if *size != singleSize && *size != bulkSize {
		log.Fatalf("error invalid size: %s", *size)
	}

	if *size == singleSize && *action == getAction && *postID == "" {
		log.Fatal("error invalid arguments: argument 'id' is required for 'single' 'get' operation")
	}

	config := util.Config{}
	configContent, err := os.ReadFile("etc/config/config.json")
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}
	if err := json.Unmarshal(configContent, &config); err != nil {
		log.Fatalf("error unmarshalling config file: %v", err)
	}

	newSession, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})
	if err != nil {
		log.Fatalf("error creating aws session: %v", err)
	}

	cntClient := cnt.New()
	nlpClient := nlp.New(
		newSession,
		config.OpenAI.APIKey,
		config.AWS.S3.DataBucketName,
	)
	dbClient := db.New(
		newSession,
		config.AWS.S3.DataBucketName,
		config.AWS.DynamoDB.QuestionsTableName,
		config.AWS.DynamoDB.SummariesTableName,
	)

	if *action == getAction {
		items, err := cntClient.GetItems(ctx, "http://www.aaronsw.com/2002/feeds/pgessays.rss")
		if err != nil {
			log.Fatalf("error getting items: %v", err)
		}

		summaries := []summaryJSON{}
		for _, item := range items {
			if strings.Contains(item.Link, "1638975042") {
				continue
			}

			if (*size == bulkSize) || (*size == singleSize && strings.Contains(item.Link, "/"+*postID+".html")) {
				text, err := cntClient.GetText(ctx, item.Link)
				if err != nil {
					log.Fatalf("error getting text: %v", err)
				}

				summary, err := nlpClient.GetSummary(ctx, *text)
				if err != nil {
					log.Fatalf("error getting summary: %v", err)
				}

				id := util.GetIDFromURL(item.Link)
				summaries = append(summaries, summaryJSON{
					ID:      id,
					URL:     item.Link,
					Title:   item.Title,
					Summary: *summary,
					Number:  item.Number,
				})
			}
		}

		summariesBytes, err := json.Marshal(summariesJSON{
			Items: summaries,
		})

		filename := ""
		if *size == singleSize {
			filename = summaryFilename
		} else if *size == bulkSize {
			filename = summariesFilename
		}

		if err := os.WriteFile(filename, summariesBytes, 0644); err != nil {
			log.Fatalf("error writing summaries file: %v", err)
		}

	} else if *action == setAction {
		filename := ""
		if *size == singleSize {
			filename = summaryFilename
		} else if *size == bulkSize {
			filename = summariesFilename
		}

		summariesBytes, err := os.ReadFile(filename)
		if err != nil {
			log.Fatalf("error reading summaries file: %v", err)
		}

		summaries := summariesJSON{}
		if err := json.Unmarshal(summariesBytes, &summaries); err != nil {
			log.Fatalf("error unmarshalling summaries file: %v", err)
		}

		summariesData := []db.Summary{}
		for _, item := range summaries.Items {
			summariesData = append(summariesData, db.Summary{
				ID:      item.ID,
				URL:     item.URL,
				Title:   item.Title,
				Summary: item.Summary,
				Number:  item.Number,
			})
		}
		if err := dbClient.StoreSummaries(ctx, summariesData); err != nil {
			log.Fatalf("error batch writing summaries: %v", err)
		}
	}
}
