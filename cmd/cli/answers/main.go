package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/forstmeier/askpaulgraham/pkg/cnt"
	"github.com/forstmeier/askpaulgraham/util"
)

const (
	answerFilename  = "answer.json"
	answersFilename = "answers.jsonl"
)

const (
	getAction  = "get"
	setAction  = "set"
	singleSize = "single"
	bulkSize   = "bulk"
)

type samConfigTOML struct {
	DataBucket   string `toml:"data_bucket"`
	OpenAIAPIKey string `toml:"open_ai_api_key"`
}

type answerJSON struct {
	Text     string `json:"text"`
	Metadata string `json:"metadata"`
}

func main() {
	ctx := context.Background()

	action := flag.String("action", "get", `action to perform ("get" or "set")`)
	size := flag.String("size", "single", `size of the action ("single" or "bulk")`)
	postID := flag.String("id", "", "blog post id")

	flag.Parse()

	if *action != getAction && *action != setAction {
		log.Fatalf("invalid action: %s", *action)
	}

	if *size != singleSize && *size != bulkSize {
		log.Fatalf("invalid size: %s", *size)
	}

	if *size == singleSize && *postID == "" {
		log.Fatal("error invalid arguments: argument 'id' is required for 'single' operation")
	}

	config := samConfigTOML{}
	configContent, err := ioutil.ReadFile("samconfig.toml")
	if err != nil {
		log.Fatalf("error reading config file: %v", err)
	}
	if err := toml.Unmarshal(configContent, &config); err != nil {
		log.Fatalf("error unmarshalling config file: %v", err)
	}

	newSession, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})
	if err != nil {
		log.Fatalf("error creating aws session: %v", err)
	}

	s3Client := s3.New(newSession)
	cntClient := cnt.New()

	if *action == getAction {
		if *size == singleSize {
			postURL := fmt.Sprintf("http://www.paulgraham.com/%s.html", *postID)
			text, err := cntClient.GetText(ctx, postURL)
			if err != nil {
				log.Fatalf("error getting text: %v", err)
			}

			bodyBytes, err := json.Marshal(answerJSON{
				Metadata: *postID,
				Text:     *text,
			})
			if err != nil {
				log.Fatalf("error marshalling answer: %v", err)
			}

			if err := os.WriteFile(answerFilename, bodyBytes, 0644); err != nil {
				log.Fatalf("error writing answer file: %v", err)
			}

		} else if *size == bulkSize {
			items, err := cntClient.GetItems(ctx, "http://www.aaronsw.com/2002/feeds/pgessays.rss")
			if err != nil {
				log.Fatalf("error getting items: %v", err)
			}

			answersBody := bytes.Buffer{}
			encoder := json.NewEncoder(&answersBody)
			for _, item := range items {
				if strings.Contains(item.Link, "1638975042") {
					continue
				}

				text, err := cntClient.GetText(ctx, item.Link)
				if err != nil {
					log.Fatalf("error getting text: %v", err)
				}

				id := util.GetIDFromURL(item.Link)
				if err := encoder.Encode(answerJSON{
					Text:     *text,
					Metadata: id,
				}); err != nil {
					log.Fatalf("error encoding answer: %v", err)
				}
			}

			if err := os.WriteFile(answersFilename, answersBody.Bytes(), 0644); err != nil {
				log.Fatalf("error writing answers file: %v", err)
			}
		}

	} else if *action == setAction {
		filename := ""
		if *size == singleSize {
			filename = answerFilename
		} else if *size == bulkSize {
			filename = answersFilename
		}

		bodyBytes, err := os.ReadFile(filename)
		if err != nil {
			log.Fatalf("error reading file: %v", err)
		}

		if *size == singleSize {
			newAnswer := answerJSON{}
			if err := json.Unmarshal(bodyBytes, &newAnswer); err != nil {
				log.Fatalf("error unmarshalling answer: %v", err)
			}

			getAnswersResp, err := s3Client.GetObject(&s3.GetObjectInput{
				Bucket: &config.DataBucket,
				Key:    aws.String(answersFilename),
			})
			if err != nil {
				log.Fatalf("error getting answers: %v", err)
			}

			answersBody := bytes.Buffer{}

			encoder := json.NewEncoder(&answersBody)
			if err := encoder.Encode(newAnswer); err != nil {
				log.Fatalf("error encoding new answer: %v", err)
			}
			decoder := json.NewDecoder(getAnswersResp.Body)

			for decoder.More() {
				var answer answerJSON
				if err := decoder.Decode(&answer); err == io.EOF {
					break
				} else if err != nil {
					log.Fatalf("error decoding answer: %v", err)
				}

				if newAnswer.Metadata != answer.Metadata {
					if err := encoder.Encode(answer); err != nil {
						log.Fatalf("error encoding old answer: %v", err)
					}
				}
			}

			bodyBytes = answersBody.Bytes()
		}

		_, err = s3Client.PutObject(&s3.PutObjectInput{
			Bucket: &config.DataBucket,
			Key:    aws.String(answersFilename),
			Body:   bytes.NewReader(bodyBytes),
		})
		if err != nil {
			log.Fatalf("error putting answers: %v", err)
		}
	}
}
