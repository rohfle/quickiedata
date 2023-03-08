package types

type SearchResult struct {
	Aliases     []string `json:"aliases,omitempty"`
	ConceptURI  string   `json:"concepturi"`
	Description string   `json:"description"`
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Match       struct {
		Language string `json:"language"`
		Text     string `json:"text"`
		Type     string `json:"type"`
	} `json:"match"`
	PageID     int64  `json:"pageid"`
	Repository string `json:"repository"`
	Title      string `json:"title"`
	URL        string `json:"url"`
}

type SearchResponse struct {
	Search         []*SearchResult `json:"search"`
	SearchContinue int64           `json:"search-continue"`
	SearchInfo     struct {
		Search string `json:"search"`
	} `json:"searchinfo"`
	Success  int64          `json:"success"`
	Error    *ResponseError `json:"error"`
	ServedBy string         `json:"servedby"`
}
