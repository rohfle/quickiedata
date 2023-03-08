package types

type GetEntitiesResponse struct {
	Entities map[string]*EntityInfo `json:"entities,omitempty"`
	Success  int64                  `json:"success"`
	Error    *ResponseError         `json:"error,omitempty"`
}

// This is a combined struct for all entity info types
// Use the Type field to work out which fields will have values
type EntityInfo struct {
	PageID    int64     `json:"pageid"`
	NS        int64     `json:"ns"`
	Title     string    `json:"title"`
	LastRevID int64     `json:"lastrevid"`
	Modified  string    `json:"modified"`
	Redirects *Redirect `json:"redirects,omitempty"`
	Type      string    `json:"type"`

	// Item fields
	ID           string               `json:"id"`
	Labels       map[string]*Term     `json:"labels"`
	Descriptions map[string]*Term     `json:"descriptions"`
	Aliases      map[string][]*Term   `json:"aliases"`
	Claims       map[string][]*Claim  `json:"claims"`
	Sitelinks    map[string]*Sitelink `json:"sitelinks"`

	// Property fields
	// ID (defined above)
	DataType DataType `json:"datatype"`
	// Labels (defined above)
	// Descriptions (defined above)
	// Aliases (defined above)
	// Claims (defined above)

	// Lexeme fields
	// ID (defined above)
	// DataType (defined above)
	LexicalCategory string           `json:"lexical-category"`
	Language        string           `json:"language"`
	Lemmas          map[string]*Term `json:"lemmas"`
	Forms           []*Form          `json:"forms"`
	Senses          []*Sense         `json:"senses"`

	// Form fields
	// ID (defined above)
	Representations     map[string]*Term `json:"representations"`
	GrammaticalFeatures []string         `json:"grammatical-features"`
	// Claims (defined above)

	// Sense fields
	// ID (defined above)
	Glosses map[string]*Term `json:"glosses"`
	// Claims (defined above)
}
