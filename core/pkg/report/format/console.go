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

	"github.com/greg-hellings/devdashboard/core/pkg/report"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"
)

// ConsoleFormatter renders a dependency Report in a terminal-friendly table.
// Pivoted layout (repositories as rows, packages as columns).
type ConsoleFormatter struct {
	// MaxRepoColWidth constrains the repository identifier column width (first column).
	// If 0, width is chosen dynamically.
	MaxRepoColWidth int
	// MaxPackageColWidth constrains each package column width (version cells).
	// If 0, a dynamic width distribution is applied.
	MaxPackageColWidth int
	// EnableColors toggles ANSI color output for status cells.
	EnableColors bool
}

// NewConsoleFormatter creates a formatter with sensible defaults.
func NewConsoleFormatter() *ConsoleFormatter {
	return &ConsoleFormatter{
		MaxRepoColWidth:    0,
		MaxPackageColWidth: 0,
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

	// Header row: Repository + each package
	pkgs := append([]string(nil), rpt.Packages...)
	sort.Strings(pkgs)
	header := table.Row{"Repository"}
	for _, pkg := range pkgs {
		header = append(header, pkg)
	}
	tw.AppendHeader(header)

	// Determine dynamic column widths
	colConfigs := f.buildColumnConfig(rpt, writer, pkgs)
	if len(colConfigs) > 0 {
		tw.SetColumnConfigs(colConfigs)
	}

	// Rows: each repository with versions per package
	for _, repo := range rpt.Repositories {
		row := table.Row{repo.GetRepoIdentifier()}
		for _, pkg := range pkgs {
			row = append(row, f.versionCell(&repo, pkg))
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
func (f *ConsoleFormatter) buildColumnConfig(rpt *report.Report, w io.Writer, pkgs []string) []table.ColumnConfig {
	termWidth := detectTerminalWidth(w)
	if termWidth <= 0 {
		return nil
	}

	if termWidth < 60 {
		termWidth = 60
	}

	packageCount := len(pkgs)
	if packageCount == 0 {
		return nil
	}

	// First column (repository identifier) width
	repoIDColWidth := f.MaxRepoColWidth
	if repoIDColWidth <= 0 {
		repoIDColWidth = dynamicRepoIDWidth(rpt, termWidth, packageCount)
		if repoIDColWidth < 15 {
			repoIDColWidth = 15
		}
	}

	// Remaining width for package columns
	pkgColWidth := f.MaxPackageColWidth
	if pkgColWidth <= 0 {
		remaining := termWidth - repoIDColWidth - 3
		per := remaining / packageCount
		if per < 8 {
			per = 8
		}
		if per > 24 {
			per = 24
		}
		pkgColWidth = per
	}

	configs := []table.ColumnConfig{
		{
			Number:      1,
			WidthMax:    repoIDColWidth,
			WidthMin:    minInt(10, repoIDColWidth),
			Transformer: truncTransformer(repoIDColWidth),
		},
	}

	for i := 0; i < packageCount; i++ {
		configs = append(configs, table.ColumnConfig{
			Number:      i + 2,
			WidthMax:    pkgColWidth,
			WidthMin:    minInt(5, pkgColWidth),
			Transformer: truncTransformer(pkgColWidth),
		})
	}

	return configs
}

// dynamicRepoIDWidth estimates a good repository identifier column width.
func dynamicRepoIDWidth(rpt *report.Report, termWidth, packageCount int) int {
	if packageCount == 0 {
		return termWidth
	}
	maxLen := 0
	for _, r := range rpt.Repositories {
		id := r.GetRepoIdentifier()
		l := utf8.RuneCountInString(id)
		if l > maxLen {
			maxLen = l
		}
		if maxLen >= 60 {
			break
		}
	}
	// Reserve space for package columns (heuristic similar to previous)
	minPerPkg := 8
	reserved := packageCount * minPerPkg
	available := termWidth - reserved - 3
	if available < 15 {
		available = 15
	}
	if maxLen > available {
		return available
	}
	return maxLen
}

// detectTerminalWidth attempts to get terminal width if writer is a file (stdout/stderr).
func detectTerminalWidth(w io.Writer) int {
	if f, ok := w.(*os.File); ok {
		if width, _, err := term.GetSize(int(f.Fd())); err == nil {
			return width
		}
	}
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

// RenderConsole renders the provided Report to the writer using the default console formatter.
func RenderConsole(rpt *report.Report, w io.Writer) error {
	return NewConsoleFormatter().Render(rpt, w)
}
