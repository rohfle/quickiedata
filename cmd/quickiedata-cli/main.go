package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rohfle/quickiedata"
)

func main() {
	wd := quickiedata.NewWikidataClient(&quickiedata.HTTPClientSettings{
		UserAgent:       "quickiedata",
		RequestInterval: 1 * time.Second,
		Backoff:         1 * time.Second,
		MaxBackoff:      30 * time.Second,
		MaxRetries:      5,
		MaxConnsPerHost: 1,
	})
	mode := os.Args[1]

	if mode == "search" {
		query := os.Args[2]
		options := quickiedata.NewSearchEntitiesOptions()
		result, err := wd.SearchEntities(query, options)
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
		options.Languages = []string{"en"}
		options.Sitefilter = []string{"enwiki", "enwikiquote"}
		result, err := wd.GetEntities(wikidataIDs, options)
		if err != nil {
			fmt.Printf("Error while retrieving %s: %s\n", wikidataIDs, err)
			return
		}

		simpleResult := result.Simplify()

		if val := simpleResult.GetEntityAsItem(wikidataIDs[0]).GetClaim("P31").ValueAsString(); val != "" {
			fmt.Println(wikidataIDs[0], "instance of", val)
		}

		if val := simpleResult.GetEntityAsItem(wikidataIDs[0]).GetClaim("does not exist").ValueAsString(); val != "" {
			fmt.Println("should not be here", val)
		}

		for _, item := range simpleResult.GetEntityAsItem("does not exist").GetClaims("does not exist") {
			fmt.Println("should not be here", item)
		}

		data, err := json.Marshal(simpleResult) //, "", "  ")
		if err != nil {
			fmt.Printf("Error while marshaling %s: %s\n", wikidataIDs, err)
			return
		}
		fmt.Println(string(data))
	} else if mode == "sparql" {

		sq := `
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

		options := quickiedata.NewSPARQLQueryOptions()
		options.Variables["instanceOf"] = quickiedata.WikidataID("wd:" + os.Args[2])
		options.Variables["memeIDs"] = []string{"222", "979"}
		options.Variables["foo"] = "bar"
		options.Offset = 0
		options.Limit = 10
		sdata, err := wd.SPARQLQueryAsSimple(sq, options)
		if err != nil {
			fmt.Println(err)
		}
		sraw, _ := json.MarshalIndent(sdata, "", "  ")
		fmt.Println(string(sraw))
	} else {
		fmt.Println("Usage: main.go [sparql|getentities|search] id")
	}

}
