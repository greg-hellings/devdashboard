package format

import (
	"bytes"
	"strings"
	"testing"

	"github.com/greg-hellings/devdashboard/pkg/report"
)

// helper to build a sample report
func sampleReport() *report.Report {
	return &report.Report{
		Repositories: []report.RepositoryReport{
			{
				Provider:     "github",
				Owner:        "org1",
				Repository:   "repo1",
				Analyzer:     "poetry",
				Dependencies: map[string]string{"pkgA": "1.2.3", "pkgB": "4.5.6"},
				Error:        nil,
			},
			{
				Provider:     "github",
				Owner:        "org2",
				Repository:   "repo2",
				Analyzer:     "poetry",
				Dependencies: map[string]string{"pkgA": "1.2.4"},
				// Repo has an error -> all cells should render as ERROR regardless of dependency map.
				Error: assertError("dependency scan failed"),
			},
		},
		Packages: []string{"pkgA", "pkgB"},
	}
}

// assertError creates a lightweight error for embedding in reports
type assertError string

func (e assertError) Error() string { return string(e) }

func TestConsoleFormatterBasicRender(t *testing.T) {
	rpt := sampleReport()

	var buf bytes.Buffer
	f := NewConsoleFormatter()
	f.EnableColors = false // deterministic output for assertions

	if err := f.Render(rpt, &buf); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	out := buf.String()

	// Core structural expectations (pivoted layout: repositories = rows, packages = columns)
	expectContains(t, out, "org1/repo1", "repository org1/repo1 missing")
	expectContains(t, out, "org2/repo2", "repository org2/repo2 missing")
	expectContains(t, out, "PKGA", "package header pkgA missing")
	expectContains(t, out, "PKGB", "package header pkgB missing")
	expectContains(t, out, "1.2.3", "version 1.2.3 missing for pkgA in org1/repo1 row")
	// Error repo should not show the version (1.2.4); it should show ERROR.
	expectContains(t, out, "ERROR", "error marker missing for failing repository cells")
	expectContains(t, out, "Repositories analyzed: 1/2 successful", "summary success count mismatch")
	expectContains(t, out, "Packages tracked: 2", "package summary mismatch")

	// Error section details
	expectContains(t, out, "Errors:", "errors section header missing")
	expectContains(t, out, "org2/repo2", "errored repository identifier missing")
	expectContains(t, out, "dependency scan failed", "error message missing")

	// Ensure no ANSI escapes when colors disabled
	if strings.Contains(out, "\x1b[") {
		t.Errorf("unexpected ANSI color sequences found when colors disabled")
	}
}

func TestConsoleFormatterColorsEnabledShowsANSIForError(t *testing.T) {
	rpt := sampleReport()

	var buf bytes.Buffer
	f := NewConsoleFormatter()
	f.EnableColors = true

	if err := f.Render(rpt, &buf); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	out := buf.String()

	// Look for colored ERROR (should contain ANSI ESC)
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("expected ANSI color sequences but none found")
	}

	// Verify ERROR cell appears (even with color codes)
	if !strings.Contains(stripANSI(out), "ERROR") {
		t.Errorf("expected ERROR marker in output (stripANSI)")
	}
}

func TestConsoleFormatterNilReport(t *testing.T) {
	var buf bytes.Buffer
	f := NewConsoleFormatter()
	err := f.Render(nil, &buf)
	if err == nil {
		t.Fatalf("expected error rendering nil report, got nil")
	}
}

func expectContains(t *testing.T, s, substr, msg string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("%s: expected to contain %q\nFull output:\n%s", msg, substr, s)
	}
}

// stripANSI removes ANSI escape sequences for simplified checks.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			// ESC sequences end with 'm' or a letter; simplistic but adequate here
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				inEsc = false
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
