// Package dependencies contains types and interfaces for discovering and analyzing
// dependency declaration files across repository providers. It defines analyzers
// that can enumerate candidate dependency files and extract structured dependency
// information for reporting.
package dependencies

import (
	"context"

	"github.com/greg-hellings/devdashboard/pkg/repository"
)

// Dependency represents a single dependency with its version information
type Dependency struct {
	Name    string // Name of the dependency package
	Version string // Currently specified version (e.g., "1.2.3", "^2.0.0", ">=1.0.0")
	Type    string // Type of dependency (e.g., "runtime", "dev", "optional")
	Source  string // Source/registry (e.g., "pypi", "npm", "rubygems")
}

// DependencyFile represents a file that contains dependency information
type DependencyFile struct {
	Path     string // Full path to the dependency file in the repository
	Type     string // Type of dependency file (e.g., "poetry.lock", "package-lock.json")
	Analyzer string // Name of the analyzer that handles this file type
}

// Config holds configuration for dependency analyzers
type Config struct {
	// RepositoryPaths is a list of paths within the repository to search
	// for dependency files. Empty list means search the entire repository.
	// Examples: []string{"src", "packages"} or []string{""} for root
	RepositoryPaths []string

	// RepositoryClient is the repository client implementation used to
	// fetch files from the repository
	RepositoryClient repository.Client
}

// Analyzer defines the interface for analyzing dependency files
// Each implementation handles a specific dependency management system
// (e.g., Poetry, npm, Maven, etc.)
type Analyzer interface {
	// Name returns the name of this analyzer (e.g., "poetry", "npm")
	Name() string

	// CandidateFiles returns a list of file paths that this analyzer can process
	// This method searches the configured repository paths for files that match
	// the analyzer's expected patterns (e.g., "poetry.lock", "package.json")
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - owner: Repository owner (username or organization)
	//   - repo: Repository name
	//   - ref: Git reference (branch, tag, or commit SHA)
	//   - config: Configuration with repository paths and client
	//
	// Returns:
	//   - Slice of DependencyFile objects representing candidate files
	//   - Error if the search fails
	CandidateFiles(ctx context.Context, owner, repo, ref string, config Config) ([]DependencyFile, error)

	// AnalyzeDependencies analyzes the specified dependency files and extracts
	// dependency information from them
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - owner: Repository owner (username or organization)
	//   - repo: Repository name
	//   - ref: Git reference (branch, tag, or commit SHA)
	//   - files: List of dependency files to analyze
	//   - config: Configuration with repository client
	//
	// Returns:
	//   - Map of file path to slice of dependencies found in that file
	//   - Error if analysis fails
	AnalyzeDependencies(ctx context.Context, owner, repo, ref string, files []DependencyFile, config Config) (map[string][]Dependency, error)
}

// AnalyzerType represents the type of dependency analyzer
type AnalyzerType string

const (
	// AnalyzerPoetry represents Python Poetry dependency analyzer
	AnalyzerPoetry AnalyzerType = "poetry"
	// AnalyzerPipfile represents Python Pipfile dependency analyzer
	AnalyzerPipfile AnalyzerType = "pipfile"
	// AnalyzerUvLock represents Python uv.lock dependency analyzer
	AnalyzerUvLock AnalyzerType = "uvlock"
)

// Result contains the complete dependency analysis for a repository
type Result struct {
	// Repository information
	Owner string
	Repo  string
	Ref   string

	// CandidateFiles are all files that could potentially contain dependencies
	CandidateFiles []DependencyFile

	// Dependencies is a map of file path to the dependencies found in that file
	Dependencies map[string][]Dependency

	// Analyzer is the name of the analyzer that produced this result
	Analyzer string

	// Errors encountered during analysis (non-fatal)
	Errors []error
}
