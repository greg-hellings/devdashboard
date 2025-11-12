// Package main_test contains tests for the DevDashboard CLI entrypoint and its JSON output behavior.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestCLIJSONOutputBasic verifies that invoking the CLI with --format json
// produces valid JSON with the expected shape and content (including errors).
func TestCLIJSONOutputBasic(t *testing.T) {
	cfgPath := writeTempConfig(t, `
providers:
  github:
    default:
      token: ""
    repositories:
      - owner: dummyowner
        repository: dummyrepo
        analyzer: invalidAnalyzerX
        packages:
          - pkgA
          - pkgB
`)

	root := newRootCmd()
	root.SetArgs([]string{
		"dependency-report",
		cfgPath,
		"--format", "json",
		"--json-indent",
	})

	output, err := executeCommand(root)
	if err != nil {
		t.Fatalf("command returned error: %v\nOutput: %s", err, output)
	}

	// Basic JSON validation & structure checks
	var parsed struct {
		Version      string        `json:"cliVersion"`
		GeneratedAt  string        `json:"generatedAt"`
		Repositories []interface{} `json:"repositories"`
		Packages     []string      `json:"packages"`
		Summary      struct {
			RepositoryCount int `json:"repositoryCount"`
			PackageCount    int `json:"packageCount"`
			SuccessCount    int `json:"successCount"`
			ErrorCount      int `json:"errorCount"`
		} `json:"summary"`
		Errors map[string]string `json:"errors"`
	}

	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	if parsed.Summary.RepositoryCount != 1 {
		t.Errorf("expected repositoryCount=1, got %d", parsed.Summary.RepositoryCount)
	}
	if parsed.Summary.PackageCount != 2 {
		t.Errorf("expected packageCount=2, got %d", parsed.Summary.PackageCount)
	}
	if parsed.Summary.ErrorCount != 1 {
		t.Errorf("expected errorCount=1 (invalid analyzer), got %d", parsed.Summary.ErrorCount)
	}
	if parsed.Summary.SuccessCount != 0 {
		t.Errorf("expected successCount=0, got %d", parsed.Summary.SuccessCount)
	}

	// Confirm packages list
	expectContains(t, strings.Join(parsed.Packages, ","), "pkgA", "pkgA missing from packages")
	expectContains(t, strings.Join(parsed.Packages, ","), "pkgB", "pkgB missing from packages")

	// Errors map should contain an entry for dummyowner/dummyrepo
	foundKey := false
	for k := range parsed.Errors {
		if k == "dummyowner/dummyrepo" {
			foundKey = true
			break
		}
	}
	if !foundKey {
		t.Errorf("expected errors map to contain key dummyowner/dummyrepo; keys: %v", keys(parsed.Errors))
	}

	// Pretty-print (indent) check: expect leading newline+two-spaces after an opening brace somewhere
	if !strings.Contains(output, "\n  \"repositories\"") {
		t.Errorf("expected indented JSON output (--json-indent), pattern not found")
	}
}

// TestCLIJSONFailOnError ensures that --fail-on-error causes a non-zero error when
// any repository fails (which will happen with invalid analyzer).
func TestCLIJSONFailOnError(t *testing.T) {
	cfgPath := writeTempConfig(t, `
providers:
  github:
    default:
      token: ""
    repositories:
      - owner: dummyowner
        repository: dummyrepo
        analyzer: invalidAnalyzerX
        packages:
          - pkgA
`)

	root := newRootCmd()
	root.SetArgs([]string{
		"dependency-report",
		cfgPath,
		"--format", "json",
		"--fail-on-error",
	})

	output, err := executeCommand(root)
	if err == nil {
		t.Fatalf("expected command to fail due to --fail-on-error, got success. Output: %s", output)
	}
	if !strings.Contains(err.Error(), "one or more repositories failed") {
		t.Errorf("expected fail-on-error message in error: %v", err)
	}

	// Output should still be valid JSON (partial validation).
	var parsed map[string]interface{}
	if parseErr := json.Unmarshal([]byte(output), &parsed); parseErr != nil {
		t.Fatalf("output was not valid JSON despite error condition: %v\nOutput: %s", parseErr, output)
	}
}

// Helper: write temp config file
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "repos.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

// Helper: execute a Cobra command capturing stdout
func executeCommand(root *cobra.Command) (string, error) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	var execErr error
	done := make(chan struct{})
	go func() {
		execErr = root.Execute()
		if err := w.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "pipe close error: %v\n", err)
		}
		close(done)
	}()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	<-done

	os.Stdout = oldStdout
	return buf.String(), execErr
}

// Helper: minimal contains assertion
func expectContains(t *testing.T, s, substr, msg string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("%s: expected %q to contain %q", msg, s, substr)
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
