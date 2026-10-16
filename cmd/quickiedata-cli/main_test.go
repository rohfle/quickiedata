package main

import (
	"errors"
	"os/exec"
	"path/filepath"
	"testing"
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
		{"too many arguments", []string{"get", "Q1", "Q42"}, 1},
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
