package main

import (
	"context"
	"strings"

	"github.com/forstmeier/askpaulgraham/pkg/cnt"
	"github.com/forstmeier/askpaulgraham/pkg/db"
	"github.com/forstmeier/askpaulgraham/pkg/nlp"
	"github.com/forstmeier/askpaulgraham/util"
)

func handler(cntClient cnt.Contenter, dbClient db.Databaser, nlpClient nlp.NLPer, rssURL string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		items, err := cntClient.GetItems(ctx, rssURL)
		if err != nil {
			util.Log("GET_ITEMS_ERROR", err)
			return err
		}

		oldIDs, err := dbClient.GetIDs(ctx)
		if err != nil {
			util.Log("GET_IDS_ERROR", err)
			return err
		}
		oldIDsMap := map[string]struct{}{}
		for _, oldID := range oldIDs {
			oldIDsMap[oldID] = struct{}{}
		}

		for _, item := range items {
			// ignore Common Lisp chapter posts
			if strings.Contains(item.Link, "1638975042") {
				continue
			}

			id := util.GetIDFromURL(item.Link)
			if _, ok := oldIDsMap[id]; !ok {
				text, err := cntClient.GetText(ctx, item.Link)
				if err != nil {
					util.Log("GET_TEXT_ERROR", err)
					return err
				}

				summary, err := nlpClient.GetSummary(ctx, *text)
				if err != nil {
					util.Log("GET_SUMMARY_ERROR", err)
					return err
				}

				if err := dbClient.StoreSummaries(ctx, []db.Summary{
					{
						ID:      id,
						URL:     item.Link,
						Title:   item.Title,
						Summary: *summary,
					},
				}); err != nil {
					util.Log("STORE_SUMMARY_ERROR", err)
					return err
				}

				if err := dbClient.StoreText(ctx, id, *text); err != nil {
					util.Log("STORE_TEXT_ERROR", err)
					return err
				}

				if err := nlpClient.SetAnswer(ctx, id, *text); err != nil {
					util.Log("SET_ANSWER_ERROR", err)
					return err
				}
			}
		}

		return nil
	}
}
