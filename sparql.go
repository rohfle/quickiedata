package quickiedata

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

type BindingValue struct {
	Value    *string `json:"value"`
	Type     string  `json:"type"`
	DataType string  `json:"datatype,omitempty"`
	Lang     string  `json:"xml:lang,omitempty"`
}

type SimpleBindingValue struct {
	Value any
}

func (s *SimpleBindingValue) ValueAsString() string {
	if s == nil {
		return ""
	}

	casted, ok := s.Value.(string)
	if !ok {
		return ""
	}

	return casted
}

func (s *SimpleBindingValue) ValueAsInteger() *int64 {
	if s == nil {
		return nil
	}

	casted, ok := s.Value.(int64)
	if !ok {
		return nil
	}

	return &casted
}

func (s *SimpleBindingValue) ValueAsBoolean() *bool {
	if s == nil {
		return nil
	}

	casted, ok := s.Value.(bool)
	if !ok {
		return nil
	}

	return &casted
}

func (s *SimpleBindingValue) ValueAsFloat() *float64 {
	if s == nil {
		return nil
	}

	casted, ok := s.Value.(float64)
	if !ok {
		return nil
	}

	return &casted
}

var ValidSPARQLVariableName = regexp.MustCompile(`^[A-Za-z_]\w*$`)

func RenderSPARQLQuery(query *SPARQLQuery) (string, error) {
	if query == nil || len(query.Template) == 0 {
		return "", errors.New("sparql query is empty")
	}

	queryText := cleanupSPARQL(query.Template)
	if len(query.Variables) > 0 {
		var statements []string
		// sort the variables to keep output deterministic
		var keys []string
		for key := range query.Variables {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, name := range keys {
			value := query.Variables[name]
			statement, err := renderSPARQLStatement(name, value)
			if err != nil {
				return "", err
			}
			statements = append(statements, statement)
		}
		queryText = insertStatementsInWhere(queryText, strings.Join(statements, " "))
	}

	lastIndexCurly := strings.LastIndex(queryText, "}")
	trailingOptions := ""
	if lastIndexCurly >= 0 {
		trailingOptions = strings.ToLower(queryText[lastIndexCurly:])
	}
	// probably should replace the limit and offset in the query or error
	// but for now this is good enough
	if query.Offset >= 0 && !strings.Contains(trailingOptions, "offset ") {
		queryText += fmt.Sprintf(" OFFSET %d", query.Offset)
	}
	if query.Limit > 0 && !strings.Contains(trailingOptions, "limit ") {
		queryText += fmt.Sprintf(" LIMIT %d", query.Limit)
	}
	return queryText, nil
}

func renderSPARQLStatement(name string, value any) (string, error) {
	// validate key is valid
	if !ValidSPARQLVariableName.MatchString(name) {
		return "", fmt.Errorf("invalid sparql variable name '%s'", name)
	}

	// An id list is a term list, so one case renders both.
	if ids, ok := value.([]WikidataID); ok {
		terms := make([]SPARQLTerm, 0, len(ids))
		for _, id := range ids {
			terms = append(terms, id)
		}
		value = terms
	}

	switch v := value.(type) {
	case string:
		v = strings.ReplaceAll(v, `\`, `\\`)
		v = strings.ReplaceAll(v, `"`, `\"`)
		return fmt.Sprintf(`BIND( """%s""" as ?%s)`, v, name), nil
	case []string:
		var escaped []string
		for _, s := range v {
			s = strings.ReplaceAll(s, `\`, `\\`)
			s = strings.ReplaceAll(s, `"`, `\"`)
			e := fmt.Sprintf(`"""%s"""`, s)
			escaped = append(escaped, e)
		}
		return fmt.Sprintf(`VALUES ?%s { %s }`, name, strings.Join(escaped, " ")), nil
	case SPARQLTerm:
		term, err := v.SPARQLTerm()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(`BIND( %s as ?%s)`, term, name), nil
	case []SPARQLTerm:
		terms := make([]string, 0, len(v))
		for _, term := range v {
			if term == nil {
				return "", fmt.Errorf("nil sparql term in the list for ?%s", name)
			}
			text, err := term.SPARQLTerm()
			if err != nil {
				return "", err
			}
			terms = append(terms, text)
		}
		return fmt.Sprintf(`VALUES ?%s { %s }`, name, strings.Join(terms, " ")), nil
	case int, int32, int64:
		return fmt.Sprintf(`BIND( %d as ?%s)`, v, name), nil
	default:
		return "", fmt.Errorf("unhandled %s datatype", reflect.TypeOf(v))
	}
}

// whereClausePattern matches the WHERE clause of a query, including the
// opening brace.
var whereClausePattern = regexp.MustCompile(`(?i)WHERE\s*{`)

func insertStatementsInWhere(query string, statementBlock string) string {
	// search for WHERE {
	clauses := whereClausePattern.FindAllStringIndex(query, -1)

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
	lines := strings.Split(stripSPARQLComments(query), "\n")
	var outlines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			outlines = append(outlines, line)
		}
	}
	return strings.Join(outlines, " ")
}

// stripSPARQLComments replaces every comment with one space. A "#" starts a
// comment only when it is outside a string literal and outside an IRI, so a
// "#" in a value or in an IRI stays part of the query.
func stripSPARQLComments(query string) string {
	return stripSPARQL(query, false)
}

// stripSPARQLCode replaces every comment and the contents of every string
// literal with spaces, keeping the query structure and its IRIs. Scanning a
// query for a keyword this way cannot match text inside a literal.
//
//	BIND("SERVICE wikibase:label" AS ?x)  ->  BIND("                      " AS ?x)
func stripSPARQLCode(query string) string {
	return stripSPARQL(query, true)
}

// stripSPARQL removes comments. With blankLiterals every byte inside a string
// literal becomes a space, which is for scanning only: the result is no longer
// a query.
func stripSPARQL(query string, blankLiterals bool) string {
	var out strings.Builder
	out.Grow(len(query))

	// literal returns the byte to keep for text inside a string literal.
	literal := func(char byte) byte {
		if blankLiterals {
			return ' '
		}
		return char
	}

	quote := "" // the string literal that is open, empty when outside one
	inComment := false

	for i := 0; i < len(query); i++ {
		char := query[i]

		if inComment {
			if char == '\n' {
				inComment = false
				out.WriteByte(char)
			}
			continue
		}

		if quote != "" {
			// an escaped character cannot end the string
			if char == '\\' && i+1 < len(query) {
				out.WriteByte(literal(char))
				i++
				out.WriteByte(literal(query[i]))
				continue
			}
			if strings.HasPrefix(query[i:], quote) {
				out.WriteString(quote)
				i += len(quote) - 1
				quote = ""
				continue
			}
			out.WriteByte(literal(char))
			continue
		}

		switch {
		case char == '#':
			inComment = true
			out.WriteByte(' ')
		case char == '<':
			if end, ok := iriEnd(query, i); ok {
				out.WriteString(query[i : end+1])
				i = end
				continue
			}
			out.WriteByte(char)
		case char == '\'' || char == '"':
			quote = string(char)
			// a long string literal has three delimiters
			if strings.HasPrefix(query[i:], quote+quote+quote) {
				quote = quote + quote + quote
			}
			out.WriteString(quote)
			i += len(quote) - 1
		default:
			out.WriteByte(char)
		}
	}

	return out.String()
}

// iriEnd returns the index of the ">" that closes the IRI that starts at
// start. A "<" that is a comparison operator has no such end, and an IRI
// cannot contain a space or a control character.
func iriEnd(query string, start int) (int, bool) {
	for i := start + 1; i < len(query); i++ {
		switch char := query[i]; {
		case char == '>':
			return i, true
		case char <= ' ', char == '<', char == '"', char == '{', char == '}', char == '|', char == '^', char == '`', char == '\\':
			return 0, false
		}
	}
	return 0, false
}
