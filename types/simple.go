package types

type SimpleProperty struct {
	DataType     DataType                  `json:"type"`
	Labels       map[string]string         `json:"labels,omitempty"`
	Descriptions map[string]string         `json:"descriptions,omitempty"`
	Aliases      map[string][]string       `json:"aliases,omitempty"`
	Claims       map[string][]*SimpleClaim `json:"claims,omitempty"`
}

type SimpleItem struct {
	Labels       map[string]string         `json:"labels,omitempty"`
	Descriptions map[string]string         `json:"descriptions,omitempty"`
	Aliases      map[string][]string       `json:"aliases,omitempty"`
	Claims       map[string][]*SimpleClaim `json:"claims,omitempty"`
	Sitelinks    map[string]string         `json:"sitelinks,omitempty"`
}

type SimpleLexeme struct {
	DataType        DataType          `json:"type,omitempty"`
	LexicalCategory string            `json:"category,omitempty"`
	Language        string            `json:"language,omitempty"`
	Lemmas          map[string]string `json:"lemmas,omitempty"`
	Forms           []*SimpleForm     `json:"forms,omitempty"`
	Senses          []*SimpleSense    `json:"senses,omitempty"`
}

type SimpleForm struct {
	Representations     map[string]string         `json:"representations,omitempty"`
	GrammaticalFeatures []string                  `json:"features,omitempty"`
	Claims              map[string][]*SimpleClaim `json:"claims,omitempty"`
}

type SimpleSense struct {
	Glosses map[string]string         `json:"glosses,omitempty"`
	Claims  map[string][]*SimpleClaim `json:"claims,omitempty"`
}

type SimpleClaim struct {
	Type       DataType                `json:"type,omitempty"`
	Value      interface{}             `json:"value,omitempty"`
	Qualifiers map[string][]*SnakValue `json:"qualifiers,omitempty"`
}
