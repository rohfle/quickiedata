package quickiedata_test

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/rohfle/quickiedata"
)

// TestSPARQLCompatibilityIssues checks which constructs are reported, and that
// the same words mentioned in a comment or a string literal are not.
func TestSPARQLCompatibilityIssues(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string // constructs, in any order
	}{
		{
			name:  "label service",
			query: `SELECT ?item ?itemLabel WHERE { ?item wdt:P31 wd:Q515 . SERVICE wikibase:label { bd:serviceParam wikibase:language "en". } }`,
			want:  []string{"wikibase:label"},
		},
		{
			name:  "mwapi service",
			query: `SELECT ?item WHERE { SERVICE wikibase:mwapi { bd:serviceParam wikibase:api "EntitySearch". } }`,
			want:  []string{"wikibase:mwapi"},
		},
		{
			name:  "geospatial services",
			query: `SELECT ?p WHERE { SERVICE wikibase:around { ?p bd:serviceParam wikibase:center "Point(1 2)"^^geo:wktLiteral. } }`,
			want:  []string{"wikibase:around or wikibase:box"},
		},
		{
			name:  "geof distance",
			query: `SELECT ?p WHERE { ?p wdt:P625 ?c . BIND(geof:distance(?c, ?c) AS ?d) }`,
			want:  []string{"geof:distance"},
		},
		{
			name:  "slice and sample",
			query: `SELECT ?x WHERE { ?x wdt:P31 wd:Q515 . SERVICE bd:slice { ?x ?p ?o . bd:slice.offset 0 . bd:slice.limit 10 . } SERVICE bd:sample { ?x ?p ?o . bd:sample.limit 5 } }`,
			want:  []string{"bd:slice", "bd:sample"},
		},
		{
			name:  "query hint",
			query: `SELECT ?x WHERE { hint:Prior hint:runFirst true . ?x wdt:P31 wd:Q515 }`,
			want:  []string{"query hint"},
		},
		{
			name:  "named subquery",
			query: `SELECT ?x WHERE { INCLUDE %cities } WITH %cities AS { ?x wdt:P31 wd:Q515 }`,
			want:  []string{"named subquery"},
		},
		{
			name:  "removed functions",
			query: `SELECT ?x ?y WHERE { ?x wdt:P856 ?u . BIND(wikibase:decodeURI(?u) AS ?y) FILTER(wikibase:isSomeValue(?u)) }`,
			want:  []string{"wikibase:decodeURI", "wikibase:isSomeValue"},
		},
		{
			name:  "gas service",
			query: `SELECT ?x WHERE { SERVICE gas:service { gas:program gas:gasClass "com.bigdata.rdf.graph.analytics.BFS" } }`,
			want:  []string{"GAS service"},
		},
		{
			name:  "date without a time part",
			query: `SELECT ?x WHERE { ?x wdt:P569 ?d . FILTER(?d >= "2001-08-25"^^xsd:dateTime) }`,
			want:  []string{"date without a time part"},
		},
		{
			name:  "duration instead of dayTimeDuration",
			query: `SELECT ?x WHERE { BIND("P60D"^^xsd:duration AS ?d) }`,
			want:  []string{"xsd:duration"},
		},
		{
			name:  "repeated variable binding",
			query: `SELECT ?item (SAMPLE(?label) AS ?label) WHERE { ?item rdfs:label ?l } GROUP BY ?item`,
			want:  []string{"repeated variable binding"},
		},
		{
			name:  "the same words in a comment",
			query: "SELECT ?x WHERE { ?x wdt:P31 wd:Q515 } # SERVICE wikibase:label, bd:slice, hint:Prior",
		},
		{
			name:  "the same words in a string literal",
			query: `SELECT ?x WHERE { BIND("SERVICE wikibase:label { bd:sample }" AS ?x) }`,
		},
		{
			name:  "an IRI carrying the type name",
			query: `SELECT ?x WHERE { ?x <http://www.wikidata.org/prop/direct/P31> wd:Q515 }`,
		},
		{
			name:  "a valid dateTime literal",
			query: `SELECT ?x WHERE { ?x wdt:P569 ?d . FILTER(?d >= "2001-08-25T00:00:00"^^xsd:dateTime) }`,
		},
		{
			name:  "a dayTimeDuration",
			query: `SELECT ?x WHERE { BIND("P60D"^^xsd:dayTimeDuration AS ?d) }`,
		},
		{
			name:  "an aggregate binding a new name",
			query: `SELECT ?item (SAMPLE(?label) AS ?oneLabel) WHERE { ?item rdfs:label ?label } GROUP BY ?item`,
		},
		{
			name:  "a plain label triple",
			query: `SELECT ?label WHERE { wd:Q42 rdfs:label ?label . FILTER(LANG(?label) = "en") }`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues := quickiedata.SPARQLCompatibilityIssues(test.query)

			var got []string
			for _, issue := range issues {
				if issue.Message == "" || issue.DocURL == "" {
					t.Errorf("%s: issue %+v has no message or doc link", issue.Construct, issue)
				}
				got = append(got, issue.Construct)
			}

			if !slices.Equal(sorted(got), sorted(test.want)) {
				t.Errorf("constructs = %v, want %v", sorted(got), sorted(test.want))
			}
		})
	}
}

func sorted(values []string) []string {
	out := slices.Clone(values)
	slices.Sort(out)
	return out
}

// TestSPARQLQueryWarnsAboutCompatibility checks that sending a query does not
// silently ignore a construct WDQS v2 will reject: the warning goes to the
// debug log, and the query is still sent, since the endpoint in use decides.
func TestSPARQLQueryWarnsAboutCompatibility(t *testing.T) {
	server, seen := newUAServer(t)
	client := newClientForServer(t, testUserAgent, server)

	var logged bytes.Buffer
	previous := quickiedata.DebugLog.Writer()
	quickiedata.DebugLog.SetOutput(&logged)
	t.Cleanup(func() { quickiedata.DebugLog.SetOutput(previous) })

	query := quickiedata.NewSPARQLQuery()
	query.Template = `SELECT ?item ?itemLabel WHERE { ?item wdt:P31 wd:Q515 . SERVICE wikibase:label { bd:serviceParam wikibase:language "en". } }`

	if _, err := client.SPARQLQueryRaw(context.Background(), query, nil); err != nil {
		t.Fatalf("SPARQLQueryRaw: %s", err)
	}
	if requests := seen(); len(requests) != 1 {
		t.Fatalf("server saw %d requests, want 1", len(requests))
	}
	if !strings.Contains(logged.String(), "wikibase:label") {
		t.Errorf("debug log = %q, want it to name the incompatible construct", logged.String())
	}
}

// TestSPARQLEndpointConstants pins the two endpoints the migration turns on,
// since a typo here would silently send users to the wrong backend.
func TestSPARQLEndpointConstants(t *testing.T) {
	if got, want := quickiedata.SPARQLEndpointV1, "https://query.wikidata.org/sparql"; got != want {
		t.Errorf("SPARQLEndpointV1 = %q, want %q", got, want)
	}
	if got, want := quickiedata.SPARQLEndpointV2, "https://query-next.wikidata.org/sparql"; got != want {
		t.Errorf("SPARQLEndpointV2 = %q, want %q", got, want)
	}

	client, err := quickiedata.NewClient("quickiedata-test/0.0", nil)
	if err != nil {
		t.Fatalf("NewClient: %s", err)
	}
	if client.SPARQLEndpoint != quickiedata.SPARQLEndpointV1 {
		t.Errorf("default SPARQL endpoint = %q, want the v1 endpoint until v2 is generally available", client.SPARQLEndpoint)
	}
}
