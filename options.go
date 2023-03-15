package quickiedata

type GetEntitiesOptions struct {
	Languages  []string
	Sitefilter []string
	Props      []string
	Format     string
	Redirects  bool
}

func NewGetEntitiesOptions() *GetEntitiesOptions {
	return &GetEntitiesOptions{
		Languages:  []string{},
		Sitefilter: []string{},
		Props:      []string{},
		Format:     "json",
		Redirects:  true,
	}
}

type SearchEntitiesOptions struct {
	Language   string
	Limit      int64
	Offset     int64
	Format     string
	UseLang    string
	EntityType string
}

func NewSearchEntitiesOptions() *SearchEntitiesOptions {
	return &SearchEntitiesOptions{
		Language:   "en",
		UseLang:    "",
		Limit:      20,
		Offset:     0,
		Format:     "json",
		EntityType: "item",
	}
}

type GetSPARQLQueryOptions struct {
	Timeout int64
}

func NewSPARQLQueryOptions() *GetSPARQLQueryOptions {
	return &GetSPARQLQueryOptions{
		Timeout: -1,
	}
}

type SPARQLQuery struct {
	Template  string
	Variables map[string]interface{}
	Offset    int64
	Limit     int64
}

func NewSPARQLQuery() *SPARQLQuery {
	return &SPARQLQuery{
		Variables: make(map[string]interface{}),
		Offset:    -1,
		Limit:     -1,
	}
}
