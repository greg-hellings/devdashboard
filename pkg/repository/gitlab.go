package repository

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"

	"github.com/xanzy/go-gitlab"
)

// GitLabClient implements the Client interface for GitLab repositories
type GitLabClient struct {
	client *gitlab.Client
	config Config
}

// NewGitLabClient creates a new GitLab client with the provided configuration
// If no token is provided, the client will only have access to public repositories
// If a custom BaseURL is provided, it will be used for self-hosted GitLab instances
func NewGitLabClient(config Config) (*GitLabClient, error) {
	var client *gitlab.Client
	var err error

	// Configure client options
	opts := []gitlab.ClientOptionFunc{}

	// Set custom base URL for self-hosted GitLab if provided
	if config.BaseURL != "" {
		opts = append(opts, gitlab.WithBaseURL(config.BaseURL))
	}

	// Create client with authentication if token is provided
	if config.Token != "" {
		client, err = gitlab.NewClient(config.Token, opts...)
	} else {
		client, err = gitlab.NewClient("", opts...)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &GitLabClient{
		client: client,
		config: config,
	}, nil
}

// ListFiles retrieves files and directories at a specific path in the repository
// This returns the contents of a single directory level
func (g *GitLabClient) ListFiles(ctx context.Context, owner, repo, ref, path string) ([]FileInfo, error) {
	// GitLab uses project ID or "namespace/project" format
	projectID := fmt.Sprintf("%s/%s", owner, repo)

	// Configure options for listing tree
	opts := &gitlab.ListTreeOptions{
		Path: gitlab.String(path),
		Ref:  gitlab.String(ref),
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	}

	// If ref is empty, use default branch
	if ref == "" {
		opts.Ref = nil
	}

	// Get repository tree from GitLab API
	trees, resp, err := g.client.Repositories.ListTree(projectID, opts, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to list files from GitLab: %w", err)
	}
	defer resp.Body.Close()

	// Convert GitLab's TreeNode to our FileInfo format
	files := make([]FileInfo, 0, len(trees))
	for _, node := range trees {
		fileType := node.Type
		// Normalize type names to match our interface
		if fileType == "blob" {
			fileType = "file"
		} else if fileType == "tree" {
			fileType = "dir"
		}

		fileInfo := FileInfo{
			Path: node.Path,
			Name: node.Name,
			Type: fileType,
			Mode: node.Mode,
			SHA:  node.ID,
		}

		// Construct web URL for the file/directory
		if ref != "" {
			fileInfo.URL = fmt.Sprintf("%s/-/blob/%s/%s", g.getProjectURL(owner, repo), ref, node.Path)
		}

		files = append(files, fileInfo)
	}

	return files, nil
}

// GetRepositoryInfo retrieves metadata about a GitLab repository
func (g *GitLabClient) GetRepositoryInfo(ctx context.Context, owner, repo string) (*RepositoryInfo, error) {
	projectID := fmt.Sprintf("%s/%s", owner, repo)

	project, resp, err := g.client.Projects.GetProject(projectID, nil, gitlab.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get repository info from GitLab: %w", err)
	}
	defer resp.Body.Close()

	repoInfo := &RepositoryInfo{
		ID:            fmt.Sprintf("%d", project.ID),
		Name:          project.Name,
		FullName:      project.PathWithNamespace,
		Description:   project.Description,
		DefaultBranch: project.DefaultBranch,
		URL:           project.WebURL,
	}

	return repoInfo, nil
}

// ListFilesRecursive retrieves all files recursively in a repository
// This traverses the entire repository tree and returns only files (not directories)
func (g *GitLabClient) ListFilesRecursive(ctx context.Context, owner, repo, ref string) ([]FileInfo, error) {
	projectID := fmt.Sprintf("%s/%s", owner, repo)

	// Use default branch if ref is not specified
	refToUse := ref
	if refToUse == "" {
		repoInfo, err := g.GetRepositoryInfo(ctx, owner, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to get default branch: %w", err)
		}
		refToUse = repoInfo.DefaultBranch
	}

	// Get the repository tree recursively
	opts := &gitlab.ListTreeOptions{
		Recursive: gitlab.Bool(true),
		Ref:       gitlab.String(refToUse),
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	}

	allFiles := make([]FileInfo, 0)
	page := 1

	// GitLab may paginate results, so we need to handle multiple pages
	for {
		opts.Page = page

		trees, resp, err := g.client.Repositories.ListTree(projectID, opts, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("failed to get repository tree from GitLab: %w", err)
		}
		defer resp.Body.Close()

		// Filter and convert tree nodes to FileInfo
		for _, node := range trees {
			// Only include files (blobs), skip trees (directories)
			if node.Type == "blob" {
				fileInfo := FileInfo{
					Path: node.Path,
					Name: filepath.Base(node.Path),
					Type: "file",
					Mode: node.Mode,
					SHA:  node.ID,
					URL:  fmt.Sprintf("%s/-/blob/%s/%s", g.getProjectURL(owner, repo), refToUse, node.Path),
				}
				allFiles = append(allFiles, fileInfo)
			}
		}

		// Check if there are more pages
		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	return allFiles, nil
}

// getProjectURL constructs the base web URL for a GitLab project
// This handles both gitlab.com and self-hosted instances
func (g *GitLabClient) getProjectURL(owner, repo string) string {
	baseURL := g.config.BaseURL
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	return fmt.Sprintf("%s/%s/%s", baseURL, owner, repo)
}

// GetFileContent retrieves the content of a specific file from a GitLab repository
func (g *GitLabClient) GetFileContent(ctx context.Context, owner, repo, ref, path string) (string, error) {
	projectID := fmt.Sprintf("%s/%s", owner, repo)

	// Use default branch if ref is not specified
	refToUse := ref
	if refToUse == "" {
		repoInfo, err := g.GetRepositoryInfo(ctx, owner, repo)
		if err != nil {
			return "", fmt.Errorf("failed to get default branch: %w", err)
		}
		refToUse = repoInfo.DefaultBranch
	}

	// Get file content from GitLab API
	opts := &gitlab.GetFileOptions{
		Ref: gitlab.String(refToUse),
	}

	file, resp, err := g.client.RepositoryFiles.GetFile(projectID, path, opts, gitlab.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("failed to get file content from GitLab: %w", err)
	}
	defer resp.Body.Close()

	// GitLab returns base64 encoded content in the Content field
	// We need to decode it manually
	if file.Content == "" {
		return "", fmt.Errorf("file content is empty: %s", path)
	}

	// Decode base64 content
	decodedContent, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 content: %w", err)
	}

	return string(decodedContent), nil
}
