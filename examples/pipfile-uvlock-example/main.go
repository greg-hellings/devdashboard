package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/greg-hellings/devdashboard/pkg/dependencies"
	"github.com/greg-hellings/devdashboard/pkg/repository"
)

func main() {
	// Example demonstrating Pipfile and uv.lock analyzers
	fmt.Println("=== DevDashboard Pipfile and uv.lock Analyzer Example ===")

	// Get configuration from environment
	provider := getEnv("REPO_PROVIDER", "github")
	token := os.Getenv("REPO_TOKEN")
	owner := getEnv("REPO_OWNER", "pypa")
	repo := getEnv("REPO_NAME", "pipenv")
	ref := getEnv("REPO_REF", "main")

	fmt.Printf("Provider: %s\n", provider)
	fmt.Printf("Owner: %s\n", owner)
	fmt.Printf("Repository: %s\n", repo)
	fmt.Printf("Reference: %s\n\n", ref)

	// Create repository client
	repoFactory := repository.NewFactory(repository.Config{
		Token: token,
	})
	repoClient, err := repoFactory.CreateClient(provider)
	if err != nil {
		log.Fatalf("Failed to create repository client: %v", err)
	}

	ctx := context.Background()

	// Example 1: Pipfile Analyzer
	fmt.Println("--- Pipfile Analyzer Example ---")
	pipfileAnalyzer, err := dependencies.NewAnalyzer("pipfile")
	if err != nil {
		log.Fatalf("Failed to create Pipfile analyzer: %v", err)
	}

	config := dependencies.Config{
		RepositoryPaths:  []string{""}, // Search entire repository
		RepositoryClient: repoClient,
	}

	// Find Pipfile.lock files
	fmt.Println("\n1. Finding Pipfile.lock files...")
	pipfileFiles, err := pipfileAnalyzer.CandidateFiles(ctx, owner, repo, ref, config)
	if err != nil {
		log.Printf("Error finding Pipfile.lock files: %v", err)
	} else {
		fmt.Printf("Found %d Pipfile.lock file(s):\n", len(pipfileFiles))
		for _, file := range pipfileFiles {
			fmt.Printf("  - %s (type: %s, analyzer: %s)\n", file.Path, file.Type, file.Analyzer)
		}
	}

	// Analyze Pipfile.lock dependencies
	if len(pipfileFiles) > 0 {
		fmt.Println("\n2. Analyzing Pipfile.lock dependencies...")
		pipfileDeps, err := pipfileAnalyzer.AnalyzeDependencies(ctx, owner, repo, ref, pipfileFiles, config)
		if err != nil {
			log.Printf("Error analyzing Pipfile.lock dependencies: %v", err)
		} else {
			for filePath, deps := range pipfileDeps {
				fmt.Printf("\nDependencies in %s (%d total):\n", filePath, len(deps))

				// Group by type
				runtimeDeps := []dependencies.Dependency{}
				devDeps := []dependencies.Dependency{}

				for _, dep := range deps {
					if dep.Type == "dev" {
						devDeps = append(devDeps, dep)
					} else {
						runtimeDeps = append(runtimeDeps, dep)
					}
				}

				if len(runtimeDeps) > 0 {
					fmt.Printf("\n  Runtime Dependencies (%d):\n", len(runtimeDeps))
					for i, dep := range runtimeDeps {
						if i < 10 { // Show first 10
							fmt.Printf("    - %s == %s (source: %s)\n", dep.Name, dep.Version, dep.Source)
						}
					}
					if len(runtimeDeps) > 10 {
						fmt.Printf("    ... and %d more\n", len(runtimeDeps)-10)
					}
				}

				if len(devDeps) > 0 {
					fmt.Printf("\n  Dev Dependencies (%d):\n", len(devDeps))
					for i, dep := range devDeps {
						if i < 5 { // Show first 5
							fmt.Printf("    - %s == %s (source: %s)\n", dep.Name, dep.Version, dep.Source)
						}
					}
					if len(devDeps) > 5 {
						fmt.Printf("    ... and %d more\n", len(devDeps)-5)
					}
				}
			}
		}
	}

	// Example 2: uv.lock Analyzer
	fmt.Println("\n\n--- uv.lock Analyzer Example ---")

	// For uv.lock, let's try a different repository that might use uv
	uvOwner := getEnv("UV_OWNER", "astral-sh")
	uvRepo := getEnv("UV_REPO", "uv")
	uvRef := getEnv("UV_REF", "main")

	fmt.Printf("\nSearching in: %s/%s @ %s\n", uvOwner, uvRepo, uvRef)

	uvlockAnalyzer, err := dependencies.NewAnalyzer("uvlock")
	if err != nil {
		log.Fatalf("Failed to create uv.lock analyzer: %v", err)
	}

	// Find uv.lock files
	fmt.Println("\n1. Finding uv.lock files...")
	uvlockFiles, err := uvlockAnalyzer.CandidateFiles(ctx, uvOwner, uvRepo, uvRef, config)
	if err != nil {
		log.Printf("Error finding uv.lock files: %v", err)
	} else {
		fmt.Printf("Found %d uv.lock file(s):\n", len(uvlockFiles))
		for _, file := range uvlockFiles {
			fmt.Printf("  - %s (type: %s, analyzer: %s)\n", file.Path, file.Type, file.Analyzer)
		}
	}

	// Analyze uv.lock dependencies
	if len(uvlockFiles) > 0 {
		fmt.Println("\n2. Analyzing uv.lock dependencies...")
		uvlockDeps, err := uvlockAnalyzer.AnalyzeDependencies(ctx, uvOwner, uvRepo, uvRef, uvlockFiles, config)
		if err != nil {
			log.Printf("Error analyzing uv.lock dependencies: %v", err)
		} else {
			for filePath, deps := range uvlockDeps {
				fmt.Printf("\nDependencies in %s (%d total):\n", filePath, len(deps))

				// Group by type and source
				runtimeDeps := []dependencies.Dependency{}
				devDeps := []dependencies.Dependency{}
				gitDeps := []dependencies.Dependency{}

				for _, dep := range deps {
					if dep.Type == "dev" {
						devDeps = append(devDeps, dep)
					} else {
						runtimeDeps = append(runtimeDeps, dep)
					}
					if dep.Source == "git" {
						gitDeps = append(gitDeps, dep)
					}
				}

				if len(runtimeDeps) > 0 {
					fmt.Printf("\n  Runtime Dependencies (%d):\n", len(runtimeDeps))
					for i, dep := range runtimeDeps {
						if i < 10 { // Show first 10
							fmt.Printf("    - %s @ %s (source: %s)\n", dep.Name, dep.Version, dep.Source)
						}
					}
					if len(runtimeDeps) > 10 {
						fmt.Printf("    ... and %d more\n", len(runtimeDeps)-10)
					}
				}

				if len(devDeps) > 0 {
					fmt.Printf("\n  Dev Dependencies (%d):\n", len(devDeps))
					for i, dep := range devDeps {
						if i < 5 { // Show first 5
							fmt.Printf("    - %s @ %s (source: %s)\n", dep.Name, dep.Version, dep.Source)
						}
					}
					if len(devDeps) > 5 {
						fmt.Printf("    ... and %d more\n", len(devDeps)-5)
					}
				}

				if len(gitDeps) > 0 {
					fmt.Printf("\n  Git-based Dependencies (%d):\n", len(gitDeps))
					for _, dep := range gitDeps {
						fmt.Printf("    - %s @ %s (source: %s)\n", dep.Name, dep.Version, dep.Source)
					}
				}
			}
		}
	}

	// Summary
	fmt.Println("\n\n--- Summary ---")
	fmt.Println("Supported Python dependency analyzers:")
	for _, analyzer := range dependencies.SupportedAnalyzers() {
		fmt.Printf("  - %s\n", analyzer)
	}

	fmt.Println("\nTo use this example:")
	fmt.Println("  export REPO_PROVIDER=github")
	fmt.Println("  export REPO_TOKEN=your_token_here  # Optional for public repos")
	fmt.Println("  export REPO_OWNER=pypa")
	fmt.Println("  export REPO_NAME=pipenv")
	fmt.Println("  go run examples/pipfile_uvlock_example.go")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
