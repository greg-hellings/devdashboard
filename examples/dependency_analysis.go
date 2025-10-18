package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/greg-hellings/devdashboard/pkg/dependencies"
	"github.com/greg-hellings/devdashboard/pkg/repository"
)

func main() {
	fmt.Println("DevDashboard Dependency Analysis Examples")
	fmt.Println("==========================================")
	fmt.Println()

	// Example 1: Analyze a Python Poetry project
	fmt.Println("Example 1: Python Poetry Dependency Analysis")
	fmt.Println("---------------------------------------------")
	if err := examplePoetryAnalysis(); err != nil {
		log.Printf("Poetry example failed: %v\n", err)
	}
	fmt.Println()

	// Example 2: Find all Poetry lock files in a repository
	fmt.Println("Example 2: Find Poetry Lock Files")
	fmt.Println("----------------------------------")
	if err := exampleFindPoetryFiles(); err != nil {
		log.Printf("Find files example failed: %v\n", err)
	}
	fmt.Println()

	// Example 3: Analyze with environment variables
	fmt.Println("Example 3: Custom Repository Analysis")
	fmt.Println("--------------------------------------")
	if err := exampleCustomRepository(); err != nil {
		log.Printf("Custom repository example failed: %v\n", err)
	}
}

// examplePoetryAnalysis demonstrates analyzing a Python Poetry project
func examplePoetryAnalysis() error {
	// Create a repository client
	repoConfig := repository.Config{}
	repoClient, err := repository.NewClient("github", repoConfig)
	if err != nil {
		return fmt.Errorf("failed to create repository client: %w", err)
	}

	// Create a Poetry analyzer
	analyzer, err := dependencies.NewAnalyzer("poetry")
	if err != nil {
		return fmt.Errorf("failed to create analyzer: %w", err)
	}

	// Configure the dependency analyzer
	depConfig := dependencies.Config{
		RepositoryPaths:  []string{""}, // Search entire repository
		RepositoryClient: repoClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Example: Analyze python-poetry/poetry repository
	owner := "python-poetry"
	repo := "poetry"
	ref := "master"

	fmt.Printf("Searching for Poetry lock files in %s/%s...\n", owner, repo)

	// Find candidate files
	candidates, err := analyzer.CandidateFiles(ctx, owner, repo, ref, depConfig)
	if err != nil {
		return fmt.Errorf("failed to find candidate files: %w", err)
	}

	fmt.Printf("Found %d poetry.lock file(s)\n", len(candidates))
	for _, candidate := range candidates {
		fmt.Printf("  - %s\n", candidate.Path)
	}

	if len(candidates) == 0 {
		fmt.Println("No poetry.lock files found")
		return nil
	}

	// Analyze the first few files (limit to avoid long processing)
	filesToAnalyze := candidates
	if len(filesToAnalyze) > 3 {
		filesToAnalyze = candidates[:3]
	}

	fmt.Printf("\nAnalyzing %d file(s)...\n", len(filesToAnalyze))
	results, err := analyzer.AnalyzeDependencies(ctx, owner, repo, ref, filesToAnalyze, depConfig)
	if err != nil {
		return fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	// Display results
	for filePath, deps := range results {
		fmt.Printf("\nDependencies in %s (%d total):\n", filePath, len(deps))

		// Show first 10 dependencies
		displayCount := 10
		if len(deps) < displayCount {
			displayCount = len(deps)
		}

		for i := 0; i < displayCount; i++ {
			dep := deps[i]
			fmt.Printf("  %-30s  v%-15s  [%s]\n", dep.Name, dep.Version, dep.Type)
		}

		if len(deps) > displayCount {
			fmt.Printf("  ... and %d more dependencies\n", len(deps)-displayCount)
		}
	}

	return nil
}

// exampleFindPoetryFiles demonstrates finding Poetry lock files
func exampleFindPoetryFiles() error {
	// Create clients
	repoClient, err := repository.NewClient("github", repository.Config{})
	if err != nil {
		return fmt.Errorf("failed to create repository client: %w", err)
	}

	analyzer := dependencies.NewPoetryAnalyzer()

	// Search a specific path
	config := dependencies.Config{
		RepositoryPaths:  []string{""},
		RepositoryClient: repoClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Search in a different repository
	owner := "psf"
	repo := "requests"
	ref := "main"

	fmt.Printf("Searching for Poetry files in %s/%s...\n", owner, repo)

	candidates, err := analyzer.CandidateFiles(ctx, owner, repo, ref, config)
	if err != nil {
		return fmt.Errorf("failed to find candidates: %w", err)
	}

	if len(candidates) == 0 {
		fmt.Println("No poetry.lock files found in this repository")
		return nil
	}

	fmt.Printf("Found %d poetry.lock file(s):\n", len(candidates))
	for _, candidate := range candidates {
		fmt.Printf("  Path: %s\n", candidate.Path)
		fmt.Printf("  Type: %s\n", candidate.Type)
		fmt.Printf("  Analyzer: %s\n", candidate.Analyzer)
		fmt.Println()
	}

	return nil
}

// exampleCustomRepository demonstrates using environment variables for configuration
func exampleCustomRepository() error {
	// Check for environment variables
	provider := os.Getenv("REPO_PROVIDER")
	token := os.Getenv("REPO_TOKEN")
	owner := os.Getenv("REPO_OWNER")
	repo := os.Getenv("REPO_NAME")
	ref := os.Getenv("REPO_REF")
	analyzerType := os.Getenv("ANALYZER_TYPE")

	// Set defaults if not provided
	if provider == "" {
		provider = "github"
	}
	if analyzerType == "" {
		analyzerType = "poetry"
	}
	if ref == "" {
		ref = "main"
	}

	// Check if required variables are set
	if owner == "" || repo == "" {
		fmt.Println("Skipping: Set REPO_OWNER and REPO_NAME environment variables")
		fmt.Println("Optional: REPO_PROVIDER, REPO_TOKEN, REPO_REF, ANALYZER_TYPE")
		return nil
	}

	fmt.Printf("Analyzing %s/%s (provider: %s, analyzer: %s)\n", owner, repo, provider, analyzerType)

	// Create repository client
	repoConfig := repository.Config{
		Token: token,
	}
	repoClient, err := repository.NewClient(provider, repoConfig)
	if err != nil {
		return fmt.Errorf("failed to create repository client: %w", err)
	}

	// Create analyzer
	analyzer, err := dependencies.NewAnalyzer(analyzerType)
	if err != nil {
		return fmt.Errorf("failed to create analyzer: %w", err)
	}

	// Configure dependency analysis
	depConfig := dependencies.Config{
		RepositoryPaths:  []string{""}, // Search entire repo
		RepositoryClient: repoClient,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Find candidate files
	fmt.Println("Searching for dependency files...")
	candidates, err := analyzer.CandidateFiles(ctx, owner, repo, ref, depConfig)
	if err != nil {
		return fmt.Errorf("failed to find candidates: %w", err)
	}

	fmt.Printf("Found %d candidate file(s)\n", len(candidates))

	if len(candidates) == 0 {
		fmt.Println("No dependency files found")
		return nil
	}

	// Analyze dependencies
	fmt.Println("Analyzing dependencies...")
	results, err := analyzer.AnalyzeDependencies(ctx, owner, repo, ref, candidates, depConfig)
	if err != nil {
		return fmt.Errorf("failed to analyze: %w", err)
	}

	// Summary
	totalDeps := 0
	for _, deps := range results {
		totalDeps += len(deps)
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Files analyzed: %d\n", len(results))
	fmt.Printf("  Total dependencies: %d\n", totalDeps)

	// Count by type
	typeCount := make(map[string]int)
	for _, deps := range results {
		for _, dep := range deps {
			typeCount[dep.Type]++
		}
	}

	if len(typeCount) > 0 {
		fmt.Println("\nDependencies by type:")
		for depType, count := range typeCount {
			fmt.Printf("  %-15s: %d\n", depType, count)
		}
	}

	return nil
}
