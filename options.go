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

// Variables holds the terms bound to a SPARQL variable: one term is a BIND, a
// list is a VALUES clause. Use Set and Add, and NewSPARQLQuery to initialise it.
// A variable's statement is inserted in the outer group pattern, and again
// inside each subquery that uses it; a subquery is its own scope, so a binding
// outside it does not reach it.
type Variables map[string][]SPARQLTerm

// Set replaces the terms bound to name.
func (v Variables) Set(name string, terms ...SPARQLTerm) {
	v[name] = terms
}

// Add appends terms to the ones bound to name.
func (v Variables) Add(name string, terms ...SPARQLTerm) {
	v[name] = append(v[name], terms...)
}

// Get returns the terms bound to name.
func (v Variables) Get(name string) []SPARQLTerm {
	return v[name]
}

type SPARQLQuery struct {
	Template  string
	Variables Variables

	// Offset is appended to the generated query as "OFFSET n" when not
	// negative. The template is not checked for an existing OFFSET clause.
	Offset int64
	// Limit is appended to the generated query as "LIMIT n" when positive, so
	// 0 means no limit and -1, the default, means unset. The template is not
	// checked for an existing LIMIT clause.
	Limit int64
}

func NewSPARQLQuery() *SPARQLQuery {
	return &SPARQLQuery{
		Variables: make(Variables),
		Offset:    -1,
		Limit:     -1,
	}
}
