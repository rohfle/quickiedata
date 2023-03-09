package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rohfle/quickiedata"
)

func main() {
	wd := quickiedata.NewWikidataService()
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
		wikidataID := os.Args[2]
		options := quickiedata.NewGetEntitiesOptions()
		options.Languages = []string{"en"}
		options.Sitefilter = []string{"enwiki", "enwikibooks"}
		result, err := wd.GetEntities([]string{wikidataID}, options)
		if err != nil {
			fmt.Printf("Error while retrieving %s: %s\n", wikidataID, err)
			return
		}

		simpleResult := quickiedata.SimplifyEntities(result)

		data, err := json.MarshalIndent(simpleResult, "", "  ")
		if err != nil {
			fmt.Printf("Error while marshaling %s: %s\n", wikidataID, err)
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
