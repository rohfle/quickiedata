package quickiedata_test

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/rohfle/quickiedata"
)

// renderBound renders a query with one bound variable and returns the result.
func renderBound(t *testing.T, term quickiedata.SPARQLTerm) (string, error) {
	t.Helper()

	query := quickiedata.NewSPARQLQuery()
	query.Template = "SELECT ?x WHERE { ?x wdt:P31 wd:Q5 }"
	query.Variables.Set("at", term)
	return quickiedata.RenderSPARQLQuery(query)
}

// renderBoundList renders a query whose variable holds a term list.
func renderBoundList(t *testing.T, terms []quickiedata.SPARQLTerm) (string, error) {
	t.Helper()

	query := quickiedata.NewSPARQLQuery()
	query.Template = "SELECT ?x WHERE { ?x wdt:P31 wd:Q5 }"
	query.Variables.Set("at", terms...)
	return quickiedata.RenderSPARQLQuery(query)
}

func TestWikidataDateTimeLiteral(t *testing.T) {
	tests := []struct {
		name  string
		value time.Time
		want  string
	}{
		{
			name:  "timestamp",
			value: time.Date(1952, 3, 11, 0, 0, 0, 0, time.UTC),
			want:  `BIND( "1952-03-11T00:00:00Z"^^<http://www.w3.org/2001/XMLSchema#dateTime> as ?at)`,
		},
		{
			name:  "offset becomes UTC",
			value: time.Date(2020, 1, 1, 1, 30, 0, 0, time.FixedZone("CET", 2*60*60)),
			want:  `BIND( "2019-12-31T23:30:00Z"^^<http://www.w3.org/2001/XMLSchema#dateTime> as ?at)`,
		},
		{
			name:  "fractional second",
			value: time.Date(2020, 5, 6, 7, 8, 9, 500000000, time.UTC),
			want:  `BIND( "2020-05-06T07:08:09.5Z"^^<http://www.w3.org/2001/XMLSchema#dateTime> as ?at)`,
		},
		{
			name:  "year 0",
			value: time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC),
			want:  `BIND( "0000-01-01T00:00:00Z"^^<http://www.w3.org/2001/XMLSchema#dateTime> as ?at)`,
		},
		{
			name:  "negative year",
			value: time.Date(-383, 1, 1, 0, 0, 0, 0, time.UTC),
			want:  `BIND( "-0383-01-01T00:00:00Z"^^<http://www.w3.org/2001/XMLSchema#dateTime> as ?at)`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rendered, err := renderBound(t, quickiedata.NewWikidataDateTime(test.value))
			if err != nil {
				t.Fatalf("RenderSPARQLQuery: %s", err)
			}
			if !strings.Contains(rendered, test.want) {
				t.Errorf("rendered = %q, want it to contain %q", rendered, test.want)
			}
		})
	}
}

func TestWikidataDurationLiteral(t *testing.T) {
	tests := []struct {
		name    string
		value   quickiedata.WikidataDuration
		want    string
		wantErr string
	}{
		{
			name:  "days only",
			value: quickiedata.NewWikidataDuration(60 * 24 * time.Hour),
			want:  `BIND( "P60D"^^<http://www.w3.org/2001/XMLSchema#dayTimeDuration> as ?at)`,
		},
		{
			name:  "days, time and fraction",
			value: quickiedata.NewWikidataDuration(26*time.Hour + 3*time.Minute + 4*time.Second + 500*time.Millisecond),
			want:  `BIND( "P1DT2H3M4.5S"^^<http://www.w3.org/2001/XMLSchema#dayTimeDuration> as ?at)`,
		},
		{
			name:  "time only",
			value: quickiedata.NewWikidataDuration(90 * time.Minute),
			want:  `BIND( "PT1H30M0S"^^<http://www.w3.org/2001/XMLSchema#dayTimeDuration> as ?at)`,
		},
		{
			name:  "zero",
			value: quickiedata.NewWikidataDuration(0),
			want:  `BIND( "PT0S"^^<http://www.w3.org/2001/XMLSchema#dayTimeDuration> as ?at)`,
		},
		{
			name:  "longer than time.Duration holds",
			value: quickiedata.WikidataDuration{Days: 365250000000},
			want:  `BIND( "P365250000000D"^^<http://www.w3.org/2001/XMLSchema#dayTimeDuration> as ?at)`,
		},
		{
			name:    "negative",
			value:   quickiedata.NewWikidataDuration(-time.Hour),
			wantErr: "must not be negative",
		},
		{
			name:    "hour should be a day",
			value:   quickiedata.WikidataDuration{Days: 1, Hours: 24},
			wantErr: "out of range",
		},
		{
			name:    "minute should be an hour",
			value:   quickiedata.WikidataDuration{Minutes: 60},
			wantErr: "out of range",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rendered, err := renderBound(t, test.value)

			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("RenderSPARQLQuery = %q, want an error", rendered)
				}
				if !strings.Contains(err.Error(), test.wantErr) {
					t.Errorf("error = %q, want it to contain %q", err, test.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("RenderSPARQLQuery: %s", err)
			}
			if !strings.Contains(rendered, test.want) {
				t.Errorf("rendered = %q, want it to contain %q", rendered, test.want)
			}
		})
	}
}

// testTerm is a caller-defined term, to check that the renderer goes through
// the interface rather than a fixed list of types.
type testTerm struct{ text string }

func (t testTerm) SPARQLTerm() (string, error) { return t.text, nil }

func TestSPARQLTermRendering(t *testing.T) {
	tests := []struct {
		name    string
		value   quickiedata.SPARQLTerm
		want    string
		wantErr string
	}{
		{
			name:  "wikidata id",
			value: quickiedata.WikidataID("wd:Q42"),
			want:  `BIND( <http://www.wikidata.org/entity/Q42> as ?at)`,
		},
		{
			name:  "wikibase prefix is accepted",
			value: quickiedata.WikidataID("wikibase:Q42"),
			want:  `BIND( <http://www.wikidata.org/entity/Q42> as ?at)`,
		},
		{
			name:    "another prefix is rejected",
			value:   quickiedata.WikidataID("foo:Q42"),
			wantErr: "invalid wikidata reference",
		},
		{
			name:  "caller defined term",
			value: testTerm{text: `"custom"^^<http://example.org/t>`},
			want:  `BIND( "custom"^^<http://example.org/t> as ?at)`,
		},
		{
			name:    "invalid wikidata id",
			value:   quickiedata.WikidataID("nope"),
			wantErr: "invalid wikidata reference",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rendered, err := renderBound(t, test.value)

			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RenderSPARQLQuery: %s", err)
			}
			if !strings.Contains(rendered, test.want) {
				t.Errorf("rendered = %q, want it to contain %q", rendered, test.want)
			}
		})
	}
}

func TestSPARQLTermListRendering(t *testing.T) {
	tests := []struct {
		name    string
		value   []quickiedata.SPARQLTerm
		want    string
		wantErr string
	}{
		{
			name: "mixed terms",
			value: []quickiedata.SPARQLTerm{
				quickiedata.WikidataID("wd:Q42"),
				quickiedata.NewWikidataDateTime(time.Date(1952, 3, 11, 0, 0, 0, 0, time.UTC)),
			},
			want: `VALUES ?at { <http://www.wikidata.org/entity/Q42> "1952-03-11T00:00:00Z"^^<http://www.w3.org/2001/XMLSchema#dateTime> }`,
		},
		{
			name:  "wikidata ids keep working",
			value: quickiedata.IDs("wd:Q42", "wd:Q1"),
			want:  `VALUES ?at { <http://www.wikidata.org/entity/Q42> <http://www.wikidata.org/entity/Q1> }`,
		},
		{
			name:    "an invalid term fails the whole list",
			value:   []quickiedata.SPARQLTerm{quickiedata.WikidataID("wd:Q42"), quickiedata.WikidataID("nope")},
			wantErr: "invalid wikidata reference",
		},
		{
			name:    "a nil term is reported, not a panic",
			value:   []quickiedata.SPARQLTerm{nil},
			wantErr: "nil sparql term",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rendered, err := renderBoundList(t, test.value)

			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RenderSPARQLQuery: %s", err)
			}
			if !strings.Contains(rendered, test.want) {
				t.Errorf("rendered = %q, want it to contain %q", rendered, test.want)
			}
		})
	}
}

func TestTermConstructors(t *testing.T) {
	decimal, err := quickiedata.NewDecimal("12.5")
	if err != nil {
		t.Fatalf("NewDecimal: %s", err)
	}

	tests := []struct {
		name    string
		term    quickiedata.SPARQLTerm
		want    string
		wantErr string
	}{
		{
			name: "string",
			term: quickiedata.String("Oscar"),
			want: `BIND( """Oscar""" as ?at)`,
		},
		{
			name: "string with a quote",
			term: quickiedata.String(`say "hi"`),
			want: `say \"hi\"`,
		},
		{
			name: "integer",
			term: quickiedata.Integer(42),
			want: `BIND( 42 as ?at)`,
		},
		{
			name: "boolean",
			term: quickiedata.Boolean(true),
			want: `BIND( true as ?at)`,
		},
		{
			name: "float",
			term: quickiedata.Float(1500),
			want: `BIND( 1.5E+03 as ?at)`,
		},
		{
			name: "float infinity",
			term: quickiedata.Float(math.Inf(1)),
			want: `BIND( "INF"^^<http://www.w3.org/2001/XMLSchema#double> as ?at)`,
		},
		{
			name: "decimal",
			term: decimal,
			want: `BIND( "12.5"^^<http://www.w3.org/2001/XMLSchema#decimal> as ?at)`,
		},
		{
			name:    "decimal with an exponent",
			term:    quickiedata.Decimal("1e5"),
			wantErr: "is not an xsd:decimal",
		},
		{
			name: "language text",
			term: quickiedata.Text{Language: "EN", Value: "Douglas Adams"},
			want: `BIND( "Douglas Adams"@en as ?at)`,
		},
		{
			name:    "text with a broken language tag",
			term:    quickiedata.Text{Language: "e n", Value: "x"},
			wantErr: "invalid language tag",
		},
		{
			name: "coordinate",
			term: quickiedata.Coordinate{Latitude: 52.5, Longitude: 13.4},
			want: `BIND( "Point(13.4 52.5)"^^<http://www.opengis.net/ont/geosparql#wktLiteral> as ?at)`,
		},
		{
			name:    "coordinate out of range",
			term:    quickiedata.Coordinate{Latitude: 91, Longitude: 0},
			wantErr: "latitude",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rendered, err := renderBound(t, test.term)

			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RenderSPARQLQuery: %s", err)
			}
			if !strings.Contains(rendered, test.want) {
				t.Errorf("rendered = %q, want it to contain %q", rendered, test.want)
			}
		})
	}
}
