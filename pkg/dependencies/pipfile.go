package dependencies

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// PipfileAnalyzer implements the Analyzer interface for Python Pipfile projects
// It analyzes Pipfile.lock files to extract dependency information
type PipfileAnalyzer struct{}

// NewPipfileAnalyzer creates a new Pipfile dependency analyzer
func NewPipfileAnalyzer() *PipfileAnalyzer {
	return &PipfileAnalyzer{}
}

// Name returns the name of this analyzer
func (p *PipfileAnalyzer) Name() string {
	return string(AnalyzerPipfile)
}

// CandidateFiles searches for Pipfile.lock files in the configured repository paths
func (p *PipfileAnalyzer) CandidateFiles(ctx context.Context, owner, repo, ref string, config Config) ([]DependencyFile, error) {
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

		// Filter for Pipfile.lock files
		for _, file := range files {
			// Only consider files, not directories
			if file.Type != "file" {
				continue
			}

			// Check if this is a Pipfile.lock file
			if strings.HasSuffix(file.Path, "Pipfile.lock") {
				// If searchPath is specified, ensure file is within that path
				if searchPath != "" && !strings.HasPrefix(file.Path, searchPath) {
					continue
				}

				candidates = append(candidates, DependencyFile{
					Path:     file.Path,
					Type:     "Pipfile.lock",
					Analyzer: p.Name(),
				})
			}
		}
	}

	return candidates, nil
}

// AnalyzeDependencies analyzes Pipfile.lock files and extracts dependency information
func (p *PipfileAnalyzer) AnalyzeDependencies(ctx context.Context, owner, repo, ref string, files []DependencyFile, config Config) (map[string][]Dependency, error) {
	if config.RepositoryClient == nil {
		return nil, fmt.Errorf("repository client is required")
	}

	result := make(map[string][]Dependency)

	for _, file := range files {
		deps, err := p.analyzeFile(ctx, owner, repo, ref, file.Path, config)
		if err != nil {
			// Don't fail completely if one file fails, just skip it
			continue
		}
		result[file.Path] = deps
	}

	return result, nil
}

// analyzeFile analyzes a single Pipfile.lock file
func (p *PipfileAnalyzer) analyzeFile(ctx context.Context, owner, repo, ref, filePath string, config Config) ([]Dependency, error) {
	// Get the file content from the repository
	content, err := config.RepositoryClient.GetFileContent(ctx, owner, repo, ref, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content for %s: %w", filePath, err)
	}

	// Parse the Pipfile.lock file
	dependencies, err := p.parsePipfileLock(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	return dependencies, nil
}

// pipfileLockFile represents the structure of a Pipfile.lock file
type pipfileLockFile struct {
	Meta    pipfileMeta                   `json:"_meta"`
	Default map[string]pipfilePackageInfo `json:"default"`
	Develop map[string]pipfilePackageInfo `json:"develop"`
}

// pipfileMeta represents the metadata section of Pipfile.lock
type pipfileMeta struct {
	Hash        pipfileHash       `json:"hash"`
	PipfileSpec int               `json:"pipfile-spec"`
	Requires    map[string]string `json:"requires"`
	Sources     []pipfileSource   `json:"sources"`
}

// pipfileHash represents the hash section
type pipfileHash struct {
	Sha256 string `json:"sha256"`
}

// pipfileSource represents a package source/index
type pipfileSource struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	VerifySSL bool   `json:"verify_ssl"`
}

// pipfilePackageInfo represents package information in Pipfile.lock
type pipfilePackageInfo struct {
	Version string   `json:"version"`
	Hashes  []string `json:"hashes"`
	Index   string   `json:"index"`
	Markers string   `json:"markers"`
	Extras  []string `json:"extras"`
}

// parsePipfileLock parses the content of a Pipfile.lock file
func (p *PipfileAnalyzer) parsePipfileLock(content string) ([]Dependency, error) {
	var lockFile pipfileLockFile

	if err := json.Unmarshal([]byte(content), &lockFile); err != nil {
		return nil, fmt.Errorf("failed to parse Pipfile.lock: %w", err)
	}

	var dependencies []Dependency

	// Process default (runtime) dependencies
	for name, pkg := range lockFile.Default {
		dep := Dependency{
			Name:    name,
			Version: strings.TrimPrefix(pkg.Version, "=="),
			Type:    "runtime",
			Source:  "pypi",
		}
		dependencies = append(dependencies, dep)
	}

	// Process development dependencies
	for name, pkg := range lockFile.Develop {
		dep := Dependency{
			Name:    name,
			Version: strings.TrimPrefix(pkg.Version, "=="),
			Type:    "dev",
			Source:  "pypi",
		}
		dependencies = append(dependencies, dep)
	}

	return dependencies, nil
}
