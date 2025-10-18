package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/greg-hellings/devdashboard/pkg/config"
	"github.com/greg-hellings/devdashboard/pkg/report"
	consolefmt "github.com/greg-hellings/devdashboard/pkg/report/format"
	"github.com/spf13/cobra"
)

// build-time override (e.g. -ldflags "-X main.version=1.2.3")
var version = "dev"

// Global (root-level) flag variables
var (
	flagVerbose bool
	flagDebug   bool
)

// dependency-report command flags
type depReportFlags struct {
	outputFormat      string
	outputFile        string
	noColor           bool
	packageColWidth   int
	repoColWidth      int
	timeout           time.Duration
	failOnRepoError   bool
	jsonIndent        bool
	jsonIncludeErrors bool
}

var depFlags depReportFlags

func main() {
	root := newRootCmd()
	root.SilenceUsage = true
	root.SilenceErrors = true

	if err := root.Execute(); err != nil {
		// If Execute() returns an error, logging may or may not be initialized yet.
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// newRootCmd creates the root Cobra command.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devdashboard",
		Short: "DevDashboard CLI",
		Long: strings.TrimSpace(`
DevDashboard - Dependency reporting tool

Current focus: Generate a cross-repository dependency version report using a
configuration file that declares providers, repositories, analyzers, and the
packages to track.`),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			initLogging()
			return nil
		},
	}

	// Global flags
	cmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose (info) logging")
	cmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "Enable debug logging (overrides --verbose)")
	cmd.Version = version

	// Add subcommands
	cmd.AddCommand(newDependencyReportCmd())
	cmd.AddCommand(newVersionCmd())

	return cmd
}

// newVersionCmd prints version info (simple helper).
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("DevDashboard version: %s\n", version)
		},
	}
}

// newDependencyReportCmd creates the 'dependency-report' subcommand.
func newDependencyReportCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "dependency-report <config-file>",
		Short: "Generate a dependency version report across repositories",
		Long: strings.TrimSpace(`
Generate a cross-repository dependency version report defined by a YAML
configuration file. The configuration specifies providers, repositories,
analyzer type, and the packages to track.

Formats:
  console (default) - adaptive terminal table
  json              - machine-readable JSON

Examples:
  devdashboard dependency-report repos.yaml
  devdashboard dependency-report repos.yaml --format json --json-indent
  devdashboard dependency-report repos.yaml --format console --no-color
`),
		Args: cobra.ExactArgs(1),
		RunE: runDependencyReport,
	}

	c.Flags().StringVarP(&depFlags.outputFormat, "format", "f", "console", "Output format: console|json")
	c.Flags().StringVarP(&depFlags.outputFile, "out", "o", "", "Write output to file instead of stdout")
	c.Flags().BoolVar(&depFlags.noColor, "no-color", false, "Disable ANSI colors (console format)")
	c.Flags().IntVar(&depFlags.packageColWidth, "package-col-width", 0, "Max width of package column (console format; 0=auto)")
	c.Flags().IntVar(&depFlags.repoColWidth, "repo-col-width", 0, "Max width of repository/version columns (console format; 0=auto)")
	c.Flags().DurationVar(&depFlags.timeout, "timeout", 5*time.Minute, "Timeout for generating the report")
	c.Flags().BoolVar(&depFlags.failOnRepoError, "fail-on-error", false, "Exit with non-zero status if any repository failed to analyze")
	c.Flags().BoolVar(&depFlags.jsonIndent, "json-indent", false, "Pretty-print JSON output")
	c.Flags().BoolVar(&depFlags.jsonIncludeErrors, "json-include-errors", true, "Include repository errors section in JSON output")

	return c
}

func initLogging() {
	// If already initialized (e.g., multiple subcommands), skip.
	// We rely on slog default logger replacement here idempotently.
	var level slog.Level
	switch {
	case flagDebug:
		level = slog.LevelDebug
	case flagVerbose:
		level = slog.LevelInfo
	default:
		level = slog.LevelWarn
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
	slog.Debug("Logging initialized", "level", level.String())
}

// runDependencyReport executes the core logic for dependency-report.
func runDependencyReport(cmd *cobra.Command, args []string) error {
	start := time.Now()
	configFile := args[0]

	slog.Info("Starting dependency report",
		"configFile", configFile,
		"format", depFlags.outputFormat)

	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	repos := cfg.GetAllRepos()
	if len(repos) == 0 {
		return errors.New("no repositories configured in the provided file")
	}

	ctx, cancel := context.WithTimeout(context.Background(), depFlags.timeout)
	defer cancel()

	generator := report.NewGenerator()
	rpt, err := generator.Generate(ctx, repos)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	var outWriter ioWriteCloser = stdOutWriteCloser{w: os.Stdout}
	if depFlags.outputFile != "" {
		if err := os.MkdirAll(filepath.Dir(depFlags.outputFile), 0o755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		f, err := os.Create(depFlags.outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		outWriter = f
	}
	defer outWriter.Close()

	switch strings.ToLower(depFlags.outputFormat) {
	case "console":
		if err := renderConsole(rpt, outWriter); err != nil {
			return fmt.Errorf("failed to render console output: %w", err)
		}
	case "json":
		if err := renderJSON(rpt, outWriter); err != nil {
			return fmt.Errorf("failed to render JSON output: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format: %s", depFlags.outputFormat)
	}

	duration := time.Since(start)
	slog.Info("Dependency report complete",
		"repositories", len(rpt.Repositories),
		"packages", len(rpt.Packages),
		"duration", duration.String())

	if depFlags.failOnRepoError && rpt.HasErrors() {
		return errors.New("one or more repositories failed (fail-on-error enabled)")
	}

	return nil
}

// renderConsole renders the report using the console formatter.
func renderConsole(rpt *report.Report, w ioWriter) error {
	fmt.Fprintf(w, "Dependency Version Report (format=console)\n\n")

	formatter := consolefmt.NewConsoleFormatter()
	formatter.EnableColors = !depFlags.noColor
	if depFlags.packageColWidth > 0 {
		formatter.MaxPackageColWidth = depFlags.packageColWidth
	}
	if depFlags.repoColWidth > 0 {
		formatter.MaxRepoColWidth = depFlags.repoColWidth
	}
	return formatter.Render(rpt, w)
}

// jsonOutput is the structured JSON shape we emit (allows adding summary without
// changing core report.Report struct).
type jsonOutput struct {
	Version      string                    `json:"cliVersion"`
	GeneratedAt  time.Time                 `json:"generatedAt"`
	Repositories []report.RepositoryReport `json:"repositories"`
	Packages     []string                  `json:"packages"`
	Summary      jsonSummary               `json:"summary"`
	Errors       map[string]string         `json:"errors,omitempty"`
}

type jsonSummary struct {
	RepositoryCount int `json:"repositoryCount"`
	PackageCount    int `json:"packageCount"`
	SuccessCount    int `json:"successCount"`
	ErrorCount      int `json:"errorCount"`
}

// renderJSON marshals the report to JSON with additional metadata.
func renderJSON(rpt *report.Report, w ioWriter) error {
	successCount := 0
	for _, rr := range rpt.Repositories {
		if rr.Error == nil {
			successCount++
		}
	}
	errCount := len(rpt.Repositories) - successCount

	var errMap map[string]string
	if depFlags.jsonIncludeErrors && rpt.HasErrors() {
		errMap = make(map[string]string)
		for repoID, err := range rpt.GetErrors() {
			errMap[repoID] = err.Error()
		}
	}

	payload := jsonOutput{
		Version:      version,
		GeneratedAt:  time.Now().UTC(),
		Repositories: rpt.Repositories,
		Packages:     rpt.Packages,
		Summary: jsonSummary{
			RepositoryCount: len(rpt.Repositories),
			PackageCount:    len(rpt.Packages),
			SuccessCount:    successCount,
			ErrorCount:      errCount,
		},
		Errors: errMap,
	}

	var data []byte
	var err error
	if depFlags.jsonIndent {
		data, err = json.MarshalIndent(payload, "", "  ")
	} else {
		data, err = json.Marshal(payload)
	}
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	_, _ = w.Write(data)
	_, _ = w.Write([]byte("\n"))
	return nil
}

/* ---------- Minimal ioWriter / ioWriteCloser helpers (avoid extra imports) ---------- */

type ioWriter interface {
	Write(p []byte) (n int, err error)
}

type ioWriteCloser interface {
	ioWriter
	Close() error
}

type stdOutWriteCloser struct {
	w ioWriter
}

func (s stdOutWriteCloser) Write(p []byte) (int, error) {
	return s.w.Write(p)
}

func (s stdOutWriteCloser) Close() error {
	// stdout should not be closed
	return nil
}
