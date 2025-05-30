package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rohfle/nicehttp"
	"github.com/rohfle/quickiedata"
	"github.com/spf13/cobra"
)

const UserAgent = "quickiedata-cli/0.1"

func parseArray(val string) ([]string, error) {
	var rawArr []interface{}
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
	wd := quickiedata.NewClient(&nicehttp.Settings{
		DefaultHeaders: http.Header{
			"User-Agent": {UserAgent},
		},
		RequestInterval: 1 * time.Second,
		Backoff:         1 * time.Second,
		MaxBackoff:      30 * time.Second,
		MaxTries:        5,
		MaxConnsPerHost: 1,
	})

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
				if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
					parsedVals, err := parseArray(val)
					if err != nil {
						return fmt.Errorf("invalid array syntax for %s: %v", key, err)
					}
					query.Variables[key] = parsedVals
				} else if quickiedata.IsEntityID(val) {
					query.Variables[key] = quickiedata.WikidataID("wd:" + val)
				} else {
					query.Variables[key] = val
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
				return fmt.Errorf("failed while rendering results for %q: %w", query, err)
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
		Use:   "search [term]",
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
		Use:   "get [id1 id2 ...]",
		Short: "Fetch entity data for given IDs",
		Args:  cobra.ExactArgs(1),
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

	rootCmd.AddCommand(queryCmd, searchCmd, getCmd)

	rootCmd.Execute()
}
