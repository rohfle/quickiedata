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

// GetSPARQLQueryOptions carries the options of a SPARQL query. No option is
// defined yet; per-request timeouts come from the context.
type GetSPARQLQueryOptions struct{}

func NewSPARQLQueryOptions() *GetSPARQLQueryOptions {
	return &GetSPARQLQueryOptions{}
}

type SPARQLQuery struct {
	Template  string
	Variables map[string]any
	Offset    int64
	Limit     int64
}

func NewSPARQLQuery() *SPARQLQuery {
	return &SPARQLQuery{
		Variables: make(map[string]any),
		Offset:    -1,
		Limit:     -1,
	}
}
