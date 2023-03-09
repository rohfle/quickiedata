package quickiedata

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

type WikidataID string

var VALID_SPARQL_WIKIDATA_ID = regexp.MustCompile(`^[a-z]+:[PLSFQ][1-9]\d*$`)
var VALID_SPARQL_VARIABLE_NAME = regexp.MustCompile(`^[A-Za-z_]\w*$`)

func RenderSPARQLQuery(baseQuery string, options *GetSPARQLQueryOptions) (string, error) {
	query := cleanupSPARQL(baseQuery)
	if len(options.Variables) > 0 {
		var statements []string
		for name, value := range options.Variables {
			statement, err := renderSPARQLStatement(name, value)
			if err != nil {
				return "", err
			}
			statements = append(statements, statement)
		}
		query = insertStatementsInWhere(query, strings.Join(statements, " "))
	}

	lastIndexCurly := strings.LastIndex(query, "}")
	trailingOptions := ""
	if lastIndexCurly >= 0 {
		trailingOptions = strings.ToLower(query[lastIndexCurly:])
	}
	// probably should replace the limit and offset in the query or error
	// but for now this is good enough
	if options.Offset >= 0 && strings.Index(trailingOptions, "offset ") <= 0 {
		query += fmt.Sprintf(" OFFSET %d", options.Offset)
	}
	if options.Limit > 0 && strings.Index(trailingOptions, "limit ") <= 0 {
		query += fmt.Sprintf(" LIMIT %d", options.Limit)
	}
	return query, nil
}

func renderSPARQLStatement(name string, value interface{}) (string, error) {
	// validate key is valid
	if !VALID_SPARQL_VARIABLE_NAME.MatchString(name) {
		return "", fmt.Errorf("invalid sparql variable name '%s'", name)
	}

	switch v := value.(type) {
	case string:
		v = strings.ReplaceAll(v, "\"", "\\\"")
		return fmt.Sprintf(`BIND( """%s""" as ?%s)`, v, name), nil
	case []string:
		var escaped []string
		for _, s := range v {
			e := fmt.Sprintf(`"""%s"""`, strings.ReplaceAll(s, "\"", "\\\""))
			escaped = append(escaped, e)
		}
		return fmt.Sprintf(`VALUES ?%s { %s }`, name, strings.Join(escaped, " ")), nil
	case WikidataID:
		if !VALID_SPARQL_WIKIDATA_ID.MatchString(string(v)) {
			return "", fmt.Errorf("invalid wikidata reference '%s'", v)
		}
		return fmt.Sprintf(`BIND( %s as ?%s)`, v, name), nil
	case []WikidataID:
		var values []string
		for _, wid := range v {
			if !VALID_SPARQL_WIKIDATA_ID.MatchString(string(wid)) {
				return "", fmt.Errorf("invalid wikidata reference '%s'", v)
			}
			values = append(values, string(wid))
		}
		return fmt.Sprintf(`VALUES ?%s { %s }`, name, strings.Join(values, " ")), nil
	default:
		return "", fmt.Errorf("unhandled %s datatype", reflect.TypeOf(v))
	}
}

func insertStatementsInWhere(query string, statementBlock string) string {
	// search for WHERE\s+{
	clauses := regexp.MustCompile(`(?i)WHERE\s*{`).FindAllStringIndex(query, -1)

	output := ""
	lastIndex := 0
	for _, match := range clauses {
		output += query[lastIndex:match[1]]
		output += " " + statementBlock + " "
		lastIndex = match[1]
	}
	return output + query[lastIndex:]
}

func cleanupSPARQL(query string) string {
	lines := strings.Split(query, "\n")
	var outlines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line := findAndRemoveComment(line)
		if line != "" {
			outlines = append(outlines, line)
		}
	}
	return strings.Join(outlines, " ")
}

func findAndRemoveComment(line string) string {
	inQuotes := false
	for cidx, char := range line {
		switch char {
		case '\'', '"':
			inQuotes = !inQuotes
		case '#':
			if !inQuotes {
				// comment found, remove rest of line
				line = line[:cidx]
				line = strings.TrimSpace(line)
				return line
			}
		}
	}
	return line
}
