# quickiedata

**`quickiedata`** is a library for querying [Wikidata](https://www.wikidata.org/) simply and ergonomically. Supports performing SPARQL queries, searching and getting entities.

Based on https://github.com/maxlath/wikibase-sdk

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
---

## Examples

### CLI

`quickiedata-cli` is a cli tool that provides search, get and SPARQL capabilities using the library

```bash
quickiedata-cli search "hubble" --limit 1
quickiedata-cli get Q1 --props labels,claims
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
```

### Search

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rohfle/nicehttp"
	"github.com/rohfle/quickiedata"
)

func main() {
	ctx := context.Background()
	client := quickiedata.NewClient(&nicehttp.Settings{
		DefaultHeaders: http.Header{
			"User-Agent": []string{"your-user-agent-here"},
		},
		RequestInterval: 1 * time.Second,
		Backoff:         1 * time.Second,
		MaxBackoff:      30 * time.Second,
		MaxTries:        5,
		MaxConnsPerHost: 1,
	})

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
	"net/http"
	"time"

	"github.com/rohfle/nicehttp"
	"github.com/rohfle/quickiedata"
)

func main() {
	ctx := context.Background()
	client := quickiedata.NewClient(&nicehttp.Settings{
		DefaultHeaders: http.Header{
			"User-Agent": []string{"your-user-agent-here"},
		},
		RequestInterval: 1 * time.Second,
		Backoff:         1 * time.Second,
		MaxBackoff:      30 * time.Second,
		MaxTries:        5,
		MaxConnsPerHost: 1,
	})

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
	"net/http"
	"strings"
	"time"

	"github.com/rohfle/nicehttp"
	"github.com/rohfle/quickiedata"
)

func main() {
	ctx := context.Background()
	client := quickiedata.NewClient(&nicehttp.Settings{
		DefaultHeaders: http.Header{
			"User-Agent": []string{"your-user-agent-here"},
		},
		RequestInterval: 1 * time.Second,
		Backoff:         1 * time.Second,
		MaxBackoff:      30 * time.Second,
		MaxTries:        5,
		MaxConnsPerHost: 1,
	})

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
		fmt.Printf("failed while rendering results for %q: %s\n", query, err)
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

## Installation

```bash
go install github.com/rohfle/quickiedata@latest
```

## License

MIT