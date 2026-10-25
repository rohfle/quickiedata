package quickiedata

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// SPARQLTerm is a value that renders as one SPARQL term, an IRI or a literal,
// which a query binds with BIND.
type SPARQLTerm interface {
	SPARQLTerm() (string, error)
}

// WikidataID is an entity reference such as "wd:Q42", bound as written, so the
// query has to declare the prefix it uses.
type WikidataID string

// ValidSPARQLWikipediaID matches a prefixed entity id such as wd:Q42, wd:P31 or
// wd:L1.
var ValidSPARQLWikipediaID = regexp.MustCompile(`^[a-z]+:(?:NOOP|[PLSFQ][1-9]\d*)$`)

// SPARQLTerm returns the reference once it validates.
func (id WikidataID) SPARQLTerm() (string, error) {
	if !ValidSPARQLWikipediaID.MatchString(string(id)) {
		return "", fmt.Errorf("invalid wikidata reference '%s'", id)
	}
	return string(id), nil
}

// Datatype IRIs, written in full so a query needs no prefix for them.
const (
	xsdDateTime        = "http://www.w3.org/2001/XMLSchema#dateTime"
	xsdDayTimeDuration = "http://www.w3.org/2001/XMLSchema#dayTimeDuration"
)

// WikidataDateTime is an instant, bound as an xsd:dateTime literal. Precision
// is not part of a literal: a date known only to the year is stored as its
// first day, so filter a period with a range instead.
type WikidataDateTime time.Time

// NewWikidataDateTime returns the instant in UTC.
func NewWikidataDateTime(t time.Time) WikidataDateTime {
	return WikidataDateTime(t.UTC())
}

// Time returns the instant.
func (d WikidataDateTime) Time() time.Time {
	return time.Time(d)
}

// SPARQLTerm renders "2001-08-25T00:00:00Z"^^xsd:dateTime. Fractional seconds
// appear only when set, and the year follows XSD 1.1, where 1 BCE is year 0
// rather than the -0001 the entity API uses.
func (d WikidataDateTime) SPARQLTerm() (string, error) {
	instant := time.Time(d)
	return fmt.Sprintf("%q^^<%s>", instant.Format("2006-01-02T15:04:05.999999999Z"), xsdDateTime), nil
}

// WikidataDuration is a length of time, bound as an xsd:dayTimeDuration
// literal, the duration type the query service compares. Days are separate
// because that type has no month or year component.
type WikidataDuration struct {
	Days        int64
	Hours       int
	Minutes     int
	Seconds     int
	Nanoseconds int
}

// NewWikidataDuration splits a duration into its parts. It cannot hold a month
// or a span longer than about 292 years; build the struct directly for those.
func NewWikidataDuration(d time.Duration) WikidataDuration {
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	return WikidataDuration{
		Days:        int64(days),
		Hours:       int(d / time.Hour),
		Minutes:     int(d % time.Hour / time.Minute),
		Seconds:     int(d % time.Minute / time.Second),
		Nanoseconds: int(d % time.Second),
	}
}

// SPARQLTerm renders "P1DT2H3M4.5S"^^xsd:dayTimeDuration. The time part has to
// be normalised, and a negative duration has no representation here.
func (d WikidataDuration) SPARQLTerm() (string, error) {
	if d.Days < 0 || d.Hours < 0 || d.Minutes < 0 || d.Seconds < 0 || d.Nanoseconds < 0 {
		return "", fmt.Errorf("duration must not be negative, got %d days %02d:%02d:%02d.%09d",
			d.Days, d.Hours, d.Minutes, d.Seconds, d.Nanoseconds)
	}
	if d.Hours > 23 || d.Minutes > 59 || d.Seconds > 59 || d.Nanoseconds > 999999999 {
		return "", fmt.Errorf("duration parts out of range, want hours < 24 and minutes, seconds, nanoseconds < 60, 60, 1e9; got %d days %02d:%02d:%02d.%09d",
			d.Days, d.Hours, d.Minutes, d.Seconds, d.Nanoseconds)
	}

	seconds := fmt.Sprintf("%d", d.Seconds)
	if d.Nanoseconds != 0 {
		seconds += "." + strings.TrimRight(fmt.Sprintf("%09d", d.Nanoseconds), "0")
	}

	literal := "P"
	if d.Days != 0 {
		literal += fmt.Sprintf("%dD", d.Days)
	}
	switch {
	case d.Hours != 0 || d.Minutes != 0 || d.Seconds != 0 || d.Nanoseconds != 0:
		literal += fmt.Sprintf("T%dH%dM%sS", d.Hours, d.Minutes, seconds)
	case d.Days == 0:
		// XSD writes a zero duration as PT0S
		literal = "PT0S"
	}

	return fmt.Sprintf("%q^^<%s>", literal, xsdDayTimeDuration), nil
}

var (
	_ SPARQLTerm = WikidataID("")
	_ SPARQLTerm = WikidataDateTime{}
	_ SPARQLTerm = WikidataDuration{}
)
