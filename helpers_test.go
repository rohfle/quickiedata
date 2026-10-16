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
