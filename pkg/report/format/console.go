// Package format provides console rendering utilities for dependency reports.
// It adapts column widths to the terminal and supports color and truncation.
package format

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/greg-hellings/devdashboard/pkg/report"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"
)

// ConsoleFormatter renders a dependency Report in a terminal-friendly table
// that attempts to adapt to the current console width. It replaces the
// previous ad-hoc printf formatting logic.
type ConsoleFormatter struct {
	// MaxPackageColWidth constrains the package name column. If 0, a dynamic
	// width is chosen based on terminal width (with a sane minimum).
	MaxPackageColWidth int

	// MaxRepoColWidth constrains each repository column (version cells).
	// If 0, a dynamic width is chosen. Set this higher if versions frequently
	// include long qualifiers (e.g. pre-release tags).
	MaxRepoColWidth int

	// EnableColors toggles ANSI color output for status cells.
	EnableColors bool
}

// NewConsoleFormatter creates a formatter with sensible defaults.
func NewConsoleFormatter() *ConsoleFormatter {
	return &ConsoleFormatter{
		MaxPackageColWidth: 0,
		MaxRepoColWidth:    0,
		EnableColors:       true,
	}
}

// Render writes the formatted report to writer.
func (f *ConsoleFormatter) Render(rpt *report.Report, writer io.Writer) error {
	if rpt == nil {
		return fmt.Errorf("nil report")
	}

	tw := table.NewWriter()
	tw.SetOutputMirror(writer)
	tw.SetStyle(table.StyleRounded)
	tw.Style().Options.SeparateRows = false
	tw.Style().Options.SeparateColumns = false
	tw.Style().Options.DrawBorder = true

	// Header row: Package + each repository
	header := table.Row{"Package"}
	for _, repo := range rpt.Repositories {
		header = append(header, repo.GetRepoIdentifier())
	}
	tw.AppendHeader(header)

	// Determine dynamic column widths
	colConfigs := f.buildColumnConfig(rpt, writer)
	if len(colConfigs) > 0 {
		tw.SetColumnConfigs(colConfigs)
	}

	// Sort packages for deterministic output
	pkgs := append([]string(nil), rpt.Packages...)
	sort.Strings(pkgs)

	for _, pkg := range pkgs {
		row := table.Row{pkg}
		for _, repo := range rpt.Repositories {
			cell := f.versionCell(&repo, pkg)
			row = append(row, cell)
		}
		tw.AppendRow(row)
	}

	// Render the table
	tw.Render()

	// Summary / errors
	successCount := 0
	for _, rr := range rpt.Repositories {
		if rr.Error == nil {
			successCount++
		}
	}

	if _, err := fmt.Fprintln(writer); err != nil {
		return fmt.Errorf("failed writing summary spacer newline: %w", err)
	}
	if _, err := fmt.Fprintf(writer, "Summary:\n"); err != nil {
		return fmt.Errorf("failed writing summary header: %w", err)
	}
	if _, err := fmt.Fprintf(writer, "  Repositories analyzed: %d/%d successful\n", successCount, len(rpt.Repositories)); err != nil {
		return fmt.Errorf("failed writing repositories analyzed line: %w", err)
	}
	if _, err := fmt.Fprintf(writer, "  Packages tracked: %d\n", len(rpt.Packages)); err != nil {
		return fmt.Errorf("failed writing packages tracked line: %w", err)
	}

	if rpt.HasErrors() {
		if _, err := fmt.Fprintln(writer); err != nil {
			return fmt.Errorf("failed writing errors spacer newline: %w", err)
		}
		if _, err := fmt.Fprintf(writer, "Errors:\n"); err != nil {
			return fmt.Errorf("failed writing errors header: %w", err)
		}
		for _, rr := range rpt.Repositories {
			if rr.Error != nil {
				name := rr.GetRepoIdentifier()
				if _, err := fmt.Fprintf(writer, "  %-30s %v\n", name, rr.Error); err != nil {
					return fmt.Errorf("failed writing error line for %s: %w", name, err)
				}
			}
		}
	}

	return nil
}

// versionCell returns the string (with optional color) for a repository/package cell.
func (f *ConsoleFormatter) versionCell(repo *report.RepositoryReport, pkg string) string {
	if repo.Error != nil {
		return f.color("ERROR", text.FgRed)
	}
	ver, ok := repo.Dependencies[pkg]
	if !ok || ver == "" {
		return f.color("—", text.FgHiBlack)
	}
	return ver
}

// buildColumnConfig creates per-column sizing to fit the terminal.
func (f *ConsoleFormatter) buildColumnConfig(rpt *report.Report, w io.Writer) []table.ColumnConfig {
	termWidth := detectTerminalWidth(w)
	if termWidth <= 0 {
		// Fallback: do not constrain if width unknown
		return nil
	}

	// Guard rails
	if termWidth < 60 {
		termWidth = 60
	}

	repoCount := len(rpt.Repositories)
	if repoCount == 0 {
		return nil
	}

	// Choose package column width
	pkgColWidth := f.MaxPackageColWidth
	if pkgColWidth <= 0 {
		pkgColWidth = dynamicPackageWidth(rpt, termWidth, repoCount)
		if pkgColWidth < 15 {
			pkgColWidth = 15
		}
	}

	repoColWidth := f.MaxRepoColWidth
	if repoColWidth <= 0 {
		// Rough allocation: leave space for borders & package col
		remaining := termWidth - pkgColWidth - 3 /* separators & borders fudge */
		per := remaining / repoCount
		if per < 8 {
			per = 8
		}
		if per > 24 {
			per = 24
		}
		repoColWidth = per
	}

	configs := []table.ColumnConfig{
		{
			Number:      1,
			WidthMax:    pkgColWidth,
			WidthMin:    minInt(10, pkgColWidth),
			Transformer: truncTransformer(pkgColWidth),
		},
	}

	// Columns are 1-based; repository columns start at 2
	for i := 0; i < repoCount; i++ {
		configs = append(configs, table.ColumnConfig{
			Number:      i + 2,
			WidthMax:    repoColWidth,
			WidthMin:    minInt(5, repoColWidth),
			Transformer: truncTransformer(repoColWidth),
		})
	}

	return configs
}

// dynamicPackageWidth estimates a good package column width.
func dynamicPackageWidth(rpt *report.Report, termWidth, repoCount int) int {
	if repoCount == 0 {
		return termWidth
	}

	// Compute max observed package length (capped)
	maxPkgLen := 0
	for _, p := range rpt.Packages {
		l := utf8.RuneCountInString(p)
		if l > maxPkgLen {
			maxPkgLen = l
		}
		if maxPkgLen >= 50 { // upper bound
			break
		}
	}

	// Reserve minimum space for repo columns
	minPerRepo := 8
	reserved := repoCount * minPerRepo
	available := termWidth - reserved - 3
	if available < 15 {
		available = 15
	}
	if maxPkgLen > available {
		return available
	}
	return maxPkgLen
}

// detectTerminalWidth attempts to get terminal width if writer is a file (stdout/stderr).
func detectTerminalWidth(w io.Writer) int {
	if f, ok := w.(*os.File); ok {
		if width, _, err := term.GetSize(int(f.Fd())); err == nil {
			return width
		}
	}
	// Try stdout as fallback
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		return width
	}
	return -1
}

// truncTransformer returns a text.Transformer to ellipsize overly wide cells.
func truncTransformer(max int) text.Transformer {
	return func(val interface{}) string {
		s := fmt.Sprint(val)
		if runeLen := utf8.RuneCountInString(s); runeLen > max {
			if max <= 1 {
				return "…"
			}
			return truncateRunes(s, max)
		}
		return s
	}
}

// truncateRunes truncates a string to (max) runes with ellipsis.
func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	var b strings.Builder
	count := 0
	for _, r := range s {
		if count >= max-1 {
			break
		}
		b.WriteRune(r)
		count++
	}
	b.WriteRune('…')
	return b.String()
}

func (f *ConsoleFormatter) color(s string, c text.Color) string {
	if !f.EnableColors {
		return s
	}
	return text.Colors{c}.Sprint(s)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RenderConsole renders a Report to the provided writer using the console formatter.

// RenderConsole renders the provided Report to the writer using the default console formatter.
func RenderConsole(rpt *report.Report, w io.Writer) error {
	return NewConsoleFormatter().Render(rpt, w)
}
