# quickiedata

**`quickiedata`** is a library written in go for querying [Wikidata](https://www.wikidata.org/) simply and ergonomically. Supports performing SPARQL queries, searching and getting entities.

---

## Installation

```bash
go get github.com/rohfle/quickiedata@latest
```

---

## Features

- Run SPARQL queries with variable support
- Search for items by term, with filters by language and entity type
- Get entities by id, with filters by language and props

- Offset and limit support
- Optional simplification of returned data structures
- Helper methods with return typed values from claims and snaks, or typed nil if the value is empty. This makes it possible to chain even with nil values. For example:
```go
if coord := simpleResult.GetEntityAsItem("Q2112").GetClaim("P625").ValueAsCoordinate(); coord != nil {
	fmt.Println("Latitude:", coord.Latitude)
}
```
- Wikimedia-compliant User-Agent, pacing and retries by default

---

## User-Agent and rate limits

Wikimedia asks every client to identify itself, in the form
`<client name>/<version> (<contact information>)`. That first argument of
`NewClient` is required, and the library appends its own
`quickiedata/<version>` token to it:

```go
client, err := quickiedata.NewClient("your-app/1.0 (you@example.org)", nil)
```

The transport built by default allows one request per second, retries up to
five times with exponential backoff, honours `Retry-After`, and bounds each
attempt to sixty seconds.

`WikidataClientOptions.DownstreamTransport` replaces only the connection layer
(sockets, timeouts, proxies), keeping the User-Agent, the pacing and the
retries. To take over the whole client, replace `client.Client.Transport` after
construction: the User-Agent is set on each request, so it still goes out.
`WikidataClientOptions.SuppressLibraryUA` drops the library token, leaving the
whole header to the caller. For policy details see the
[Wikimedia User-Agent policy](https://foundation.wikimedia.org/wiki/Policy:Wikimedia_Foundation_User-Agent_Policy).

---

## Examples

### CLI

`quickiedata-cli` is a cli tool that provides search, get and SPARQL capabilities using the library

```bash
quickiedata-cli search "hubble" --limit 1
quickiedata-cli get Q1 --props labels,claims
# read query from stdin
quickiedata-cli query name=Oscar <<EOF
SELECT ?item ?itemLabel
WHERE
{
  ?item wdt:P31 wd:Q146. # Must be a cat
  ?item rdfs:label ?itemLabel.
  FILTER(LANG(?itemLabel) = "en").
  FILTER(STR(?itemLabel) = ?name)
}
EOF
# or read query from file
quickiedata-cli query path/to/cats.sparql name=Oscar
# report SPARQL that the new WDQS backend does not support
quickiedata-cli check-query path/to/cats.sparql
```

### Search

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/rohfle/quickiedata"
)

func main() {
	ctx := context.Background()
	client, err := quickiedata.NewClient("your-app/1.0 (you@example.org)", nil)
	if err != nil {
		log.Fatal(err)
	}

	query := "hubble space telescope"
	options := quickiedata.NewSearchEntitiesOptions()
	options.Language = "en"
	options.EntityType = "item"
	options.Offset = 0
	options.Limit = 1
	results, err := client.SearchEntities(ctx, query, options)
	if err != nil {
		fmt.Printf("failed while searching for %q: %s\n", query, err)
		return
	}
	if len(results) == 0 {
		fmt.Println("no results")
		return
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Printf("failed while rendering results for %q: %s\n", query, err)
		return
	}
	fmt.Println(string(data))

	if len(results) > 0 {
		result := results[0]
		label := result.Label
		wdid := result.ID
		fmt.Printf("first result: %s (id=%s)\n", label, wdid)
	}
}
```
gives you as output
```json
[
  {
    "concepturi": "http://www.wikidata.org/entity/Q2513",
    "description": "NASA and ESA space telescope (launched 1990)",
    "id": "Q2513",
    "label": "Hubble Space Telescope",
    "match": {
      "language": "en",
      "text": "Hubble Space Telescope",
      "type": "label"
    },
    "pageid": 3480,
    "repository": "wikidata",
    "title": "Q2513",
    "url": "//www.wikidata.org/wiki/Q2513"
  }
]
```
```
first result: Hubble Space Telescope (id=Q2513)
```

### Get

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/rohfle/quickiedata"
)

func main() {
	ctx := context.Background()
	client, err := quickiedata.NewClient("your-app/1.0 (you@example.org)", nil)
	if err != nil {
		log.Fatal(err)
	}

	ids := []string{"Q2112"}
	options := quickiedata.NewGetEntitiesOptions()
	options.Languages = []string{"en"}
	options.Sitefilter = []string{"enwiki"}
	options.Props = []string{"labels", "claims"}

	result, err := client.GetEntities(ctx, ids, options)
	if err != nil {
		fmt.Printf("failed while retrieving %s: %s", ids, err)
		return
	}

	simpleResult := result.Simplify()
	data, err := json.MarshalIndent(simpleResult, "", "  ")
	if err != nil {
		fmt.Printf("failed while rendering results for %q: %s\n", ids, err)
		return
	}
	fmt.Println(string(data))

	item := simpleResult.GetEntityAsItem("Q2112")
	if item != nil {
		fmt.Printf("%s label: %s\n", "Q2112", item.Labels["en"])
	}
	// Examples of chaining
	// Claim exists and has coords
	if coord := simpleResult.GetEntityAsItem("Q2112").GetClaim("P625").ValueAsCoordinate(); coord != nil {
		fmt.Println("P625 Latitude:", coord.Latitude)
	}
	// Claim exists but does not have coords
	if coord := simpleResult.GetEntityAsItem("Q2112").GetClaim("P47").ValueAsCoordinate(); coord == nil {
		fmt.Println("P47 coords:", coord)
	}
	// Claim does not exist
	if coord := simpleResult.GetEntityAsItem("Q2112").GetClaim("P999999").ValueAsCoordinate(); coord == nil {
		fmt.Println("P999999 coords:", coord)
	}
}
```
gives you as output
```json
{
  "entities": {
    "Q1": {
      "labels": {
        "en": "Universe"
      },
      "claims": {
        //...
      },
      "type": "item"
    }
  }
}
```
```
Q2112 label: Bielefeld
P625 Latitude: 52.016666666667
P47 coords: <nil>
P999999 coords: <nil>
```

### SPARQL

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/rohfle/quickiedata"
)

func main() {
	ctx := context.Background()
	client, err := quickiedata.NewClient("your-app/1.0 (you@example.org)", nil)
	if err != nil {
		log.Fatal(err)
	}

	query := quickiedata.NewSPARQLQuery()
	query.Offset = 0
	query.Limit = 10
	query.Template = `
		#Cats
		SELECT ?item ?itemLabel
		WHERE
		{
			?item wdt:P31 wd:Q146. # Must be a cat
			?item rdfs:label ?itemLabel.
			FILTER(LANG(?itemLabel) = "en").
			FILTER(STR(?itemLabel) = ?name)
		}
	`
	query.Variables["name"] = "Oscar"

	options := quickiedata.NewSPARQLQueryOptions()
	resp, err := client.SPARQLQuerySimple(ctx, query, options)
	if err != nil {
		fmt.Printf("sparql request failed:\nquery:\n  %s\noptions: %+v\nerror: %s\n",
			strings.ReplaceAll(query.Template, "\n", "\n  "),
			options,
			err,
		)
		return
	}

	if len(resp.Results) == 0 {
		fmt.Println("no results")
		return
	}

	data, err := json.MarshalIndent(resp.Results, "", "  ")
	if err != nil {
		fmt.Printf("failed while rendering results for %+v: %s\n", query, err)
		return
	}
	fmt.Println(string(data))

	if len(resp.Results) > 0 {
		result := resp.Results[0]
		label := result["itemLabel"].ValueAsString()
		wdid := result["item"].ValueAsString()
		fmt.Printf("first result: %s (id=%s)\n", label, wdid)
	}
}
```
gives you as output
```json
[
  {
    "item": {
      "Value": "Q1185550"
    },
    "itemLabel": {
      "Value": "Oscar"
    }
  },
  {
    "item": {
      "Value": "Q7105840"
    },
    "itemLabel": {
      "Value": "Oscar"
    }
  }
]
```
```
first result: Oscar (id=Q1185550)
```

## Migrating to WDQS v2

The Wikidata Query Service is moving from v1 to v2. New endpoints are
available from November 2026, the v1 endpoint stays up but may
degrade from February 2027, and the migration also requires rewriting some
SPARQL. The entity and search APIs are not affected.

The two endpoints are exported as constants, and the client uses v1 until v2 is
generally available:

```go
client, err := quickiedata.NewClient("your-app/1.0 (you@example.org)", &quickiedata.WikidataClientOptions{
	SPARQLEndpoint: quickiedata.SPARQLEndpointV2,
})
```

`SPARQLCompatibilityIssues` reports the constructs v2 does not support, so a
query can be checked before it is sent anywhere. The check is textual: it looks
at the query with comments and string literals removed.

```go
for _, issue := range quickiedata.SPARQLCompatibilityIssues(queryText) {
	log.Printf("%s: %s (%s)", issue.Construct, issue.Message, issue.DocURL)
}
```

`quickiedata-cli check-query file.sparql` prints the same report and exits
non-zero when it finds something, which makes it usable in a pipeline.

| Construct | v2 |
| --- | --- |
| `SERVICE wikibase:label` | not available, use `rdfs:label` with `OPTIONAL` |
| `SERVICE wikibase:mwapi` | not available, call the API separately |
| `wikibase:around`, `wikibase:box`, `geof:distance` | rewrite with GeoSPARQL |
| `bd:slice`, `bd:sample` | rewrite with `ORDER BY` and `OFFSET`/`LIMIT` |
| `hint:*` | ignored |
| `%name` subqueries | not SPARQL 1.1, restructure |
| `wikibase:decodeURI`, `wikibase:isSomeValue` | not available |
| GAS service | use path search |
| `"2001-08-25"^^xsd:dateTime`, `^^xsd:duration` | needs a time part, use `dayTimeDuration` |
| `SAMPLE(?v) AS ?v` | a variable cannot be bound twice |

The label service is used by most queries. This:

```sparql
SELECT ?item ?itemLabel WHERE {
  ?item wdt:P31 wd:Q515 .
  SERVICE wikibase:label { bd:serviceParam wikibase:language "en,de". }
}
```

becomes plain SPARQL 1.1, which runs on both backends:

```sparql
SELECT ?item ?itemLabel WHERE {
  ?item wdt:P31 wd:Q515 .
  OPTIONAL { ?item rdfs:label ?itemLabel . FILTER(lang(?itemLabel) = "en") }
  OPTIONAL { ?item rdfs:label ?itemLabel . FILTER(lang(?itemLabel) = "de") }
  OPTIONAL { ?item rdfs:label ?itemLabel . FILTER(lang(?itemLabel) = "mul") }
}
```

Chained `OPTIONAL`s give language priority: all of them bind `?itemLabel`, so
once the first one matches, the next pattern has nowhere to bind. An entity with
no label in any listed language keeps its row with `?itemLabel` unbound, where a
mandatory `?item rdfs:label ?label` would drop the row instead.

Keep the `mul` line: some items, including Q42, carry their label only as `mul`.
Use `LANGMATCHES(lang(?itemLabel), "en")` instead of `=` if regional variants
such as `en-gb` should count. Aliases need `GROUP_CONCAT` because they are
multi-valued.

Dates keep their precision outside the value: a statement with year precision is
stored as `1984-01-01T00:00:00Z^^xsd:dateTime`, so filtering by a period takes a
range, not a year literal:

```sparql
?dob >= "1984-01-01T00:00:00Z"^^xsd:dateTime &&
?dob <  "1985-01-01T00:00:00Z"^^xsd:dateTime
```

The full rewriting guide is at
<https://www.wikidata.org/wiki/Wikidata:SPARQL_query_service/WDQS_backend_update/Migrate>.
A differential test against both endpoints is kept behind a build tag and is
skipped unless the endpoints are given:

```bash
QUICKIEDATA_ENDPOINT_V1=https://query.wikidata.org/sparql \
QUICKIEDATA_ENDPOINT_V2=https://query-next.wikidata.org/sparql \
go test -tags live -run TestLiveEndpointsAgree .
```

## License

MIT
