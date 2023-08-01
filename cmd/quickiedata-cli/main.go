package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rohfle/quickiedata"
)

func main() {
	wd := quickiedata.NewWikidataClient(&quickiedata.HTTPClientSettings{
		DefaultHeaders: http.Header{
			"User-Agent": {"quickiedata"},
		},
		RequestInterval: 1 * time.Second,
		Backoff:         1 * time.Second,
		MaxBackoff:      30 * time.Second,
		MaxRetries:      5,
		MaxConnsPerHost: 1,
	})
	mode := os.Args[1]

	ctx := context.Background()

	if mode == "search" {
		query := os.Args[2]
		options := quickiedata.NewSearchEntitiesOptions()
		result, err := wd.SearchEntities(ctx, query, options)
		if err != nil {
			fmt.Printf("Error while searching for %s: %s\n", query, err)
			return
		}

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Printf("Error while marshaling %s: %s\n", query, err)
			return
		}
		fmt.Println(string(data))

	} else if mode == "getentities" {
		wikidataIDs := os.Args[2:]
		options := quickiedata.NewGetEntitiesOptions()
		// options.Languages = []string{"en"}
		// options.Sitefilter = []string{"enwiki", "enwikiquote"}
		result, err := wd.GetEntities(ctx, wikidataIDs, options)
		if err != nil {
			fmt.Printf("Error while retrieving %s: %s\n", wikidataIDs, err)
			return
		}

		simpleResult := result //.Simplify()

		data, err := json.MarshalIndent(simpleResult, "", "  ")
		if err != nil {
			fmt.Printf("Error while marshaling %s: %s\n", wikidataIDs, err)
			return
		}
		fmt.Println(string(data))
	} else if mode == "getentitiessimple" {
		wikidataIDs := os.Args[2:]
		options := quickiedata.NewGetEntitiesOptions()
		// options.Languages = []string{"en"}
		// options.Sitefilter = []string{"enwiki", "enwikiquote"}
		result, err := wd.GetEntities(ctx, wikidataIDs, options)
		if err != nil {
			fmt.Printf("Error while retrieving %s: %s\n", wikidataIDs, err)
			return
		}

		simpleResult := result.Simplify()

		data, err := json.MarshalIndent(simpleResult, "", "  ")
		if err != nil {
			fmt.Printf("Error while marshaling %s: %s\n", wikidataIDs, err)
			return
		}
		fmt.Println(string(data))
	} else if mode == "sparql" {

		queryText := `
		#Cats
		SELECT DISTINCT ?item ?itemLabel ?instanceOf ?memeID
		WHERE
		{
			{
			SELECT DISTINCT ?a ?b ?c ?d
			WHERE
			{
			  ?item wdt:P31 ?instanceOf. # Must be of a cat
			  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". } # Helps get the label in your language, if not, then en language
			} LIMIT 10
			}
		  ?item wdt:P31 ?instanceOf. # Must be of a cat
		  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". } # Helps get the label in your language, if not, then en language
		}
		`

		query := quickiedata.NewSPARQLQuery()
		query.Template = queryText
		query.Variables["instanceOf"] = quickiedata.WikidataID("wd:" + os.Args[2])
		query.Variables["memeIDs"] = []string{"222", "979"}
		query.Variables["foo"] = "bar"
		query.Offset = 0
		query.Limit = 10
		options := quickiedata.NewSPARQLQueryOptions()
		sdata, err := wd.SPARQLQuerySimple(ctx, query, options)
		if err != nil {
			fmt.Println(err)
		}
		sraw, _ := json.MarshalIndent(sdata.Results, "", "  ")
		fmt.Println(string(sraw))
	} else {
		fmt.Println("Usage: main.go [sparql|getentities|search] id")
	}

}
