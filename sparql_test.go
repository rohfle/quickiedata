package quickiedata_test

import (
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
