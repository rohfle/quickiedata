package types

type SPARQLResults struct {
	Head struct {
		Vars []string
	}
	Results struct {
		Bindings []map[string]*SPARQLBindingValue
	}
}

type SPARQLBindingValue struct {
	Value    *string `json:"value"`
	Type     string  `json:"type"`
	DataType string  `json:"datatype,omitempty"`
	Lang     string  `json:"xml:lang,omitempty"`
}

type SimpleSPARQLResults struct {
	Results []map[string]interface{}
}
