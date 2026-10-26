//go:build live

package quickiedata_test

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/rohfle/quickiedata"
)

// TestLiveEndpointsAgree runs the same WDQS v2 compatible queries against both
// endpoints and compares their results, which is how a parsing or dialect
// difference between v1 and v2 shows up. It needs the endpoints in
// the environment and network access:
//
//	QUICKIEDATA_ENDPOINT_V1=https://query.wikidata.org/sparql \
//	QUICKIEDATA_ENDPOINT_V2=https://query-next.wikidata.org/sparql \
//	go test -tags live -run TestLiveEndpointsAgree -v .
func TestLiveEndpointsAgree(t *testing.T) {
	v1 := os.Getenv("QUICKIEDATA_ENDPOINT_V1")
	v2 := os.Getenv("QUICKIEDATA_ENDPOINT_V2")
	if v1 == "" || v2 == "" {
		t.Skip("set QUICKIEDATA_ENDPOINT_V1 and QUICKIEDATA_ENDPOINT_V2 to compare the endpoints")
	}

	queries := map[string]string{
		"date literal":      `SELECT ?dob WHERE { wd:Q42 wdt:P569 ?dob }`,
		"quantity literal":  `SELECT ?population WHERE { wd:Q42 wdt:P1082 ?population }`,
		"entity and string": `SELECT ?item ?sitelinkTitle WHERE { BIND(wd:Q42 AS ?item) OPTIONAL { ?item schema:description ?d . FILTER(LANG(?d) = "en") } BIND(STR(?item) AS ?sitelinkTitle) }`,
		"optional label":    `SELECT ?label WHERE { OPTIONAL { wd:Q42 rdfs:label ?label . FILTER(LANG(?label) = "en") } }`,
		"absent value":      `SELECT ?x ?label WHERE { VALUES ?x { wd:Q42 } OPTIONAL { ?x rdfs:label ?label . FILTER(LANG(?label) = "zz") } }`,
	}

	for name, text := range queries {
		t.Run(name, func(t *testing.T) {
			query := quickiedata.NewSPARQLQuery()
			query.Template = text

			gotV1 := runLiveQuery(t, v1, query)
			gotV2 := runLiveQuery(t, v2, query)

			if !slices.Equal(gotV1, gotV2) {
				t.Errorf("endpoints disagree\nv1 %q\nv2 %q", gotV1, gotV2)
			}
		})
	}
}

// runLiveQuery returns the rows of a query as sorted "variable=type:value"
// strings, so two engines can be compared without depending on row order or on
// the shape of the JSON.
func runLiveQuery(t *testing.T, endpoint string, query *quickiedata.SPARQLQuery) []string {
	t.Helper()

	client, err := quickiedata.NewClient("quickiedata-live-test/0.0 (https://github.com/rohfle/quickiedata)", nil)
	if err != nil {
		t.Fatalf("NewClient: %s", err)
	}
	client.SPARQLEndpoint = endpoint

	response, err := client.SPARQLQuery(context.Background(), query)
	if err != nil {
		t.Fatalf("%s: %s", endpoint, err)
	}

	rows := make([]string, 0, len(response.Results.Bindings))
	for _, binding := range response.Results.Bindings {
		keys := make([]string, 0, len(binding))
		for key := range binding {
			keys = append(keys, key)
		}
		slices.Sort(keys)

		row := ""
		for _, key := range keys {
			value := binding[key]
			text := "<nil>"
			if value.Value != nil {
				text = *value.Value
			}
			row += fmt.Sprintf("%s=%s:%s;", key, value.Type, text)
		}
		rows = append(rows, row)
	}
	slices.Sort(rows)

	return rows
}
