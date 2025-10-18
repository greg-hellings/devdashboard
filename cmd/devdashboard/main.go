package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/greg-hellings/devdashboard/pkg/config"
	"github.com/greg-hellings/devdashboard/pkg/dependencies"
	"github.com/greg-hellings/devdashboard/pkg/report"
	"github.com/greg-hellings/devdashboard/pkg/repository"
)

func main() {
	// Initialize logging
	initLogging()

	// Filter out flags to find the actual command
	command := ""
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-") {
			command = arg
			break
		}
	}

	if command == "" {
		printUsage()
		os.Exit(1)
	}

	switch command {
	case "list-files":
		if err := listFiles(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "repo-info":
		if err := repoInfo(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "find-dependencies":
		if err := findDependencies(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "analyze-dependencies":
		if err := analyzeDependencies(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "dependency-report":
		if err := dependencyReport(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func initLogging() {
	// Check for verbosity flags
	verbose := false
	debug := false

	for _, arg := range os.Args {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
		}
		if arg == "-vv" || arg == "--debug" {
			debug = true
		}
	}

	// Set log level based on flags
	var level slog.Level
	if debug {
		level = slog.LevelDebug
	} else if verbose {
		level = slog.LevelInfo
	} else {
		level = slog.LevelWarn
	}

	// Create a new text handler with the appropriate level
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	// Set the default logger
	slog.SetDefault(slog.New(handler))

	slog.Debug("Logging initialized", "level", level.String())
}

func printUsage() {
	fmt.Println("DevDashboard CLI - Repository Management Tool")
	fmt.Println("\nUsage: devdashboard [flags] <command> [arguments]")
	fmt.Println("\nFlags:")
	fmt.Println("  -v, --verbose         Enable verbose output (INFO level)")
	fmt.Println("  -vv, --debug          Enable debug output (DEBUG level)")
	fmt.Println("\nRepository Commands:")
	fmt.Println("  list-files            List files in a repository")
	fmt.Println("  repo-info             Get repository information")
	fmt.Println("\nDependency Commands:")
	fmt.Println("  find-dependencies     Find dependency files in a repository")
	fmt.Println("  analyze-dependencies  Analyze dependencies from dependency files")
	fmt.Println("  dependency-report     Generate dependency version report across repositories")
	fmt.Println("\nGeneral:")
	fmt.Println("  help                  Show this help message")
	fmt.Println("\nEnvironment Variables:")
	fmt.Println("  REPO_PROVIDER      Repository provider (github, gitlab)")
	fmt.Println("  REPO_TOKEN         Authentication token for private repositories")
	fmt.Println("  REPO_BASEURL       Custom base URL for self-hosted instances")
	fmt.Println("  REPO_OWNER         Repository owner/organization")
	fmt.Println("  REPO_NAME          Repository name")
	fmt.Println("  REPO_REF           Git reference (branch/tag/commit, optional)")
	fmt.Println("  ANALYZER_TYPE      Dependency analyzer type (poetry, pipfile, uvlock)")
	fmt.Println("  SEARCH_PATHS       Comma-separated paths to search (optional)")
	fmt.Println("\nExamples:")
	fmt.Println("  # Repository operations")
	fmt.Println("  export REPO_PROVIDER=github")
	fmt.Println("  export REPO_OWNER=torvalds")
	fmt.Println("  export REPO_NAME=linux")
	fmt.Println("  devdashboard list-files")
	fmt.Println()
	fmt.Println("  # With verbose output")
	fmt.Println("  devdashboard -v list-files")
	fmt.Println()
	fmt.Println("  # With debug output")
	fmt.Println("  devdashboard --debug find-dependencies")
	fmt.Println()
	fmt.Println("  # Find Poetry lock files")
	fmt.Println("  export REPO_PROVIDER=github")
	fmt.Println("  export REPO_OWNER=python-poetry")
	fmt.Println("  export REPO_NAME=poetry")
	fmt.Println("  export ANALYZER_TYPE=poetry")
	fmt.Println("  devdashboard find-dependencies")
	fmt.Println()
	fmt.Println("  # Analyze dependencies")
	fmt.Println("  export ANALYZER_TYPE=poetry")
	fmt.Println("  devdashboard analyze-dependencies")
	fmt.Println()
	fmt.Println("  # Generate dependency report")
	fmt.Println("  devdashboard dependency-report config.yaml")
}

func getConfig() (repository.Config, error) {
	token := os.Getenv("REPO_TOKEN")
	baseURL := os.Getenv("REPO_BASEURL")

	return repository.Config{
		Token:   token,
		BaseURL: baseURL,
	}, nil
}

func getRepositoryParams() (provider, owner, repo, ref string, err error) {
	provider = os.Getenv("REPO_PROVIDER")
	if provider == "" {
		return "", "", "", "", fmt.Errorf("REPO_PROVIDER environment variable is required")
	}

	owner = os.Getenv("REPO_OWNER")
	if owner == "" {
		return "", "", "", "", fmt.Errorf("REPO_OWNER environment variable is required")
	}

	repo = os.Getenv("REPO_NAME")
	if repo == "" {
		return "", "", "", "", fmt.Errorf("REPO_NAME environment variable is required")
	}

	ref = os.Getenv("REPO_REF") // Optional

	return provider, owner, repo, ref, nil
}

func listFiles() error {
	slog.Debug("Starting list-files command")

	config, err := getConfig()
	if err != nil {
		return err
	}

	provider, owner, repo, ref, err := getRepositoryParams()
	if err != nil {
		return err
	}

	slog.Info("Listing repository files",
		"provider", provider,
		"owner", owner,
		"repo", repo,
		"ref", ref)

	// Create client using the factory
	client, err := repository.NewClient(provider, config)
	if err != nil {
		return fmt.Errorf("failed to create %s client: %w", provider, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Listing files from %s repository: %s/%s\n", provider, owner, repo)
	if ref != "" {
		fmt.Printf("Reference: %s\n", ref)
	}
	fmt.Println()

	// List files recursively
	files, err := client.ListFilesRecursive(ctx, owner, repo, ref)
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	slog.Info("Files retrieved", "count", len(files))
	fmt.Printf("Found %d files:\n\n", len(files))
	for _, file := range files {
		fmt.Printf("%-80s  (SHA: %s)\n", file.Path, file.SHA[:8])
	}

	return nil
}

func findDependencies() error {
	slog.Debug("Starting find-dependencies command")

	config, err := getConfig()
	if err != nil {
		return err
	}

	provider, owner, repo, ref, err := getRepositoryParams()
	if err != nil {
		return err
	}

	// Get analyzer type
	analyzerType := os.Getenv("ANALYZER_TYPE")
	if analyzerType == "" {
		analyzerType = "poetry" // Default to poetry
	}

	slog.Info("Finding dependency files",
		"provider", provider,
		"owner", owner,
		"repo", repo,
		"ref", ref,
		"analyzer", analyzerType)

	// Create repository client
	repoClient, err := repository.NewClient(provider, config)
	if err != nil {
		return fmt.Errorf("failed to create %s client: %w", provider, err)
	}

	// Create dependency analyzer
	analyzer, err := dependencies.NewAnalyzer(analyzerType)
	if err != nil {
		return fmt.Errorf("failed to create %s analyzer: %w", analyzerType, err)
	}

	// Parse search paths
	var searchPaths []string
	searchPathsEnv := os.Getenv("SEARCH_PATHS")
	if searchPathsEnv != "" {
		searchPaths = strings.Split(searchPathsEnv, ",")
		// Trim whitespace from each path
		for i := range searchPaths {
			searchPaths[i] = strings.TrimSpace(searchPaths[i])
		}
	}

	// Configure dependency analyzer
	depConfig := dependencies.Config{
		RepositoryPaths:  searchPaths,
		RepositoryClient: repoClient,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Printf("Searching for %s dependency files in %s/%s\n", analyzerType, owner, repo)
	if ref != "" {
		fmt.Printf("Reference: %s\n", ref)
	}
	if len(searchPaths) > 0 {
		fmt.Printf("Search paths: %v\n", searchPaths)
	}
	fmt.Println()

	// Find candidate files
	candidates, err := analyzer.CandidateFiles(ctx, owner, repo, ref, depConfig)
	if err != nil {
		return fmt.Errorf("failed to find dependency files: %w", err)
	}

	slog.Info("Candidate files found", "count", len(candidates))

	if len(candidates) == 0 {
		fmt.Println("No dependency files found")
		return nil
	}

	fmt.Printf("Found %d dependency file(s):\n\n", len(candidates))
	for i, candidate := range candidates {
		fmt.Printf("%d. %s\n", i+1, candidate.Path)
		fmt.Printf("   Type: %s\n", candidate.Type)
		fmt.Printf("   Analyzer: %s\n", candidate.Analyzer)
		fmt.Println()
	}

	return nil
}

func analyzeDependencies() error {
	slog.Debug("Starting analyze-dependencies command")

	config, err := getConfig()
	if err != nil {
		return err
	}

	provider, owner, repo, ref, err := getRepositoryParams()
	if err != nil {
		return err
	}

	// Get analyzer type
	analyzerType := os.Getenv("ANALYZER_TYPE")
	if analyzerType == "" {
		analyzerType = "poetry" // Default to poetry
	}

	slog.Info("Analyzing dependencies",
		"provider", provider,
		"owner", owner,
		"repo", repo,
		"ref", ref,
		"analyzer", analyzerType)

	// Create repository client
	repoClient, err := repository.NewClient(provider, config)
	if err != nil {
		return fmt.Errorf("failed to create %s client: %w", provider, err)
	}

	// Create dependency analyzer
	analyzer, err := dependencies.NewAnalyzer(analyzerType)
	if err != nil {
		return fmt.Errorf("failed to create %s analyzer: %w", analyzerType, err)
	}

	// Parse search paths
	var searchPaths []string
	searchPathsEnv := os.Getenv("SEARCH_PATHS")
	if searchPathsEnv != "" {
		searchPaths = strings.Split(searchPathsEnv, ",")
		for i := range searchPaths {
			searchPaths[i] = strings.TrimSpace(searchPaths[i])
		}
	}

	// Configure dependency analyzer
	depConfig := dependencies.Config{
		RepositoryPaths:  searchPaths,
		RepositoryClient: repoClient,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fmt.Printf("Analyzing %s dependencies in %s/%s\n", analyzerType, owner, repo)
	if ref != "" {
		fmt.Printf("Reference: %s\n", ref)
	}
	fmt.Println()

	// Find candidate files
	fmt.Println("Step 1: Finding dependency files...")
	candidates, err := analyzer.CandidateFiles(ctx, owner, repo, ref, depConfig)
	if err != nil {
		return fmt.Errorf("failed to find dependency files: %w", err)
	}

	slog.Info("Candidate files found", "count", len(candidates))

	if len(candidates) == 0 {
		fmt.Println("No dependency files found")
		return nil
	}

	fmt.Printf("Found %d dependency file(s)\n\n", len(candidates))

	// Analyze dependencies
	fmt.Println("Step 2: Analyzing dependencies...")
	slog.Debug("Starting dependency analysis", "fileCount", len(candidates))
	results, err := analyzer.AnalyzeDependencies(ctx, owner, repo, ref, candidates, depConfig)
	if err != nil {
		return fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	slog.Info("Dependency analysis complete", "filesAnalyzed", len(results))

	// Display results
	fmt.Println()
	fmt.Println("Analysis Results")
	fmt.Println("================")
	fmt.Println()

	totalDeps := 0
	typeCount := make(map[string]int)

	for filePath, deps := range results {
		fmt.Printf("File: %s\n", filePath)
		fmt.Printf("Dependencies: %d\n", len(deps))
		fmt.Println()

		// Display first 20 dependencies
		displayCount := 20
		if len(deps) < displayCount {
			displayCount = len(deps)
		}

		for i := 0; i < displayCount; i++ {
			dep := deps[i]
			fmt.Printf("  %-35s v%-15s [%s]\n", dep.Name, dep.Version, dep.Type)
			typeCount[dep.Type]++
			totalDeps++
		}

		if len(deps) > displayCount {
			fmt.Printf("\n  ... and %d more dependencies\n", len(deps)-displayCount)
			// Count remaining dependencies
			for i := displayCount; i < len(deps); i++ {
				typeCount[deps[i].Type]++
				totalDeps++
			}
		}

		fmt.Println()
		fmt.Println(strings.Repeat("-", 80))
		fmt.Println()
	}

	// Summary
	fmt.Println("Summary")
	fmt.Println("=======")
	fmt.Printf("Files analyzed: %d\n", len(results))
	fmt.Printf("Total dependencies: %d\n", totalDeps)

	if len(typeCount) > 0 {
		fmt.Println("\nDependencies by type:")
		for depType, count := range typeCount {
			fmt.Printf("  %-15s: %d\n", depType, count)
		}
	}

	return nil
}

func repoInfo() error {
	slog.Debug("Starting repo-info command")

	config, err := getConfig()
	if err != nil {
		return err
	}

	provider, owner, repo, _, err := getRepositoryParams()
	if err != nil {
		return err
	}

	slog.Info("Getting repository information",
		"provider", provider,
		"owner", owner,
		"repo", repo)

	// Create client using the factory
	client, err := repository.NewClient(provider, config)
	if err != nil {
		return fmt.Errorf("failed to create %s client: %w", provider, err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get repository info
	info, err := client.GetRepositoryInfo(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to get repository info: %w", err)
	}

	fmt.Printf("Repository Information (%s)\n", provider)
	fmt.Println("========================================")
	fmt.Printf("ID:             %s\n", info.ID)
	fmt.Printf("Name:           %s\n", info.Name)
	fmt.Printf("Full Name:      %s\n", info.FullName)
	fmt.Printf("Description:    %s\n", info.Description)
	fmt.Printf("Default Branch: %s\n", info.DefaultBranch)
	fmt.Printf("URL:            %s\n", info.URL)

	return nil
}

func dependencyReport() error {
	slog.Debug("Starting dependency-report command")

	// Get config file from arguments
	args := os.Args
	var configFile string

	// Find config file argument (skip flags and command name)
	for i, arg := range args {
		if arg == "dependency-report" && i+1 < len(args) {
			// Next argument should be the config file
			if !strings.HasPrefix(args[i+1], "-") {
				configFile = args[i+1]
				break
			}
		}
	}

	if configFile == "" {
		return fmt.Errorf("config file path required\nUsage: devdashboard dependency-report <config-file>")
	}

	slog.Info("Loading configuration", "file", configFile)

	// Load configuration
	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get all repositories from config
	repos := cfg.GetAllRepos()
	if len(repos) == 0 {
		return fmt.Errorf("no repositories configured")
	}

	slog.Info("Configuration loaded",
		"providers", len(cfg.Providers),
		"repositories", len(repos))

	fmt.Printf("Generating dependency report for %d repositories...\n\n", len(repos))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Generate report
	generator := report.NewGenerator()
	rpt, err := generator.Generate(ctx, repos)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Display report
	printDependencyReport(rpt)

	// Show errors if any
	if rpt.HasErrors() {
		fmt.Println("\nErrors encountered:")
		fmt.Println(strings.Repeat("=", 80))
		for repo, err := range rpt.GetErrors() {
			fmt.Printf("  %s: %v\n", repo, err)
		}
	}

	return nil
}

func printDependencyReport(rpt *report.Report) {
	fmt.Println("Dependency Version Report")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// Print header with repository names
	fmt.Printf("%-30s", "Package")
	for _, repo := range rpt.Repositories {
		fmt.Printf(" | %-20s", truncate(repo.Repository, 20))
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 80))

	// Print each package with versions across repositories
	for _, pkg := range rpt.Packages {
		fmt.Printf("%-30s", truncate(pkg, 30))

		for _, repo := range rpt.Repositories {
			version := repo.Dependencies[pkg]
			if version == "" {
				if repo.Error != nil {
					fmt.Printf(" | %-20s", "ERROR")
				} else {
					fmt.Printf(" | %-20s", "N/A")
				}
			} else {
				fmt.Printf(" | %-20s", truncate(version, 20))
			}
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("Summary:")
	successCount := 0
	for _, repo := range rpt.Repositories {
		if repo.Error == nil {
			successCount++
		}
	}
	fmt.Printf("  Repositories analyzed: %d/%d successful\n", successCount, len(rpt.Repositories))
	fmt.Printf("  Packages tracked: %d\n", len(rpt.Packages))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
