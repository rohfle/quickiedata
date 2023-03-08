package types

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
	Variables map[string]interface{}
	Offset    int64
	Limit     int64
	Timeout   int64
}

func NewGetSPARQLQueryOptions() *GetSPARQLQueryOptions {
	return &GetSPARQLQueryOptions{
		Variables: make(map[string]interface{}),
		Offset:    -1,
		Limit:     -1,
		Timeout:   -1,
	}
}
