package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/greg-hellings/devdashboard/pkg/dependencies"
	"github.com/greg-hellings/devdashboard/pkg/repository"
)

func main() {
	fmt.Println("=== DevDashboard Logging Demo ===")

	// Demonstrate different log levels
	fmt.Println("1. Testing with WARN level (default - minimal output):")
	fmt.Println("   Only warnings and errors will be shown")
	initLogger(slog.LevelWarn)
	testAnalysis()

	fmt.Println("\n" + string(make([]byte, 80)) + "\n")

	fmt.Println("2. Testing with INFO level (verbose):")
	fmt.Println("   Info, warnings, and errors will be shown")
	initLogger(slog.LevelInfo)
	testAnalysis()

	fmt.Println("\n" + string(make([]byte, 80)) + "\n")

	fmt.Println("3. Testing with DEBUG level (very verbose):")
	fmt.Println("   All log messages including debug will be shown")
	initLogger(slog.LevelDebug)
	testAnalysis()
}

func initLogger(level slog.Level) {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
	slog.Info("Logger initialized", "level", level.String())
}

func testAnalysis() {
	// This demonstrates how logging works when errors occur during dependency analysis

	// Create a mock repository client that will fail
	mockClient := &failingMockClient{}

	// Create an analyzer
	analyzer := dependencies.NewPoetryAnalyzer()

	// Configure with the failing client
	config := dependencies.Config{
		RepositoryClient: mockClient,
		RepositoryPaths:  []string{""},
	}

	ctx := context.Background()

	// Try to find candidate files - this will succeed
	fmt.Println("\nFinding candidate files...")
	candidates := []dependencies.DependencyFile{
		{Path: "poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
		{Path: "invalid/poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
		{Path: "broken/poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
	}
	fmt.Printf("Found %d candidate files\n", len(candidates))

	// Try to analyze - this will log debug messages for failures
	fmt.Println("\nAnalyzing dependencies (some will fail)...")
	results, err := analyzer.AnalyzeDependencies(ctx, "test-owner", "test-repo", "main", candidates, config)

	if err != nil {
		fmt.Printf("Fatal error: %v\n", err)
		return
	}

	fmt.Printf("\nSuccessfully analyzed %d out of %d files\n", len(results), len(candidates))

	// The other files that failed will have debug log messages
	if len(results) < len(candidates) {
		fmt.Printf("Note: %d files failed to parse (see debug logs above)\n", len(candidates)-len(results))
	}
}

// failingMockClient simulates a repository client that returns errors
type failingMockClient struct {
	callCount int
}

func (m *failingMockClient) GetRepositoryInfo(ctx context.Context, owner, repo string) (*repository.Info, error) {
	return &repository.Info{
		Name:     repo,
		FullName: owner + "/" + repo,
	}, nil
}

func (m *failingMockClient) ListFiles(ctx context.Context, owner, repo, ref, path string) ([]repository.FileInfo, error) {
	return []repository.FileInfo{
		{Path: "poetry.lock", Type: "file"},
		{Path: "invalid/poetry.lock", Type: "file"},
	}, nil
}

func (m *failingMockClient) ListFilesRecursive(ctx context.Context, owner, repo, ref string) ([]repository.FileInfo, error) {
	return m.ListFiles(ctx, owner, repo, ref, "")
}

func (m *failingMockClient) GetFileContent(ctx context.Context, owner, repo, ref, path string) (string, error) {
	m.callCount++

	// First call succeeds with valid content
	if m.callCount == 1 {
		return `[[package]]
name = "requests"
version = "2.28.1"
description = "HTTP library"
category = "main"

[metadata]
python-versions = ">=3.7"
content-hash = "abc123"
`, nil
	}

	// Second call succeeds but with invalid content
	if m.callCount == 2 {
		return "invalid toml content {{{", nil
	}

	// Third call fails with error
	return "", fmt.Errorf("simulated network error: connection timeout")
}
