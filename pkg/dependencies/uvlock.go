package dependencies

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/BurntSushi/toml"
)

// UvLockAnalyzer implements the Analyzer interface for Python uv projects
// It analyzes uv.lock files to extract dependency information
type UvLockAnalyzer struct{}

// NewUvLockAnalyzer creates a new uv.lock dependency analyzer
func NewUvLockAnalyzer() *UvLockAnalyzer {
	return &UvLockAnalyzer{}
}

// Name returns the name of this analyzer
func (u *UvLockAnalyzer) Name() string {
	return string(AnalyzerUvLock)
}

// CandidateFiles searches for uv.lock files in the configured repository paths
func (u *UvLockAnalyzer) CandidateFiles(ctx context.Context, owner, repo, ref string, config Config) ([]DependencyFile, error) {
	if config.RepositoryClient == nil {
		return nil, fmt.Errorf("repository client is required")
	}

	var candidates []DependencyFile
	searchPaths := config.RepositoryPaths

	// If no paths specified, search from root
	if len(searchPaths) == 0 {
		searchPaths = []string{""}
	}

	// Search each configured path
	for _, searchPath := range searchPaths {
		// List all files recursively in this path
		files, err := config.RepositoryClient.ListFilesRecursive(ctx, owner, repo, ref)
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		// Filter for uv.lock files
		for _, file := range files {
			// Only consider files, not directories
			if file.Type != "file" {
				continue
			}

			// Check if this is a uv.lock file
			if strings.HasSuffix(file.Path, "uv.lock") {
				// If searchPath is specified, ensure file is within that path
				if searchPath != "" && !strings.HasPrefix(file.Path, searchPath) {
					continue
				}

				candidates = append(candidates, DependencyFile{
					Path:     file.Path,
					Type:     "uv.lock",
					Analyzer: u.Name(),
				})
			}
		}
	}

	return candidates, nil
}

// AnalyzeDependencies analyzes uv.lock files and extracts dependency information
func (u *UvLockAnalyzer) AnalyzeDependencies(ctx context.Context, owner, repo, ref string, files []DependencyFile, config Config) (map[string][]Dependency, error) {
	if config.RepositoryClient == nil {
		return nil, fmt.Errorf("repository client is required")
	}

	result := make(map[string][]Dependency)

	for _, file := range files {
		deps, err := u.analyzeFile(ctx, owner, repo, ref, file.Path, config)
		if err != nil {
			// Don't fail completely if one file fails, just skip it
			slog.Debug("Failed to analyze uv.lock file",
				"file", file.Path,
				"owner", owner,
				"repo", repo,
				"ref", ref,
				"error", err)
			continue
		}
		result[file.Path] = deps
	}

	return result, nil
}

// analyzeFile analyzes a single uv.lock file
func (u *UvLockAnalyzer) analyzeFile(ctx context.Context, owner, repo, ref, filePath string, config Config) ([]Dependency, error) {
	// Get the file content from the repository
	content, err := config.RepositoryClient.GetFileContent(ctx, owner, repo, ref, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content for %s: %w", filePath, err)
	}

	// Parse the uv.lock file
	dependencies, err := u.parseUvLock(content)
	if err != nil {
		slog.Debug("Failed to parse uv.lock content",
			"file", filePath,
			"error", err)
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	return dependencies, nil
}

// uvLockFile represents the structure of a uv.lock file
type uvLockFile struct {
	Version        int         `toml:"version"`
	RequiresPython string      `toml:"requires-python"`
	Packages       []uvPackage `toml:"package"`
}

// uvPackage represents a single package entry in uv.lock
type uvPackage struct {
	Name             string                    `toml:"name"`
	Version          string                    `toml:"version"`
	Source           uvSource                  `toml:"source"`
	Dependencies     []uvDependency            `toml:"dependencies"`
	DevDependencies  map[string][]uvDependency `toml:"dev-dependencies"`
	Wheels           []uvWheel                 `toml:"wheels"`
	ResolutionMarker string                    `toml:"resolution-markers"`
	Marker           string                    `toml:"marker"`
	Sdist            uvSdist                   `toml:"sdist"`
}

// uvSource represents the source of a package
type uvSource struct {
	Type     string `toml:"type"`
	URL      string `toml:"url"`
	Registry string `toml:"registry"`
	Git      string `toml:"git"`
	Rev      string `toml:"rev"`
	Tag      string `toml:"tag"`
	Branch   string `toml:"branch"`
	Path     string `toml:"path"`
	Editable string `toml:"editable"`
}

// uvDependency represents a dependency specification
type uvDependency struct {
	Name    string   `toml:"name"`
	Version string   `toml:"version"`
	Marker  string   `toml:"marker"`
	Extra   []string `toml:"extra"`
}

// uvWheel represents wheel information
type uvWheel struct {
	URL  string `toml:"url"`
	Hash string `toml:"hash"`
	Size int64  `toml:"size"`
}

// uvSdist represents source distribution information
type uvSdist struct {
	URL  string `toml:"url"`
	Hash string `toml:"hash"`
	Size int64  `toml:"size"`
}

// parseUvLock parses the content of a uv.lock file
func (u *UvLockAnalyzer) parseUvLock(content string) ([]Dependency, error) {
	var lockFile uvLockFile

	if _, err := toml.Decode(content, &lockFile); err != nil {
		slog.Debug("Failed to decode uv.lock content", "error", err)
		return nil, fmt.Errorf("failed to parse uv.lock: %w", err)
	}

	dependencies := make([]Dependency, 0, len(lockFile.Packages))

	for _, pkg := range lockFile.Packages {
		// Determine dependency type based on markers and dev-dependencies
		depType := "runtime"

		// Check if it has dev dependencies or dev markers
		if len(pkg.DevDependencies["dev"]) > 0 {
			depType = "dev"
		} else if strings.Contains(pkg.Marker, "extra == 'dev'") ||
			strings.Contains(pkg.Marker, "extra == 'test'") ||
			strings.Contains(pkg.ResolutionMarker, "extra == 'dev'") ||
			strings.Contains(pkg.ResolutionMarker, "extra == 'test'") {
			depType = "dev"
		}

		// Determine source
		source := "pypi"
		if pkg.Source.Type != "" {
			switch pkg.Source.Type {
			case "registry":
				source = "pypi"
			case "git":
				source = "git"
			case "path":
				source = "path"
			case "url":
				source = "url"
			case "directory":
				source = "path"
			default:
				source = pkg.Source.Type
			}
		}

		dep := Dependency{
			Name:    pkg.Name,
			Version: pkg.Version,
			Type:    depType,
			Source:  source,
		}

		dependencies = append(dependencies, dep)
	}

	return dependencies, nil
}
