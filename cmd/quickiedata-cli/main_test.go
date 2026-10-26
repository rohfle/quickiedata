package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rohfle/quickiedata"
	"github.com/spf13/cobra"
)

// buildCLI builds the command into a temporary directory, so the test can
// check the status code that a shell sees.
func buildCLI(t *testing.T) string {
	t.Helper()

	binary := filepath.Join(t.TempDir(), "quickiedata-cli")
	out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %s\n%s", err, out)
	}
	return binary
}

func exitCode(t *testing.T, binary string, args ...string) int {
	t.Helper()

	err := exec.Command(binary, args...).Run()
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("running %s %v: %s", binary, args, err)
	}
	return exitErr.ExitCode()
}

// A failure must not look like a success to a script or a pipeline.
func TestExitCode(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the cli")
	}
	binary := buildCLI(t)

	tests := []struct {
		name string
		args []string
		want int
	}{
		{"invalid entity id", []string{"get", "NOTANID"}, 1},
		{"get without an id", []string{"get"}, 1},
		{"search with two terms", []string{"search", "cat", "dog"}, 1},
		{"unknown command", []string{"bogus"}, 1},
		{"help", []string{"--help"}, 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := exitCode(t, binary, test.args...); got != test.want {
				t.Errorf("exit code = %d, want %d", got, test.want)
			}
		})
	}
}

// parseVariables classifies every value, including array elements, in one
// place: entity id, then predicate, then string.
func TestParseVariables(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		key     string
		want    []string
		wantErr string
	}{
		{
			name: "string",
			args: []string{"name=Oscar"},
			key:  "name",
			want: []string{`"""Oscar"""`},
		},
		{
			name: "bare entity id",
			args: []string{"item=Q42"},
			key:  "item",
			want: []string{"<http://www.wikidata.org/entity/Q42>"},
		},
		{
			name: "prefixed entity id",
			args: []string{"item=wd:Q42"},
			key:  "item",
			want: []string{"<http://www.wikidata.org/entity/Q42>"},
		},
		{
			name: "predicate",
			args: []string{"p=wdt:P31"},
			key:  "p",
			want: []string{"<http://www.wikidata.org/prop/direct/P31>"},
		},
		{
			name: "array of ids",
			args: []string{`ids=["Q42","Q1"]`},
			key:  "ids",
			want: []string{
				"<http://www.wikidata.org/entity/Q42>",
				"<http://www.wikidata.org/entity/Q1>",
			},
		},
		{
			name: "bare array of ids",
			args: []string{"ids=[Q42,Q1]"},
			key:  "ids",
			want: []string{
				"<http://www.wikidata.org/entity/Q42>",
				"<http://www.wikidata.org/entity/Q1>",
			},
		},
		{
			name: "array of strings",
			args: []string{`names=["Oscar","Felix"]`},
			key:  "names",
			want: []string{`"""Oscar"""`, `"""Felix"""`},
		},
		{
			name:    "argument without a value",
			args:    []string{"name"},
			wantErr: "must be key=value",
		},
		{
			name:    "unterminated array",
			args:    []string{`ids=[Q42`},
			key:     "ids",
			wantErr: "invalid array syntax",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			variables, err := parseVariables(test.args)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseVariables: %s", err)
			}

			var got []string
			for _, term := range variables.Get(test.key) {
				rendered, err := term.SPARQLTerm()
				if err != nil {
					t.Fatalf("SPARQLTerm: %s", err)
				}
				got = append(got, rendered)
			}
			if len(got) != len(test.want) {
				t.Fatalf("terms = %v, want %v", got, test.want)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Errorf("term %d = %q, want %q", i, got[i], test.want[i])
				}
			}
		})
	}
}

// readQueryInput treats the first argument as a file unless it is a key=value.
func TestReadQueryInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "query.sparql")
	if err := os.WriteFile(path, []byte("SELECT ?x WHERE { ?x ?p ?o }"), 0o644); err != nil {
		t.Fatalf("WriteFile: %s", err)
	}

	queryText, varArgs, err := readQueryInput([]string{path, "name=Oscar"})
	if err != nil {
		t.Fatalf("readQueryInput: %s", err)
	}
	if !strings.Contains(queryText, "SELECT ?x") {
		t.Errorf("query = %q, want the file contents", queryText)
	}
	if len(varArgs) != 1 || varArgs[0] != "name=Oscar" {
		t.Errorf("varArgs = %v, want [name=Oscar]", varArgs)
	}
}

// A paging flag reaches the query only when the caller set it, so a template
// with its own LIMIT is not given a second one.
func TestApplyPagination(t *testing.T) {
	tests := []struct {
		name       string
		setFlags   map[string]string
		wantOffset int64
		wantLimit  int64
	}{
		{"nothing set", nil, -1, -1},
		{"limit set", map[string]string{"limit": "3"}, -1, 3},
		{"offset set", map[string]string{"offset": "5"}, 5, -1},
		{"both set", map[string]string{"offset": "5", "limit": "3"}, 5, 3},
		{"limit zero", map[string]string{"limit": "0"}, -1, 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			var offset, limit int
			cmd.Flags().IntVar(&offset, "offset", 0, "")
			cmd.Flags().IntVar(&limit, "limit", 0, "")
			for name, value := range test.setFlags {
				if err := cmd.Flags().Set(name, value); err != nil {
					t.Fatalf("Set(%s): %s", name, err)
				}
			}

			query := quickiedata.NewSPARQLQuery()
			applyPagination(cmd, query, offset, limit)

			if query.Offset != test.wantOffset {
				t.Errorf("Offset = %d, want %d", query.Offset, test.wantOffset)
			}
			if query.Limit != test.wantLimit {
				t.Errorf("Limit = %d, want %d", query.Limit, test.wantLimit)
			}
		})
	}
}
