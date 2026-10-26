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

func parseArray(val string) ([]string, error) {
	var rawArr []any
	if err := json.Unmarshal([]byte(val), &rawArr); err != nil {
		return nil, err
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
			query := quickiedata.NewSPARQLQuery()
			query.Offset = int64(offset)
			query.Limit = int64(limit)

			var queryText string
			var varArgs []string

			if len(args) > 0 && !strings.Contains(args[0], "=") {
				data, err := os.ReadFile(args[0])
				if err != nil {
					return fmt.Errorf("failed to read query file: %w", err)
				}
				queryText = string(data)
				varArgs = args[1:]
			} else {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read query from stdin: %w", err)
				}
				queryText = string(data)
				varArgs = args
			}

			for _, arg := range varArgs {
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid var: %s (must be key=value)", arg)
				}
				key := parts[0]
				val := strings.TrimSpace(parts[1])
				switch {
				case strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]"):
					parsedVals, err := parseArray(val)
					if err != nil {
						return fmt.Errorf("invalid array syntax for %s: %v", key, err)
					}
					// The elements are plain strings: an id inside an array is
					// not detected the way a single value below is.
					query.Variables.Set(key, quickiedata.Strings(parsedVals...)...)
				case quickiedata.IsEntityID(val):
					query.Variables.Set(key, quickiedata.WikidataID("wd:"+val))
				default:
					query.Variables.Set(key, quickiedata.String(val))
				}
			}

			query.Template = queryText
			options := quickiedata.NewSPARQLQueryOptions()
			resp, err := wd.SPARQLQuerySimple(ctx, query, options)
			if err != nil {
				return fmt.Errorf("sparql request failed:\nquery:\n  %s\noptions: %+v\nerror: %w",
					strings.ReplaceAll(queryText, "\n", "\n  "),
					options,
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
			var data []byte
			var err error
			if len(args) == 1 && args[0] != "-" {
				data, err = os.ReadFile(args[0])
				if err != nil {
					return fmt.Errorf("failed to read query file: %w", err)
				}
			} else {
				data, err = io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read query from stdin: %w", err)
				}
			}

			issues := quickiedata.SPARQLCompatibilityIssues(string(data))
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
