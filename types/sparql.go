package types

type SPARQLResults struct {
	Head struct {
		Vars []string
	}
	Results struct {
		Bindings []map[string]*BindingValue
	}
}

type BindingValue struct {
	Value    *string `json:"value"`
	Type     string  `json:"type"`
	DataType string  `json:"datatype,omitempty"`
	Lang     string  `json:"xml:lang,omitempty"`
}

type SimpleSPARQLResults struct {
	Results []map[string]interface{}
}

/*
# Useful queries to keep around

# Retrieve a full list of units
SELECT DISTINCT ?item ?itemLabel ?unitLabel WHERE {
	?item wdt:P5061 ?unit
	FILTER(LANG(?unit) = "en")
	SERVICE wikibase:label {
		bd:serviceParam wikibase:language "en".
		?unit rdfs:label ?unitLabel.
		?item rdfs:label ?itemLabel.
	}
} ORDER BY ?unitLabel

# Retrieve a list of globes
SELECT ?globe ?globeLabel
WHERE
{
	{
		SELECT (count(?v) as ?c) ?globe
		WHERE { ?v wikibase:geoGlobe ?globe. }
		GROUP BY ?globe
	}
	FILTER (?c > 100)  # optional line to reduce size of results
	SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
*/
