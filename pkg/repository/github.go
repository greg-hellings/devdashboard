// Package repository provides clients for interacting with various source code hosting platforms
// including GitHub and GitLab. It offers a unified interface for accessing repository metadata,
// file listings, and file contents.
package repository

import (
	"context"
	"fmt"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// GitHubClient implements the Client interface for GitHub repositories
type GitHubClient struct {
	client *github.Client
	config Config
}

// NewGitHubClient creates a new GitHub client with the provided configuration
// If no token is provided, the client will only have access to public repositories
// If a custom BaseURL is provided, it will be used for GitHub Enterprise instances
func NewGitHubClient(config Config) (*GitHubClient, error) {
	var client *github.Client

	ctx := context.Background()

	// Configure authentication if token is provided
	if config.Token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: config.Token},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	} else {
		client = github.NewClient(nil)
	}

	// Set custom base URL for GitHub Enterprise if provided
	if config.BaseURL != "" {
		var err error
		client, err = client.WithEnterpriseURLs(config.BaseURL, config.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to set GitHub Enterprise URL: %w", err)
		}
	}

	return &GitHubClient{
		client: client,
		config: config,
	}, nil
}

// ListFiles retrieves files and directories at a specific path in the repository
// This returns the contents of a single directory level
func (g *GitHubClient) ListFiles(ctx context.Context, owner, repo, ref, path string) ([]FileInfo, error) {
	// Use default branch if ref is not specified
	opts := &github.RepositoryContentGetOptions{}
	if ref != "" {
		opts.Ref = ref
	}

	// Get directory contents from GitHub API
	_, directoryContent, resp, err := g.client.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list files from GitHub: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log the error but don't override the primary error
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	// Convert GitHub's RepositoryContent to our FileInfo format
	files := make([]FileInfo, 0, len(directoryContent))
	for _, content := range directoryContent {
		fileInfo := FileInfo{
			Path: content.GetPath(),
			Name: content.GetName(),
			Type: content.GetType(),
			Size: int64(content.GetSize()),
			SHA:  content.GetSHA(),
			URL:  content.GetHTMLURL(),
			// Mode is not available in directory listing API
			Mode: "",
		}

		files = append(files, fileInfo)
	}

	return files, nil
}

// GetRepositoryInfo retrieves metadata about a GitHub repository
func (g *GitHubClient) GetRepositoryInfo(ctx context.Context, owner, repo string) (*RepositoryInfo, error) {
	ghRepo, resp, err := g.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info from GitHub: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	repoInfo := &RepositoryInfo{
		ID:            fmt.Sprintf("%d", ghRepo.GetID()),
		Name:          ghRepo.GetName(),
		FullName:      ghRepo.GetFullName(),
		Description:   ghRepo.GetDescription(),
		DefaultBranch: ghRepo.GetDefaultBranch(),
		URL:           ghRepo.GetHTMLURL(),
	}

	return repoInfo, nil
}

// ListFilesRecursive retrieves all files recursively in a repository
// This traverses the entire repository tree and returns only files (not directories)
func (g *GitHubClient) ListFilesRecursive(ctx context.Context, owner, repo, ref string) ([]FileInfo, error) {
	// Use default branch if ref is not specified
	refToUse := ref
	if refToUse == "" {
		repoInfo, err := g.GetRepositoryInfo(ctx, owner, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to get default branch: %w", err)
		}
		refToUse = repoInfo.DefaultBranch
	}

	// Get the Git tree recursively
	// This is more efficient than manually traversing directory by directory
	tree, resp, err := g.client.Git.GetTree(ctx, owner, repo, refToUse, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository tree from GitHub: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	// Filter out directories and convert to FileInfo
	files := make([]FileInfo, 0)
	for _, entry := range tree.Entries {
		// Only include files (blobs), skip trees (directories) and other types
		if entry.GetType() == "blob" {
			fileInfo := FileInfo{
				Path: entry.GetPath(),
				Name: extractFileName(entry.GetPath()),
				Type: "file",
				Size: int64(entry.GetSize()),
				SHA:  entry.GetSHA(),
				Mode: entry.GetMode(),
				// Note: URL is not directly available in tree entries
				// Would need additional API call per file to get HTML URL
				URL: fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, refToUse, entry.GetPath()),
			}
			files = append(files, fileInfo)
		}
	}

	return files, nil
}

// extractFileName extracts the filename from a full path
// e.g., "path/to/file.txt" -> "file.txt"
func extractFileName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// GetFileContent retrieves the content of a specific file from a GitHub repository
func (g *GitHubClient) GetFileContent(ctx context.Context, owner, repo, ref, path string) (string, error) {
	// Use default branch if ref is not specified
	opts := &github.RepositoryContentGetOptions{}
	if ref != "" {
		opts.Ref = ref
	}

	// Get file content from GitHub API
	fileContent, _, resp, err := g.client.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return "", fmt.Errorf("failed to get file content from GitHub: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	// Check if we got a file (not a directory)
	if fileContent == nil {
		return "", fmt.Errorf("path is not a file: %s", path)
	}

	// Get the content - GitHub API returns base64 encoded content
	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode file content: %w", err)
	}

	return content, nil
}
