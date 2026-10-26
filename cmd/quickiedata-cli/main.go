package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rohfle/quickiedata"
	"github.com/spf13/cobra"
)

var UserAgent = fmt.Sprintf("quickiedata-cli/%s (https://github.com/rohfle/quickiedata)", quickiedata.Version())

// parseArray parses a [...] list, either as JSON (["a","b"]) or as a bare
// comma separated list ([a,b]).
func parseArray(val string) ([]string, error) {
	var rawArr []any
	if err := json.Unmarshal([]byte(val), &rawArr); err != nil {
		for _, element := range strings.Split(strings.TrimSuffix(strings.TrimPrefix(val, "["), "]"), ",") {
			element = strings.TrimSpace(element)
			if element == "" {
				continue
			}
			rawArr = append(rawArr, strings.Trim(element, `"'`))
		}
	}
	strArr := make([]string, 0, len(rawArr))
	for _, item := range rawArr {
		s, ok := item.(string)
		if !ok {
			s = fmt.Sprintf("%v", item)
		}
		strArr = append(strArr, s)
	}
	return strArr, nil
}

func main() {
	ctx := context.Background()
	wd, err := quickiedata.NewClient(UserAgent, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create quickiedata client: %v\n", err)
		os.Exit(1)
	}

	var language string

	rootCmd := &cobra.Command{
		Use: "quickiedata-cli",
	}

	rootCmd.PersistentFlags().StringVarP(&language, "language", "l", "en", "Language code for labels and descriptions")

	var offset int
	var limit int

	// Query command
	var queryCmd = &cobra.Command{
		Use:   "query [file] [key=value]...",
		Short: "Execute a SPARQL query with variables",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			queryText, varArgs, err := readQueryInput(args)
			if err != nil {
				return err
			}
			variables, err := parseVariables(varArgs)
			if err != nil {
				return err
			}

			query := quickiedata.NewSPARQLQuery()
			query.Offset = int64(offset)
			query.Limit = int64(limit)
			query.Template = queryText
			query.Variables = variables

			resp, err := wd.SPARQLQuerySimple(ctx, query)
			if err != nil {
				return fmt.Errorf("sparql request failed:\nquery:\n  %s\nerror: %w",
					strings.ReplaceAll(queryText, "\n", "\n  "),
					err,
				)
			}

			if len(resp.Results) == 0 {
				fmt.Println("no results")
				return nil
			}

			data, err := json.MarshalIndent(resp.Results, "", "  ")
			if err != nil {
				return fmt.Errorf("failed while rendering results for %+v: %w", query, err)
			}
			fmt.Println(string(data))
			return nil
		},
	}
	queryCmd.Flags().IntVar(&offset, "offset", 0, "Offset for results")
	queryCmd.Flags().IntVar(&limit, "limit", 10, "Limit for results")
	queryCmd.SilenceUsage = true

	// Search command
	var entityType string
	var searchCmd = &cobra.Command{
		Use:   "search term",
		Short: "Search for entities by term",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			options := quickiedata.NewSearchEntitiesOptions()
			options.Language = language
			options.EntityType = entityType
			options.Offset = int64(offset)
			options.Limit = int64(limit)
			result, err := wd.SearchEntities(ctx, query, options)
			if err != nil {
				return fmt.Errorf("failed while searching for %q: %w", query, err)
			}
			if len(result) == 0 {
				fmt.Println("no results")
				return nil
			}
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("failed while rendering results for %q: %w", query, err)
			}
			fmt.Println(string(data))
			return nil
		},
	}
	searchCmd.Flags().StringVar(&entityType, "type", "item", "Entity type to search for (item or property)")
	searchCmd.Flags().IntVar(&offset, "offset", 0, "Offset for results")
	searchCmd.Flags().IntVar(&limit, "limit", 10, "Limit for results")
	searchCmd.SilenceUsage = true

	var sitefilter string
	var props string
	var rawMode bool
	var getCmd = &cobra.Command{
		Use:   "get id [id ...]",
		Short: "Fetch entity data for given IDs",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wikidataIDs := args
			options := quickiedata.NewGetEntitiesOptions()
			options.Languages = []string{language}
			if sitefilter != "" {
				options.Sitefilter = quickiedata.SplitAndTrim(sitefilter, ",")
			}
			if props != "" {
				options.Props = quickiedata.SplitAndTrim(props, ",")
			}
			result, err := wd.GetEntities(ctx, wikidataIDs, options)
			if err != nil {
				return fmt.Errorf("failed while retrieving %s: %w", wikidataIDs, err)
			}

			var data []byte
			if rawMode {
				data, err = json.MarshalIndent(result, "", "  ")
			} else {
				simpleResult := result.Simplify()
				data, err = json.MarshalIndent(simpleResult, "", "  ")
			}
			if err != nil {
				return fmt.Errorf("failed while rendering results for %q: %w", wikidataIDs, err)
			}
			fmt.Println(string(data))
			return nil
		},
	}
	getCmd.Flags().StringVar(&sitefilter, "sitefilter", "", "Filter sitelinks by site (e.g. enwiki,enwikiquote)")
	getCmd.Flags().StringVar(&props, "props", "", "Properties to fetch (e.g. labels,descriptions,claims,sitelinks)")
	getCmd.Flags().BoolVar(&rawMode, "raw", false, "Output data without simplification")
	getCmd.SilenceUsage = true

	// Compatibility check command
	var checkCmd = &cobra.Command{
		Use:   "check-query [file]",
		Short: "Report SPARQL constructs that the WDQS v2 backend does not support",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			queryText, err := readQueryText(args)
			if err != nil {
				return err
			}

			issues := quickiedata.SPARQLCompatibilityIssues(queryText)
			if len(issues) == 0 {
				fmt.Println("no WDQS v2 compatibility issues found")
				return nil
			}
			for _, issue := range issues {
				fmt.Printf("%s: %s\n  %s\n", issue.Construct, issue.Message, issue.DocURL)
			}
			return fmt.Errorf("%d construct(s) to rewrite for WDQS v2", len(issues))
		},
	}
	checkCmd.SilenceUsage = true

	rootCmd.AddCommand(queryCmd, searchCmd, getCmd, checkCmd)

	if err := rootCmd.Execute(); err != nil {
		// cobra has already written the error to stderr
		os.Exit(1)
	}
}

// readQueryInput returns the query text and the remaining key=value arguments.
// The first argument is a file path unless it carries "=", in which case the
// query comes from stdin.
func readQueryInput(args []string) (string, []string, error) {
	if len(args) > 0 && !strings.Contains(args[0], "=") {
		queryText, err := readQueryText(args[:1])
		return queryText, args[1:], err
	}
	queryText, err := readQueryText(nil)
	return queryText, args, err
}

// readQueryText reads a query from the file in args, or from stdin when there
// is no file or it is "-".
func readQueryText(args []string) (string, error) {
	if len(args) == 1 && args[0] != "-" {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return "", fmt.Errorf("failed to read query file: %w", err)
		}
		return string(data), nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("failed to read query from stdin: %w", err)
	}
	return string(data), nil
}

// parseVariables turns key=value arguments into bound terms. A value is an
// entity id, a predicate, or a string literal, classified element by element
// when the value is an array.
func parseVariables(args []string) (quickiedata.Variables, error) {
	variables := make(quickiedata.Variables)
	for _, arg := range args {
		key, value, ok := strings.Cut(arg, "=")
		if !ok {
			return nil, fmt.Errorf("invalid var: %s (must be key=value)", arg)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if strings.HasPrefix(value, "[") || strings.HasSuffix(value, "]") {
			if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
				return nil, fmt.Errorf("invalid array syntax for %s: %q is not a [...] list", key, value)
			}
			values, err := parseArray(value)
			if err != nil {
				return nil, fmt.Errorf("invalid array syntax for %s: %v", key, err)
			}
			terms := make([]quickiedata.SPARQLTerm, 0, len(values))
			for _, element := range values {
				terms = append(terms, parseTerm(element))
			}
			variables.Set(key, terms...)
			continue
		}
		variables.Set(key, parseTerm(value))
	}
	return variables, nil
}

// parseTerm classifies one value: an entity id (Q42, wd:Q42), a predicate
// (wdt:P31), or a string literal.
func parseTerm(value string) quickiedata.SPARQLTerm {
	if id, err := quickiedata.ParseEntityID(value); err == nil {
		return id
	}
	if predicate, err := quickiedata.ParsePredicate(value); err == nil {
		return predicate
	}
	return quickiedata.String(value)
}
