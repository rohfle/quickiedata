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

// variableStatement is one variable's rendered statement, ready to insert.
type variableStatement struct {
	name      string
	statement string
}

func RenderSPARQLQuery(query *SPARQLQuery) (string, error) {
	if query == nil || len(query.Template) == 0 {
		return "", errors.New("sparql query is empty")
	}

	queryText := cleanupSPARQL(query.Template)
	if len(query.Variables) > 0 {
		var statements []variableStatement
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
			statements = append(statements, variableStatement{name: name, statement: statement})
		}
		var err error
		queryText, err = injectVariableStatements(queryText, statements)
		if err != nil {
			return "", err
		}
	}

	// The fields own these clauses. A template that carries its own OFFSET or
	// LIMIT gets a second one, so leave them out of the template.
	if query.Offset >= 0 {
		queryText += fmt.Sprintf(" OFFSET %d", query.Offset)
	}
	if query.Limit > 0 {
		queryText += fmt.Sprintf(" LIMIT %d", query.Limit)
	}
	// The rendered query is what the endpoint receives, so a rejected query can
	// be reproduced without rebuilding the template by hand.
	DebugLog.Printf("sparql query: %s", queryText)
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

// ErrNoInjectionPoint reports a query that has variables but no group pattern
// to bind them in.
var ErrNoInjectionPoint = errors.New("no group graph pattern to bind the variables in")

// injectVariableStatements inserts each variable's statement into the group
// patterns that use it: all of them at the outer group, and the ones whose name
// appears inside a subquery into that subquery as well.
func injectVariableStatements(query string, statements []variableStatement) (string, error) {
	blanked := stripSPARQLCode(query)
	if len(blanked) != len(query) {
		return "", errors.New("sparql query scanner changed the query length")
	}

	outer, subqueries, err := queryScopes(blanked)
	if err != nil {
		return "", err
	}
	// The scanner searches this view, not the query: a statement in an
	// unexpected group is explained here.
	DebugLog.Printf("sparql scanner view (literals blanked): %s", blanked)

	// One block per insertion point: the outer group takes every statement, a
	// subquery takes the statements it uses.
	blocks := map[int]string{outer: joinVariableStatements(statements, nil)}
	for _, scope := range subqueries {
		if _, ok := blocks[scope.insertAt]; ok {
			continue
		}
		block := joinVariableStatements(statements, func(name string) bool {
			return containsVariable(blanked[scope.start:scope.end], name)
		})
		if block != "" {
			blocks[scope.insertAt] = block
		}
	}

	offsets := make([]int, 0, len(blocks))
	for at := range blocks {
		offsets = append(offsets, at)
	}
	// insert back to front so the earlier offsets stay valid
	slices.SortFunc(offsets, func(a, b int) int { return b - a })
	for _, at := range offsets {
		query = query[:at] + " " + blocks[at] + " " + query[at:]
	}
	return query, nil
}

// joinVariableStatements joins the statements keep accepts, in the given order.
func joinVariableStatements(statements []variableStatement, keep func(name string) bool) string {
	parts := make([]string, 0, len(statements))
	for _, statement := range statements {
		if keep == nil || keep(statement.name) {
			parts = append(parts, statement.statement)
		}
	}
	return strings.Join(parts, " ")
}

// containsVariable reports whether code uses ?name or $name as a variable.
func containsVariable(code string, name string) bool {
	for _, prefix := range []string{"?", "$"} {
		needle := prefix + name
		for i := 0; ; {
			match := strings.Index(code[i:], needle)
			if match < 0 {
				break
			}
			end := i + match + len(needle)
			if end == len(code) || !nameByte(code[end]) {
				return true
			}
			i += match + 1
		}
	}
	return false
}

// scope is one group pattern that can receive statements.
type scope struct {
	insertAt int // index just past the '{' of the group
	start    int // first byte of the subquery, its SELECT clause
	end      int // index of the '}' that closes the subquery
}

// queryScopes returns the index just past the '{' of the outer group pattern
// and the scopes of the subqueries, in the order they appear.
func queryScopes(blanked string) (outer int, subqueries []scope, err error) {
	var topBraces []int
	whereBrace := -1
	braceDepth, parenDepth := 0, 0

	for i := 0; i < len(blanked); i++ {
		switch blanked[i] {
		case '(':
			parenDepth++
		case ')':
			parenDepth--
		case '<':
			// an IRI can hold a brace, a comparison operator cannot
			if end, ok := iriEnd(blanked, i); ok {
				i = end
			}
		case '{':
			if braceDepth == 0 && parenDepth == 0 {
				topBraces = append(topBraces, i)
			}
			if sel := nextNonSpace(blanked, i+1); keywordAt(blanked, sel, "SELECT") {
				group := nextBraceAtParenZero(blanked, sel)
				if group < 0 {
					return 0, nil, errUnbalancedBraces
				}
				end := matchingBrace(blanked, group)
				if end < 0 {
					return 0, nil, errUnbalancedBraces
				}
				subqueries = append(subqueries, scope{insertAt: group + 1, start: sel, end: end})
			}
			braceDepth++
		case '}':
			braceDepth--
		}

		if braceDepth == 0 && parenDepth == 0 && whereBrace < 0 && keywordAt(blanked, i, "WHERE") {
			if brace := nextNonSpace(blanked, i+len("WHERE")); brace < len(blanked) && blanked[brace] == '{' {
				whereBrace = brace
			}
		}
	}

	form := ""
	pos := skipPrologue(blanked)
	for _, keyword := range []string{"SELECT", "ASK", "CONSTRUCT", "DESCRIBE"} {
		if keywordAt(blanked, pos, keyword) {
			form = keyword
			break
		}
	}

	found := true
	switch {
	case whereBrace >= 0:
		outer = whereBrace + 1
	case form == "SELECT" || form == "ASK":
		found = len(topBraces) > 0
		if found {
			outer = topBraces[0] + 1
		}
	case form == "CONSTRUCT":
		// the template comes first, the group pattern after it
		found = len(topBraces) > 1
		if found {
			outer = topBraces[1] + 1
		}
	default:
		found = false
	}
	if !found {
		return 0, nil, fmt.Errorf("%w: add a WHERE clause or a SELECT, ASK or CONSTRUCT group", ErrNoInjectionPoint)
	}
	return outer, subqueries, nil
}

// errUnbalancedBraces reports a query whose braces do not pair up.
var errUnbalancedBraces = errors.New("unbalanced braces in the query")

// nextBraceAtParenZero returns the index of the first '{' at or after i that is
// outside parentheses, or -1. It is the group brace of a SELECT clause.
func nextBraceAtParenZero(code string, i int) int {
	depth := 0
	for ; i < len(code); i++ {
		switch code[i] {
		case '(':
			depth++
		case ')':
			depth--
		case '<':
			if end, ok := iriEnd(code, i); ok {
				i = end
			}
		case '{':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// matchingBrace returns the index of the '}' that closes the '{' at open,
// skipping IRIs, or -1 when the braces are unbalanced.
func matchingBrace(code string, open int) int {
	depth := 1
	for i := open + 1; i < len(code); i++ {
		switch code[i] {
		case '<':
			if end, ok := iriEnd(code, i); ok {
				i = end
			}
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// skipPrologue returns the index of the first keyword after the PREFIX and
// BASE declarations, handling an empty prefix label.
func skipPrologue(code string) int {
	pos := 0
	for {
		pos = nextNonSpace(code, pos)
		switch {
		case keywordAt(code, pos, "PREFIX"):
			pos = nextNonSpace(code, pos+len("PREFIX"))
			for pos < len(code) && (nameByte(code[pos]) || code[pos] == '-' || code[pos] == '.') {
				pos++
			}
			if pos >= len(code) || code[pos] != ':' {
				return pos
			}
			pos = nextNonSpace(code, pos+1)
			if pos >= len(code) || code[pos] != '<' {
				return pos
			}
			end, ok := iriEnd(code, pos)
			if !ok {
				return pos
			}
			pos = end + 1
		case keywordAt(code, pos, "BASE"):
			pos = nextNonSpace(code, pos+len("BASE"))
			if pos >= len(code) || code[pos] != '<' {
				return pos
			}
			end, ok := iriEnd(code, pos)
			if !ok {
				return pos
			}
			pos = end + 1
		default:
			return pos
		}
	}
}

// keywordAt reports whether code[i:] starts with keyword, case-insensitively,
// as a whole token: the byte before it is not part of a name, a prefixed name
// or a variable, and the byte after it does not continue a name or prefix.
func keywordAt(code string, i int, keyword string) bool {
	if len(code)-i < len(keyword) || !strings.EqualFold(code[i:i+len(keyword)], keyword) {
		return false
	}
	if i > 0 {
		switch before := code[i-1]; {
		case nameByte(before), before == '-', before == '.', before == ':', before == '?', before == '$', before == '@':
			return false
		}
	}
	if end := i + len(keyword); end < len(code) {
		switch after := code[end]; {
		case nameByte(after), after == '-', after == '.', after == ':':
			return false
		}
	}
	return true
}

// nextNonSpace returns the index of the first byte at or after i that is not a
// space or a tab, or len(code).
func nextNonSpace(code string, i int) int {
	for i < len(code) && (code[i] == ' ' || code[i] == '\t') {
		i++
	}
	return i
}

// nameByte reports whether b continues a SPARQL name.
func nameByte(b byte) bool {
	return b == '_' || b >= 0x80 || ('0' <= b && b <= '9') || ('A' <= b && b <= 'Z') || ('a' <= b && b <= 'z')
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
