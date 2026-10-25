package quickiedata

import "regexp"

// Migration guide for the Wikidata Query Service change from v1 to v2.
const (
	wdqsMigrationGuide = "https://www.wikidata.org/wiki/Wikidata:SPARQL_query_service/WDQS_backend_update/Migrate"
	wdqsCompareGuide   = "https://www.wikidata.org/wiki/Wikidata:SPARQL_query_service/WDQS_backend_update/Compare"
)

// SPARQLIssue reports a construct that WDQS v2 does not support, or supports
// differently. It names what was found, says what to do
// about it, and links to the part of the migration guide that explains it.
type SPARQLIssue struct {
	Construct string
	Message   string
	DocURL    string
}

// sparqlCompatibilityRule matches one deprecated construct. Detection is
// textual: it finds what the query says, not what it means.
type sparqlCompatibilityRule struct {
	construct string
	message   string
	doc       string
	pattern   *regexp.Regexp
	// inLiterals scans the query with its string literals intact, for rules
	// about the literal itself rather than about a keyword.
	inLiterals bool
}

var sparqlCompatibilityRules = []sparqlCompatibilityRule{
	{
		construct: "wikibase:label",
		message:   "the label service does not exist in WDQS v2, use OPTIONAL blocks with rdfs:label",
		doc:       wdqsMigrationGuide + "#Label_service",
		pattern:   regexp.MustCompile(`(?i)wikibase:label\b`),
	},
	{
		construct: "wikibase:mwapi",
		message:   "a SPARQL query cannot call the MediaWiki API in WDQS v2, get the values separately",
		doc:       wdqsMigrationGuide + "#MediaWiki_API_service",
		pattern:   regexp.MustCompile(`(?i)wikibase:mwapi\b`),
	},
	{
		construct: "wikibase:around or wikibase:box",
		message:   "the geospatial services are not in WDQS v2, use GeoSPARQL and RDF coordinates",
		doc:       wdqsMigrationGuide + "#Geospatial",
		pattern:   regexp.MustCompile(`(?i)wikibase:(?:around|box)\b`),
	},
	{
		construct: "geof:distance",
		message:   "geof:distance is not in WDQS v2, use GeoSPARQL and RDF coordinates",
		doc:       wdqsMigrationGuide + "#Geospatial",
		pattern:   regexp.MustCompile(`(?i)geof:distance\b`),
	},
	{
		construct: "bd:slice",
		message:   "bd:slice is not in WDQS v2, paginate with ORDER BY and OFFSET/LIMIT",
		doc:       wdqsMigrationGuide + "#bd:slice",
		pattern:   regexp.MustCompile(`(?i)bd:slice\b`),
	},
	{
		construct: "bd:sample",
		message:   "bd:sample is not in WDQS v2, use LIMIT with ORDER BY",
		doc:       wdqsMigrationGuide + "#bd:sample",
		pattern:   regexp.MustCompile(`(?i)bd:sample\b`),
	},
	{
		construct: "query hint",
		message:   "WDQS v2 has no query hints, the directive is ignored; use subqueries to control evaluation",
		doc:       wdqsMigrationGuide + "#Query_hints",
		pattern:   regexp.MustCompile(`(?i)\bhint:(?:Query|Prior|Group|GroupAndSubGroups|SubQuery)\b`),
	},
	{
		construct: "named subquery",
		message:   "%name subqueries are not SPARQL 1.1, inline or restructure them",
		doc:       wdqsMigrationGuide + "#Named_subqueries",
		pattern:   regexp.MustCompile(`(?i)(?:\bWITH\s+%|\bINCLUDE\s+%|%[A-Za-z_])`),
	},
	{
		construct: "wikibase:decodeURI",
		message:   "wikibase:decodeURI is not in WDQS v2, decode the URL in your application",
		doc:       wdqsMigrationGuide + "#wikibase:decodeURI()",
		pattern:   regexp.MustCompile(`(?i)wikibase:decodeURI\b`),
	},
	{
		construct: "wikibase:isSomeValue",
		message:   "wikibase:isSomeValue is not in WDQS v2, filter on the skolemized someValue IRI",
		doc:       wdqsMigrationGuide + "#wikibase:isSomeValue()",
		pattern:   regexp.MustCompile(`(?i)wikibase:isSomeValue\b`),
	},
	{
		construct: "GAS service",
		message:   "the gather-apply-scatter service is not in WDQS v2, use path search",
		doc:       wdqsMigrationGuide + "#GAS_service",
		pattern:   regexp.MustCompile(`(?i)SERVICE\s+gas:`),
	},
	{
		construct:  "date without a time part",
		message:    `WDQS v2 requires a time part, write "2001-01-01T00:00:00"^^xsd:dateTime`,
		doc:        wdqsMigrationGuide + "#Date_and_time_parsing",
		pattern:    regexp.MustCompile(`"\d{4,}-\d{2}-\d{2}"\s*\^\^\s*(?:xsd:dateTime|<http://www\.w3\.org/2001/XMLSchema#dateTime>)`),
		inLiterals: true,
	},
	{
		construct: "xsd:duration",
		message:   "WDQS v2 does not compare xsd:duration, use xsd:dayTimeDuration or yearMonthDuration",
		doc:       wdqsMigrationGuide + "#Date_and_time_parsing",
		pattern:   regexp.MustCompile(`\^\^\s*(?:xsd:duration\b|<http://www\.w3\.org/2001/XMLSchema#duration>)`),
	},
}

// aggregateRebindPattern matches an aggregate that binds the variable it
// aggregates, such as SAMPLE(?v) AS ?v. SPARQL 1.1 forbids binding a variable
// twice in one scope, v1 allowed it. Go's regexp has no
// backreferences, so the two captures are compared in code.
var aggregateRebindPattern = regexp.MustCompile(
	`(?i)\(\s*(?:SAMPLE|COUNT|MIN|MAX|SUM|AVG)\s*\(\s*(?:DISTINCT\s+)?\?(\w+)\s*\)\s+AS\s+\?(\w+)`)

// SPARQLCompatibilityIssues reports the constructs in a SPARQL query that
// WDQS v2 does not support, or does not support the same way as v1. The query is scanned, never rewritten: a caller migrating to
// v2 has to decide what each issue means for its query.
//
// Detection is textual and works on the query without its comments and string
// literal contents, so a keyword mentioned in a comment or a literal does not
// count. It can still miss a construct spelled with a prefix alias, for
// example a SERVICE <http://wikiba.se/ontology#label> or a non-default prefix
// declared as PREFIX wb: <http://wikiba.se/ontology#>.
func SPARQLCompatibilityIssues(query string) []SPARQLIssue {
	code := stripSPARQLCode(query)
	verbose := stripSPARQLComments(query)

	var issues []SPARQLIssue
	seen := make(map[string]bool, len(sparqlCompatibilityRules)+1)

	for _, rule := range sparqlCompatibilityRules {
		scanned := code
		if rule.inLiterals {
			scanned = verbose
		}
		if !rule.pattern.MatchString(scanned) {
			continue
		}
		seen[rule.construct] = true
		issues = append(issues, SPARQLIssue{
			Construct: rule.construct,
			Message:   rule.message,
			DocURL:    rule.doc,
		})
	}

	for _, match := range aggregateRebindPattern.FindAllStringSubmatch(code, -1) {
		if match[1] != match[2] || seen["repeated variable binding"] {
			continue
		}
		seen["repeated variable binding"] = true
		issues = append(issues, SPARQLIssue{
			Construct: "repeated variable binding",
			Message:   "SPARQL 1.1 forbids binding ?" + match[1] + " twice in one scope, as in " + match[0] + "; use two names or a subquery",
			DocURL:    wdqsCompareGuide + "#Variable_binding_and_reuse",
		})
	}

	return issues
}
