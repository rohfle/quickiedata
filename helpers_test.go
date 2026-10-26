package quickiedata_test

import (
	"testing"

	"github.com/rohfle/quickiedata"
)

func TestGetSitelinkURL(t *testing.T) {
	tests := []struct {
		site  string
		title string
		want  string
	}{
		// Wikidata entity titles belong on www.wikidata.org. A
		// wikidata.wikimedia.org host does not exist.
		{"wikidatawiki", "Q42", "https://www.wikidata.org/wiki/Q42"},
		{"wikidatawiki", "P31", "https://www.wikidata.org/wiki/Property:P31"},
		{"wikidatawiki", "L1", "https://www.wikidata.org/wiki/Lexeme:L1"},
		{"wikidatawiki", "L1-F1", "https://www.wikidata.org/wiki/Lexeme:L1#F1"},
		{"wikidatawiki", "E123", "https://www.wikidata.org/wiki/EntitySchema:E123"},

		// Sites that are hosted on their own wiki.
		{"enwiki", "Hubble Space Telescope", "https://en.wikipedia.org/wiki/Hubble_Space_Telescope"},
		{"enwikiquote", "Foo", "https://en.wikiquote.org/wiki/Foo"},
		{"commonswiki", "File:Foo.jpg", "https://commons.wikimedia.org/wiki/File%3AFoo.jpg"},
		{"metawiki", "Main Page", "https://meta.wikimedia.org/wiki/Main_Page"},
		{"mediawikiwiki", "Manual:FAQ", "https://www.mediawiki.org/wiki/Manual%3AFAQ"},

		// No url can be made from these.
		{"enwiki", "", ""},
		{"", "Q42", ""},
		{"bad", "x", ""},
	}

	for _, test := range tests {
		t.Run(test.site+"/"+test.title, func(t *testing.T) {
			if got := quickiedata.GetSitelinkURL(test.site, test.title); got != test.want {
				t.Errorf("GetSitelinkURL(%q, %q) = %q, want %q", test.site, test.title, got, test.want)
			}
		})
	}
}

// ParseEntityID is the one place that decides what an entity id is, for both
// the entity API (bare) and a SPARQL value (bare or prefixed).
func TestParseEntityID(t *testing.T) {
	tests := []struct {
		id      string
		want    quickiedata.WikidataID
		wantErr bool
	}{
		{id: "Q42", want: "wd:Q42"},
		{id: "P31", want: "wd:P31"},
		{id: "L1", want: "wd:L1"},
		{id: "M6661797", want: "wd:M6661797"},
		{id: "L1-F1", want: "wd:L1-F1"},
		{id: "L1-S1", want: "wd:L1-S1"},
		{id: "wd:Q42", want: "wd:Q42"},
		{id: "NOOP", wantErr: true},
		{id: "wikibase:Q42", wantErr: true},
		{id: "wdt:P31", wantErr: true},
		{id: "p:P31", wantErr: true},
		{id: "ps:P31", wantErr: true},
		{id: "pq:P31", wantErr: true},
		{id: "foo:Q42", wantErr: true},
		{id: "q42", wantErr: true},
		{id: "Q0", wantErr: true},
		{id: "E123", wantErr: true},
		{id: "", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			got, err := quickiedata.ParseEntityID(test.id)
			if test.wantErr {
				if err == nil {
					t.Fatalf("ParseEntityID(%q) = %q, want an error", test.id, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseEntityID(%q): %s", test.id, err)
			}
			if got != test.want {
				t.Errorf("ParseEntityID(%q) = %q, want %q", test.id, got, test.want)
			}
		})
	}
}
