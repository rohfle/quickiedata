package quickiedata

import (
	"errors"
	"fmt"
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
			statement, err := renderSPARQLStatement(name, query.Variables[name])
			if err != nil {
				return "", err
			}
			statements = append(statements, statement)
		}
		queryText = insertStatementsInWhere(queryText, strings.Join(statements, " "))
	}

	// The fields own these clauses. A template that carries its own OFFSET or
	// LIMIT gets a second one, so leave them out of the template.
	if query.Offset >= 0 {
		queryText += fmt.Sprintf(" OFFSET %d", query.Offset)
	}
	if query.Limit > 0 {
		queryText += fmt.Sprintf(" LIMIT %d", query.Limit)
	}
	return queryText, nil
}

// renderSPARQLStatement renders the terms bound to one variable: a single term
// is a BIND, a list is a VALUES clause, and an empty list is a VALUES clause
// that matches nothing.
func renderSPARQLStatement(name string, terms []SPARQLTerm) (string, error) {
	if !ValidSPARQLVariableName.MatchString(name) {
		return "", fmt.Errorf("invalid sparql variable name '%s'", name)
	}

	rendered := make([]string, 0, len(terms))
	for _, term := range terms {
		if term == nil {
			return "", fmt.Errorf("nil sparql term for ?%s", name)
		}
		text, err := term.SPARQLTerm()
		if err != nil {
			return "", err
		}
		rendered = append(rendered, text)
	}

	if len(rendered) == 1 {
		return fmt.Sprintf(`BIND( %s as ?%s)`, rendered[0], name), nil
	}
	return fmt.Sprintf(`VALUES ?%s { %s }`, name, strings.Join(rendered, " ")), nil
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
