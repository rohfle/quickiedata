package quickiedata

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SPARQLTerm is a value that renders as one SPARQL term, an IRI or a literal,
// which a query binds with BIND.
type SPARQLTerm interface {
	SPARQLTerm() (string, error)
}

// Datatype and IRI prefixes, written in full so a query needs no prefix for
// values this library injects. The entity base is Wikidata's; another Wikibase
// instance has its own, which this library does not target.
const (
	wikidataEntityIRI  = "http://www.wikidata.org/entity/"
	xsdDateTime        = "http://www.w3.org/2001/XMLSchema#dateTime"
	xsdDayTimeDuration = "http://www.w3.org/2001/XMLSchema#dayTimeDuration"
	xsdDecimal         = "http://www.w3.org/2001/XMLSchema#decimal"
	xsdDouble          = "http://www.w3.org/2001/XMLSchema#double"
	wktLiteral         = "http://www.opengis.net/ont/geosparql#wktLiteral"
)

// WikidataID is an entity reference, written bare (Q42) or with the entity
// prefix (wd:Q42). It renders as the entity IRI, so the query needs no prefix
// itself.
type WikidataID string

// SPARQLTerm returns the entity IRI once the reference validates.
func (id WikidataID) SPARQLTerm() (string, error) {
	canonical, err := ParseEntityID(string(id))
	if err != nil {
		return "", err
	}
	_, local, _ := strings.Cut(string(canonical), ":")
	return fmt.Sprintf("<%s%s>", wikidataEntityIRI, local), nil
}

// String is a plain string literal.
type String string

// SPARQLTerm renders the value with the long string form, so it can hold
// quotes and line breaks.
func (s String) SPARQLTerm() (string, error) {
	value := strings.ReplaceAll(string(s), `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return fmt.Sprintf(`"""%s"""`, value), nil
}

// Integer is an xsd:integer literal.
type Integer int64

// SPARQLTerm renders the number.
func (i Integer) SPARQLTerm() (string, error) {
	return strconv.FormatInt(int64(i), 10), nil
}

// Boolean is an xsd:boolean literal.
type Boolean bool

// SPARQLTerm renders true or false.
func (b Boolean) SPARQLTerm() (string, error) {
	if bool(b) {
		return "true", nil
	}
	return "false", nil
}

// Float is an xsd:double literal. A decimal that has to be exact is a Decimal.
type Float float64

// SPARQLTerm renders the number in exponent form, and the special values the
// way XSD writes them.
func (f Float) SPARQLTerm() (string, error) {
	switch value := float64(f); {
	case math.IsNaN(value):
		return fmt.Sprintf(`"NaN"^^<%s>`, xsdDouble), nil
	case math.IsInf(value, 1):
		return fmt.Sprintf(`"INF"^^<%s>`, xsdDouble), nil
	case math.IsInf(value, -1):
		return fmt.Sprintf(`"-INF"^^<%s>`, xsdDouble), nil
	default:
		return strconv.FormatFloat(value, 'E', -1, 64), nil
	}
}

// Decimal is an xsd:decimal literal. It is a string because float64 cannot
// represent every decimal exactly.
type Decimal string

var decimalPattern = regexp.MustCompile(`^[+-]?\d+(?:\.\d+)?$`)

// NewDecimal returns a decimal literal, rejecting anything that is not one, an
// exponent in particular, which is an xsd:double.
func NewDecimal(value string) (Decimal, error) {
	if !decimalPattern.MatchString(value) {
		return "", fmt.Errorf("decimal %q is not an xsd:decimal, use a Float for an exponent", value)
	}
	return Decimal(value), nil
}

// SPARQLTerm renders the decimal, checking it when it was built by conversion.
func (d Decimal) SPARQLTerm() (string, error) {
	if !decimalPattern.MatchString(string(d)) {
		return "", fmt.Errorf("decimal %q is not an xsd:decimal, use a Float for an exponent", d)
	}
	return fmt.Sprintf(`"%s"^^<%s>`, d, xsdDecimal), nil
}

// Text is a language-tagged string, the form a label or a description takes.
type Text struct {
	Language string
	Value    string
}

var languageTagPattern = regexp.MustCompile(`^[A-Za-z]+(?:-[A-Za-z0-9]+)*$`)

// SPARQLTerm renders "value"@language, lowering the language tag as RDF does.
func (t Text) SPARQLTerm() (string, error) {
	if !languageTagPattern.MatchString(t.Language) {
		return "", fmt.Errorf("invalid language tag %q", t.Language)
	}
	return fmt.Sprintf("%s@%s", strconv.Quote(t.Value), strings.ToLower(t.Language)), nil
}

// Coordinate is a point on the globe. It renders as a WKT literal with the
// longitude first, which is the order the WKT alphabet uses.
type Coordinate struct {
	Latitude  float64
	Longitude float64
}

// SPARQLTerm renders "Point(lon lat)"^^geo:wktLiteral.
func (c Coordinate) SPARQLTerm() (string, error) {
	if math.IsNaN(c.Latitude) || math.IsInf(c.Latitude, 0) || c.Latitude < -90 || c.Latitude > 90 {
		return "", fmt.Errorf("latitude %v is not between -90 and 90", c.Latitude)
	}
	if math.IsNaN(c.Longitude) || math.IsInf(c.Longitude, 0) || c.Longitude < -180 || c.Longitude > 180 {
		return "", fmt.Errorf("longitude %v is not between -180 and 180", c.Longitude)
	}
	return fmt.Sprintf(`"Point(%s %s)"^^<%s>`,
		strconv.FormatFloat(c.Longitude, 'f', -1, 64),
		strconv.FormatFloat(c.Latitude, 'f', -1, 64),
		wktLiteral), nil
}

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

// Strings returns a term list of string literals, for a VALUES clause.
func Strings(values ...string) []SPARQLTerm {
	terms := make([]SPARQLTerm, 0, len(values))
	for _, value := range values {
		terms = append(terms, String(value))
	}
	return terms
}

// IDs returns a term list of entity references such as "wd:Q42".
func IDs(ids ...string) []SPARQLTerm {
	terms := make([]SPARQLTerm, 0, len(ids))
	for _, id := range ids {
		terms = append(terms, WikidataID(id))
	}
	return terms
}

// Terms returns the arguments as a term list, for a mixed VALUES clause.
func Terms(terms ...SPARQLTerm) []SPARQLTerm {
	return terms
}

// Compile-time checks that every term type implements SPARQLTerm.
var (
	_ SPARQLTerm = WikidataID("")
	_ SPARQLTerm = String("")
	_ SPARQLTerm = Integer(0)
	_ SPARQLTerm = Boolean(false)
	_ SPARQLTerm = Float(0)
	_ SPARQLTerm = Decimal("")
	_ SPARQLTerm = Text{}
	_ SPARQLTerm = Coordinate{}
	_ SPARQLTerm = WikidataDateTime{}
	_ SPARQLTerm = WikidataDuration{}
)
