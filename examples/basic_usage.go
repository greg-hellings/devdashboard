package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gregoryhellings/devdashboard/pkg/repository"
)

func main() {
	fmt.Println("DevDashboard Repository Client Examples")
	fmt.Println("========================================")

	// Example 1: Using the factory to create a GitHub client
	fmt.Println("Example 1: GitHub Public Repository")
	fmt.Println("------------------------------------")
	if err := exampleGitHubPublic(); err != nil {
		log.Printf("GitHub example failed: %v\n", err)
	}
	fmt.Println()

	// Example 2: Using the factory to create a GitLab client
	fmt.Println("Example 2: GitLab Public Repository")
	fmt.Println("------------------------------------")
	if err := exampleGitLabPublic(); err != nil {
		log.Printf("GitLab example failed: %v\n", err)
	}
	fmt.Println()

	// Example 3: Using authentication for private repositories
	fmt.Println("Example 3: Private Repository with Authentication")
	fmt.Println("--------------------------------------------------")
	if err := examplePrivateRepository(); err != nil {
		log.Printf("Private repository example failed: %v\n", err)
	}
	fmt.Println()

	// Example 4: List files in a specific directory
	fmt.Println("Example 4: List Files in Specific Directory")
	fmt.Println("--------------------------------------------")
	if err := exampleListDirectory(); err != nil {
		log.Printf("List directory example failed: %v\n", err)
	}
	fmt.Println()

	// Example 5: Using factory to support multiple providers
	fmt.Println("Example 5: Factory Pattern with Multiple Providers")
	fmt.Println("---------------------------------------------------")
	if err := exampleFactoryPattern(); err != nil {
		log.Printf("Factory pattern example failed: %v\n", err)
	}
}

// exampleGitHubPublic demonstrates accessing a public GitHub repository
func exampleGitHubPublic() error {
	// Create a client for public repositories (no token needed)
	config := repository.Config{}
	client, err := repository.NewClient("github", config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get repository information
	info, err := client.GetRepositoryInfo(ctx, "golang", "go")
	if err != nil {
		return fmt.Errorf("failed to get repo info: %w", err)
	}

	fmt.Printf("Repository: %s\n", info.FullName)
	fmt.Printf("Description: %s\n", info.Description)
	fmt.Printf("Default Branch: %s\n", info.DefaultBranch)
	fmt.Printf("URL: %s\n", info.URL)

	// List first few files in the root directory
	files, err := client.ListFiles(ctx, "golang", "go", "", "")
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	fmt.Printf("\nFirst 10 items in root directory:\n")
	for i, file := range files {
		if i >= 10 {
			break
		}
		fmt.Printf("  [%s] %s\n", file.Type, file.Name)
	}

	return nil
}

// exampleGitLabPublic demonstrates accessing a public GitLab repository
func exampleGitLabPublic() error {
	// Create a client for GitLab
	config := repository.Config{}
	client, err := repository.NewClient("gitlab", config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get repository information
	info, err := client.GetRepositoryInfo(ctx, "gitlab-org", "gitlab")
	if err != nil {
		return fmt.Errorf("failed to get repo info: %w", err)
	}

	fmt.Printf("Repository: %s\n", info.FullName)
	fmt.Printf("Description: %s\n", info.Description)
	fmt.Printf("Default Branch: %s\n", info.DefaultBranch)
	fmt.Printf("URL: %s\n", info.URL)

	// List files at root
	files, err := client.ListFiles(ctx, "gitlab-org", "gitlab", "", "")
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	fmt.Printf("\nFound %d items in root directory\n", len(files))

	return nil
}

// examplePrivateRepository demonstrates accessing a private repository with authentication
func examplePrivateRepository() error {
	// Check if token is provided via environment variable
	token := os.Getenv("REPO_TOKEN")
	if token == "" {
		fmt.Println("Skipping: Set REPO_TOKEN environment variable to test private repositories")
		return nil
	}

	provider := os.Getenv("REPO_PROVIDER")
	if provider == "" {
		provider = "github" // default to GitHub
	}

	owner := os.Getenv("REPO_OWNER")
	repo := os.Getenv("REPO_NAME")

	if owner == "" || repo == "" {
		fmt.Println("Skipping: Set REPO_OWNER and REPO_NAME to test private repositories")
		return nil
	}

	// Create authenticated client
	config := repository.Config{
		Token: token,
	}

	client, err := repository.NewClient(provider, config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Access private repository
	info, err := client.GetRepositoryInfo(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to access private repository: %w", err)
	}

	fmt.Printf("Successfully accessed private repository: %s\n", info.FullName)
	fmt.Printf("Default Branch: %s\n", info.DefaultBranch)

	return nil
}

// exampleListDirectory demonstrates listing files in a specific directory
func exampleListDirectory() error {
	config := repository.Config{}
	client, err := repository.NewClient("github", config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// List files in the "src" directory of the Go repository
	files, err := client.ListFiles(ctx, "golang", "go", "master", "src")
	if err != nil {
		return fmt.Errorf("failed to list files: %w", err)
	}

	fmt.Printf("Items in 'src' directory:\n")
	for i, file := range files {
		if i >= 15 {
			fmt.Printf("  ... and %d more items\n", len(files)-15)
			break
		}
		typeIcon := "📄"
		if file.Type == "dir" {
			typeIcon = "📁"
		}
		fmt.Printf("  %s %s\n", typeIcon, file.Name)
	}

	return nil
}

// exampleFactoryPattern demonstrates using the factory to work with multiple providers
func exampleFactoryPattern() error {
	// Create a factory with shared configuration
	config := repository.Config{
		Token: os.Getenv("REPO_TOKEN"), // Optional, works without it for public repos
	}
	factory := repository.NewFactory(config)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get list of supported providers
	providers := repository.SupportedProviders()
	fmt.Printf("Supported providers: %v\n\n", providers)

	// Create clients for different providers using the same factory
	for _, providerName := range providers {
		fmt.Printf("Creating %s client...\n", providerName)
		client, err := factory.CreateClient(providerName)
		if err != nil {
			log.Printf("  Failed to create %s client: %v\n", providerName, err)
			continue
		}

		// Determine which repository to query based on provider
		var owner, repo string
		if providerName == "github" {
			owner, repo = "golang", "go"
		} else if providerName == "gitlab" {
			owner, repo = "gitlab-org", "gitlab-foss"
		}

		// Get repository info
		info, err := client.GetRepositoryInfo(ctx, owner, repo)
		if err != nil {
			log.Printf("  Failed to get info for %s/%s: %v\n", owner, repo, err)
			continue
		}

		fmt.Printf("  ✓ Successfully connected to %s\n", info.FullName)
		fmt.Printf("    Default branch: %s\n", info.DefaultBranch)
	}

	return nil
}
