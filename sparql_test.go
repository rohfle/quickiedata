package quickiedata_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/rohfle/quickiedata"
)

// The fields own the paging clauses: set means appended, unset means absent.
// A template carrying its own OFFSET or LIMIT is the caller's error, and is
// deliberately not second-guessed here.
func TestRenderSPARQLQueryOffsetAndLimit(t *testing.T) {
	tests := []struct {
		name   string
		offset int64
		limit  int64
		want   string
	}{
		{"none requested", -1, -1, ""},
		{"both requested", 5, 10, "OFFSET 5 LIMIT 10"},
		{"zero offset", 0, 10, "OFFSET 0 LIMIT 10"},
		{"offset only", 5, -1, "OFFSET 5"},
		{"limit only", -1, 10, "LIMIT 10"},
		{"zero limit is unset", -1, 0, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := quickiedata.NewSPARQLQuery()
			query.Template = "SELECT ?x WHERE { ?x ?p ?o }"
			query.Offset = test.offset
			query.Limit = test.limit

			out, err := quickiedata.RenderSPARQLQuery(query)
			if err != nil {
				t.Fatalf("RenderSPARQLQuery: %s", err)
			}

			if test.want == "" {
				for _, unwanted := range []string{"OFFSET", "LIMIT"} {
					if strings.Contains(out, unwanted) {
						t.Errorf("query gained a %s: %s", unwanted, out)
					}
				}
				return
			}
			if !strings.Contains(out, test.want) {
				t.Errorf("query lost %q: %s", test.want, out)
			}
			if strings.Count(out, "OFFSET")+strings.Count(out, "LIMIT") > 2 {
				t.Errorf("paging clauses repeated: %s", out)
			}
		})
	}
}

// A "#" must only start a comment when it is outside a string literal and
// outside an IRI. Anything else truncates the query.
func TestRenderSPARQLQueryCommentStripping(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []string
		notWant  []string
	}{
		{
			name:     "hash in a string that contains an apostrophe",
			template: `SELECT ?x WHERE { FILTER(STR(?l) = "O'Brien #1") }`,
			want:     []string{`"O'Brien #1"`},
		},
		{
			name:     "hash in an IRI",
			template: `SELECT ?x WHERE { ?x <http://ex.org/x#y> ?o }`,
			want:     []string{`<http://ex.org/x#y>`},
		},
		{
			name:     "comment after an IRI",
			template: `SELECT ?x WHERE { ?x <http://ex.org/x#y> ?o } # real comment`,
			want:     []string{`<http://ex.org/x#y>`},
			notWant:  []string{"real comment"},
		},
		{
			name:     "hash in a long string",
			template: `SELECT ?x WHERE { BIND("""a # b""" as ?y) }`,
			want:     []string{`"""a # b"""`},
		},
		{
			name:     "hash in a long string that spans lines",
			template: "SELECT ?x WHERE { BIND(\"\"\"a\n# not a comment\"\"\" as ?y) }",
			want:     []string{"a # not a comment"},
		},
		{
			name:     "escaped quote before a hash in a string",
			template: `SELECT ?x WHERE { BIND("a\"#b" as ?y) }`,
			want:     []string{`"a\"#b"`},
		},
		{
			name:     "comment with an apostrophe",
			template: "SELECT ?x WHERE { ?s ?p ?o } # don't eat me\nLIMIT 5",
			want:     []string{"LIMIT 5"},
			notWant:  []string{"eat me"},
		},
		{
			name:     "comment after a comparison",
			template: `SELECT ?x WHERE { FILTER(?n < 5) } # note`,
			want:     []string{"?n < 5"},
			notWant:  []string{"note"},
		},
		{
			name:     "whole line comment",
			template: "# leading comment\nSELECT ?x WHERE { ?x ?p ?o }",
			want:     []string{"SELECT ?x WHERE { ?x ?p ?o }"},
			notWant:  []string{"leading comment"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := quickiedata.NewSPARQLQuery()
			query.Template = test.template

			out, err := quickiedata.RenderSPARQLQuery(query)
			if err != nil {
				t.Fatalf("RenderSPARQLQuery: %s", err)
			}

			for _, want := range test.want {
				if !strings.Contains(out, want) {
					t.Errorf("rendered query lost %q:\n  %s", want, out)
				}
			}
			for _, notWant := range test.notWant {
				if strings.Contains(out, notWant) {
					t.Errorf("rendered query kept %q:\n  %s", notWant, out)
				}
			}
		})
	}
}

// An empty list is valid SPARQL and matches nothing, so it is rendered. A
// reference that is not valid is reported with the reference itself.
func TestRenderSPARQLQueryValues(t *testing.T) {
	query := quickiedata.NewSPARQLQuery()
	query.Template = "SELECT ?x WHERE { ?x ?p ?o }"
	query.Variables.Set("v")

	out, err := quickiedata.RenderSPARQLQuery(query)
	if err != nil {
		t.Fatalf("RenderSPARQLQuery with an empty list: %s", err)
	}
	if !strings.Contains(out, "VALUES ?v {") {
		t.Errorf("query = %s, want a VALUES block", out)
	}

	bad := quickiedata.NewSPARQLQuery()
	bad.Template = "SELECT ?x WHERE { ?x ?p ?o }"
	bad.Variables.Set("v", quickiedata.IDs("wd:Q42", "nope")...)

	_, err = quickiedata.RenderSPARQLQuery(bad)
	if err == nil {
		t.Fatal("expected an error for the invalid reference")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error = %q, want it to name the invalid reference", err)
	}
}

// squash collapses runs of spaces, so a test can assert on the structure of a
// rendered query without pinning the padding around an insertion.
func squash(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// renderVariables renders a template with one bound variable.
func renderVariables(t *testing.T, template string, name string, terms []quickiedata.SPARQLTerm) (string, error) {
	t.Helper()

	query := quickiedata.NewSPARQLQuery()
	query.Template = template
	query.Variables.Set(name, terms...)
	return quickiedata.RenderSPARQLQuery(query)
}

// A variable's statement goes into the outer group pattern, and again inside
// every subquery that uses the variable: a subquery is its own scope, so the
// outer binding alone does not reach it.
func TestRenderSPARQLQueryInjectionScope(t *testing.T) {
	one := func(term quickiedata.SPARQLTerm) []quickiedata.SPARQLTerm {
		return []quickiedata.SPARQLTerm{term}
	}
	const q42 = `BIND( <http://www.wikidata.org/entity/Q42> as ?item)`

	tests := []struct {
		name     string
		template string
		variable string
		terms    []quickiedata.SPARQLTerm
		contains []string
		counts   map[string]int
		wantErr  error
	}{
		{
			name:     "subquery uses the variable",
			template: `SELECT ?y ?item WHERE { { SELECT ?y ?item WHERE { ?item wdt:P31 wd:Q5 } LIMIT 1 } }`,
			variable: "item",
			terms:    one(quickiedata.WikidataID("wd:Q42")),
			contains: []string{`{ SELECT ?y ?item WHERE { ` + q42 + ` ?item wdt:P31 wd:Q5 }`},
			counts:   map[string]int{q42: 2},
		},
		{
			name:     "dollar sigil is the same variable",
			template: `SELECT ?y ?item WHERE { { SELECT ?y ?item WHERE { $item wdt:P31 wd:Q5 } LIMIT 1 } }`,
			variable: "item",
			terms:    one(quickiedata.WikidataID("wd:Q42")),
			contains: []string{`{ SELECT ?y ?item WHERE { ` + q42 + ` $item wdt:P31 wd:Q5 }`},
			counts:   map[string]int{q42: 2},
		},
		{
			name:     "subquery does not use the variable",
			template: `SELECT ?y WHERE { { SELECT ?y WHERE { ?y wdt:P31 wd:Q5 } LIMIT 1 } ?x ?p ?o }`,
			variable: "x",
			terms:    one(quickiedata.String("v")),
			contains: []string{
				`SELECT ?y WHERE { BIND( """v""" as ?x) { SELECT`,
				`{ SELECT ?y WHERE { ?y wdt:P31 wd:Q5 }`,
			},
			counts: map[string]int{`BIND( """v""" as ?x)`: 1},
		},
		{
			name:     "list in a subquery",
			template: `SELECT ?y ?item WHERE { { SELECT ?y ?item WHERE { ?item wdt:P31 wd:Q5 } LIMIT 1 } }`,
			variable: "item",
			terms:    quickiedata.Strings("a", "b"),
			contains: []string{`{ SELECT ?y ?item WHERE { VALUES ?item { """a""" """b""" } ?item`},
			counts:   map[string]int{`VALUES ?item {`: 2},
		},
		{
			name:     "union is one scope",
			template: `SELECT ?x WHERE { { ?x wdt:P31 wd:Q5 } UNION { ?x wdt:P31 wd:Q6 } }`,
			variable: "x",
			terms:    one(quickiedata.String("v")),
			contains: []string{`WHERE { BIND( """v""" as ?x) { ?x wdt:P31 wd:Q5 }`},
			counts:   map[string]int{`BIND(`: 1},
		},
		{
			name:     "WHERE { inside a literal",
			template: `SELECT ?x WHERE { BIND("""WHERE {""" AS ?x) }`,
			variable: "name",
			terms:    one(quickiedata.String("v")),
			contains: []string{`BIND("""WHERE {""" AS ?x)`, `BIND( """v""" as ?name)`},
			counts:   map[string]int{`BIND( """v""" as ?name)`: 1},
		},
		{
			name:     "multi byte literal",
			template: `SELECT ?x WHERE { BIND("é—🔥 WHERE {" AS ?x) }`,
			variable: "name",
			terms:    one(quickiedata.String("v")),
			contains: []string{`BIND("é—🔥 WHERE {" AS ?x)`, `BIND( """v""" as ?name)`},
			counts:   map[string]int{`BIND( """v""" as ?name)`: 1},
		},
		{
			name:     "lowercase where with no space",
			template: `SELECT ?x where{ ?x ?p ?o }`,
			variable: "x",
			terms:    one(quickiedata.String("v")),
			contains: []string{`where{ BIND( """v""" as ?x) ?x ?p ?o }`},
		},
		{
			name:     "variable named where",
			template: `SELECT ?where WHERE { ?where wdt:P31 wd:Q5 }`,
			variable: "where",
			terms:    one(quickiedata.String("v")),
			contains: []string{`BIND( """v""" as ?where)`},
			counts:   map[string]int{`BIND(`: 1},
		},
		{
			name:     "prefixed name ending in where",
			template: `SELECT ?x { ?x a ex:where }`,
			variable: "x",
			terms:    one(quickiedata.String("v")),
			contains: []string{`{ BIND( """v""" as ?x) ?x a ex:where }`},
			counts:   map[string]int{`BIND(`: 1},
		},
		{
			name:     "implicit group without the WHERE keyword",
			template: `SELECT ?item { ?item wdt:P31 wd:Q146 }`,
			variable: "name",
			terms:    one(quickiedata.String("Oscar")),
			contains: []string{`SELECT ?item { BIND( """Oscar""" as ?name) ?item wdt:P31 wd:Q146 }`},
		},
		{
			name:     "construct with a where clause",
			template: `CONSTRUCT { ?s ?p ?o } WHERE { ?s ?p ?o }`,
			variable: "s",
			terms:    one(quickiedata.WikidataID("wd:Q42")),
			contains: []string{`WHERE { BIND( <http://www.wikidata.org/entity/Q42> as ?s)`},
			counts:   map[string]int{`BIND(`: 1},
		},
		{
			name:     "construct short form binds after the template",
			template: `CONSTRUCT { ?s ?p ?o } { ?s ?p ?o }`,
			variable: "s",
			terms:    one(quickiedata.WikidataID("wd:Q42")),
			contains: []string{`CONSTRUCT { ?s ?p ?o } { BIND( <http://www.wikidata.org/entity/Q42> as ?s) ?s ?p ?o }`},
			counts:   map[string]int{`BIND(`: 1},
		},
		{
			name:     "prefix prologue before an implicit group",
			template: "PREFIX wd: <http://www.wikidata.org/entity/>\nSELECT ?item { ?item wdt:P31 wd:Q5 }",
			variable: "name",
			terms:    one(quickiedata.String("v")),
			contains: []string{`SELECT ?item { BIND( """v""" as ?name) ?item wdt:P31 wd:Q5 }`},
		},
		{
			name:     "prefix and base prologue",
			template: "PREFIX : <http://example.org/>\nBASE <http://example.org/>\nSELECT ?x { ?x :p ?o }",
			variable: "x",
			terms:    one(quickiedata.String("v")),
			contains: []string{`SELECT ?x { BIND( """v""" as ?x) ?x :p ?o }`},
			counts:   map[string]int{`BIND(`: 1},
		},
		{
			name:     "comparison operator is not an iri",
			template: `SELECT ?x WHERE { ?x ?p ?o . FILTER(?o < 5) }`,
			variable: "x",
			terms:    one(quickiedata.String("v")),
			contains: []string{`WHERE { BIND( """v""" as ?x) ?x ?p ?o . FILTER(?o < 5) }`},
		},
		{
			name:     "exists is not the outer group",
			template: `SELECT (EXISTS { ?x a ?y } AS ?e) { ?x a ?y }`,
			variable: "x",
			terms:    one(quickiedata.String("v")),
			contains: []string{`AS ?e) { BIND( """v""" as ?x) ?x a ?y }`},
			counts:   map[string]int{`BIND(`: 1},
		},
		{
			name:     "describe has no group to bind in",
			template: `DESCRIBE ?x`,
			variable: "x",
			terms:    one(quickiedata.String("v")),
			wantErr:  quickiedata.ErrNoInjectionPoint,
		},
		{
			name:     "select with no group",
			template: `SELECT ?x`,
			variable: "x",
			terms:    one(quickiedata.String("v")),
			wantErr:  quickiedata.ErrNoInjectionPoint,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out, err := renderVariables(t, test.template, test.variable, test.terms)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("RenderSPARQLQuery error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RenderSPARQLQuery: %s", err)
			}
			normalized := squash(out)
			for _, want := range test.contains {
				if !strings.Contains(normalized, want) {
					t.Errorf("rendered query lost %q:\n  %s", want, out)
				}
			}
			for statement, want := range test.counts {
				if got := strings.Count(out, statement); got != want {
					t.Errorf("query has %d of %q, want %d:\n  %s", got, statement, want, out)
				}
			}
		})
	}
}

// The debug log shows the view the scanner searched and the query the endpoint
// receives, so a statement in an unexpected group can be explained.
func TestRenderSPARQLQueryLogsScannerViewAndResult(t *testing.T) {
	var logged bytes.Buffer
	previous := quickiedata.DebugLog.Writer()
	quickiedata.DebugLog.SetOutput(&logged)
	defer quickiedata.DebugLog.SetOutput(previous)

	// the literal holds a group brace, so the two views differ
	const template = `SELECT ?item WHERE { BIND("""WHERE {""" AS ?item) }`
	if _, err := renderVariables(t, template, "x", []quickiedata.SPARQLTerm{quickiedata.String("v")}); err != nil {
		t.Fatalf("RenderSPARQLQuery: %s", err)
	}

	var sawScannerView, sawQuery bool
	for _, line := range strings.Split(strings.TrimSpace(logged.String()), "\n") {
		switch {
		case strings.Contains(line, "sparql scanner view"):
			sawScannerView = true
			if strings.Contains(line, `"""WHERE {"""`) {
				t.Errorf("scanner view kept a literal: %s", line)
			}
		case strings.Contains(line, "sparql query:"):
			sawQuery = true
			for _, want := range []string{`"""WHERE {"""`, `BIND( """v""" as ?x)`} {
				if !strings.Contains(line, want) {
					t.Errorf("logged query lost %q: %s", want, line)
				}
			}
		}
	}
	if !sawScannerView || !sawQuery {
		t.Errorf("log = %q, want a scanner view line and a query line", logged.String())
	}
}

// The inserted block is the only difference from the template, so a caller can
// still recognise its own query in the output.
func TestRenderSPARQLQueryInjectionIsRemovable(t *testing.T) {
	const template = `SELECT ?x WHERE { ?x ?p ?o }`
	const statement = `BIND( """v""" as ?x)`

	out, err := renderVariables(t, template, "x", []quickiedata.SPARQLTerm{quickiedata.String("v")})
	if err != nil {
		t.Fatalf("RenderSPARQLQuery: %s", err)
	}
	if got := strings.Replace(out, " "+statement+" ", "", 1); got != template {
		t.Errorf("query without the statement = %q, want %q", got, template)
	}
}

// A binding that cannot be simplified is a debug detail, not something a
// library prints over the caller's standard error.
func TestSPARQLSimplifyFailureGoesToDebugLog(t *testing.T) {
	var logged bytes.Buffer
	previous := quickiedata.DebugLog.Writer()
	quickiedata.DebugLog.SetOutput(&logged)
	defer quickiedata.DebugLog.SetOutput(previous)

	// an integer literal that does not parse
	value := "not a number"
	response := &quickiedata.SPARQLResponse{}
	response.Results.Bindings = []map[string]*quickiedata.BindingValue{
		{"count": {Value: &value, Type: "literal", DataType: "http://www.w3.org/2001/XMLSchema#integer"}},
	}

	simple := response.Simplify()
	if len(simple.Results) != 1 {
		t.Fatalf("Results = %+v, want one row", simple.Results)
	}
	if len(simple.Results[0]) != 0 {
		t.Errorf("row = %+v, want the unparsable binding dropped", simple.Results[0])
	}
	if !strings.Contains(logged.String(), "error while simplifying") {
		t.Errorf("debug log = %q, want the simplify error in it", logged.String())
	}
}

// A single term is a BIND, a list is a VALUES clause, and Set replaces while
// Add appends.
func TestVariablesSetAndAdd(t *testing.T) {
	query := quickiedata.NewSPARQLQuery()
	query.Template = "SELECT ?v WHERE { ?v ?p ?o }"

	query.Variables.Set("v", quickiedata.String("a"))
	rendered, err := quickiedata.RenderSPARQLQuery(query)
	if err != nil {
		t.Fatalf("RenderSPARQLQuery: %s", err)
	}
	if !strings.Contains(rendered, `BIND( """a""" as ?v)`) {
		t.Errorf("one term should be a BIND:\n  %s", rendered)
	}

	query.Variables.Set("v", quickiedata.Strings("a", "b")...)
	query.Variables.Add("v", quickiedata.String("c"))
	rendered, err = quickiedata.RenderSPARQLQuery(query)
	if err != nil {
		t.Fatalf("RenderSPARQLQuery: %s", err)
	}
	if !strings.Contains(rendered, `VALUES ?v { """a""" """b""" """c""" }`) {
		t.Errorf("terms should be a VALUES clause:\n  %s", rendered)
	}
	if got := len(query.Variables["v"]); got != 3 {
		t.Errorf("Get returned %d terms, want 3", got)
	}
}
