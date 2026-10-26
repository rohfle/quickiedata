package quickiedata_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rohfle/quickiedata"
)

// A template may already carry its own offset or limit, and the caller may
// ask for one: it must be added once, or not at all.
func TestRenderSPARQLQueryOffsetAndLimit(t *testing.T) {
	tests := []struct {
		name     string
		template string
		offset   int64
		limit    int64
		want     string // must appear in the result
		once     string // must appear exactly once
	}{
		{"none requested", "SELECT ?x WHERE { ?x ?p ?o }", -1, -1, "", ""},
		{"both requested", "SELECT ?x WHERE { ?x ?p ?o }", 5, 10, "OFFSET 5 LIMIT 10", "OFFSET"},
		{"zero offset", "SELECT ?x WHERE { ?x ?p ?o }", 0, 10, "OFFSET 0 LIMIT 10", "OFFSET"},
		{"template has a limit", "SELECT ?x WHERE { ?x ?p ?o } LIMIT 3", -1, 10, "LIMIT 3", "LIMIT"},
		{"template has both", "SELECT ?x WHERE { ?x ?p ?o } OFFSET 2 LIMIT 3", 5, 10, "OFFSET 2 LIMIT 3", "OFFSET"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query := quickiedata.NewSPARQLQuery()
			query.Template = test.template
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
			if strings.Count(out, test.once) != 1 {
				t.Errorf("%s appears %d times, want 1: %s", test.once, strings.Count(out, test.once), out)
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

// A template with no WHERE clause cannot take variable statements, so the
// variables are left out and the query is sent as it is.
func TestRenderSPARQLQueryWithoutWhereClause(t *testing.T) {
	query := quickiedata.NewSPARQLQuery()
	query.Template = "SELECT ?item { ?item wdt:P31 wd:Q146 }"
	query.Variables.Set("name", quickiedata.String("Oscar"))

	got, err := quickiedata.RenderSPARQLQuery(query)
	if err != nil {
		t.Fatalf("RenderSPARQLQuery: %s", err)
	}
	if got != "SELECT ?item { ?item wdt:P31 wd:Q146 }" {
		t.Errorf("RenderSPARQLQuery = %q, want the query unchanged", got)
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
	if got := len(query.Variables.Get("v")); got != 3 {
		t.Errorf("Get returned %d terms, want 3", got)
	}
}
