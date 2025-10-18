// Package repository provides abstractions and client interfaces for interacting
// with source code hosting providers (e.g., GitHub, GitLab). It defines common
// data structures for file and repository metadata plus a generic Client
// interface implemented by provider-specific clients.
package repository

import (
	"context"
)

// FileInfo represents metadata about a file in a repository
type FileInfo struct {
	Path string // Full path to the file in the repository
	Name string // Name of the file
	Type string // Type: "file", "dir", "symlink", etc.
	Size int64  // Size in bytes
	Mode string // File mode/permissions
	SHA  string // Git SHA or commit hash
	URL  string // URL to the file in the web interface
}

// Info contains metadata about a repository.
type Info struct {
	ID            string // Repository ID
	Name          string // Repository name
	FullName      string // Full name (owner/repo)
	Description   string // Repository description
	DefaultBranch string // Default branch name
	URL           string // Web URL to the repository
}

// RepositoryInfo is kept for backward compatibility.
// Deprecated: use Info instead.
//
//nolint:revive // backward compatibility alias; external code may still reference RepositoryInfo
type RepositoryInfo = Info

// Client defines the interface for interacting with git repository providers
// This interface abstracts operations across different providers (GitHub, GitLab, etc.)
type Client interface {
	// ListFiles retrieves all files in a repository at a specific reference (branch, tag, or commit)
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - owner: Repository owner (username or organization)
	//   - repo: Repository name
	//   - ref: Git reference (branch name, tag, or commit SHA). Empty string uses default branch
	//   - path: Starting path within the repository. Empty string starts at root
	// Returns:
	//   - Slice of FileInfo objects representing files and directories
	//   - Error if the operation fails
	ListFiles(ctx context.Context, owner, repo, ref, path string) ([]FileInfo, error)

	// GetRepositoryInfo retrieves metadata about a repository
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - owner: Repository owner (username or organization)
	//   - repo: Repository name
	// Returns:
	//   - RepositoryInfo containing repository metadata
	//   - Error if the operation fails
	GetRepositoryInfo(ctx context.Context, owner, repo string) (*RepositoryInfo, error)

	// ListFilesRecursive retrieves all files recursively in a repository
	// This is a convenience method that traverses the entire repository tree
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - owner: Repository owner (username or organization)
	//   - repo: Repository name
	//   - ref: Git reference (branch name, tag, or commit SHA). Empty string uses default branch
	// Returns:
	//   - Slice of FileInfo objects for all files (not directories) in the repository
	//   - Error if the operation fails
	ListFilesRecursive(ctx context.Context, owner, repo, ref string) ([]FileInfo, error)

	// GetFileContent retrieves the content of a specific file from the repository
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - owner: Repository owner (username or organization)
	//   - repo: Repository name
	//   - ref: Git reference (branch name, tag, or commit SHA). Empty string uses default branch
	//   - path: Path to the file within the repository
	// Returns:
	//   - String containing the file content
	//   - Error if the operation fails or file is not found
	GetFileContent(ctx context.Context, owner, repo, ref, path string) (string, error)
}

// Config holds common configuration for repository clients
type Config struct {
	// Token is the authentication token for accessing private repositories
	// For GitHub: Personal Access Token
	// For GitLab: Personal Access Token or OAuth token
	Token string

	// BaseURL is the base URL for the API endpoint
	// For GitHub Enterprise or GitLab self-hosted instances
	// Leave empty for public GitHub (github.com) or GitLab (gitlab.com)
	BaseURL string
}
