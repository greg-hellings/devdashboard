package dependencies

import (
	"context"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

// PoetryAnalyzer implements the Analyzer interface for Python Poetry projects
// It analyzes poetry.lock files to extract dependency information
type PoetryAnalyzer struct{}

// NewPoetryAnalyzer creates a new Poetry dependency analyzer
func NewPoetryAnalyzer() *PoetryAnalyzer {
	return &PoetryAnalyzer{}
}

// Name returns the name of this analyzer
func (p *PoetryAnalyzer) Name() string {
	return string(AnalyzerPoetry)
}

// CandidateFiles searches for poetry.lock files in the configured repository paths
func (p *PoetryAnalyzer) CandidateFiles(ctx context.Context, owner, repo, ref string, config Config) ([]DependencyFile, error) {
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

		// Filter for poetry.lock files
		for _, file := range files {
			// Only consider files, not directories
			if file.Type != "file" {
				continue
			}

			// Check if this is a poetry.lock file
			if strings.HasSuffix(file.Path, "poetry.lock") {
				// If searchPath is specified, ensure file is within that path
				if searchPath != "" && !strings.HasPrefix(file.Path, searchPath) {
					continue
				}

				candidates = append(candidates, DependencyFile{
					Path:     file.Path,
					Type:     "poetry.lock",
					Analyzer: p.Name(),
				})
			}
		}
	}

	return candidates, nil
}

// AnalyzeDependencies analyzes poetry.lock files and extracts dependency information
func (p *PoetryAnalyzer) AnalyzeDependencies(ctx context.Context, owner, repo, ref string, files []DependencyFile, config Config) (map[string][]Dependency, error) {
	if config.RepositoryClient == nil {
		return nil, fmt.Errorf("repository client is required")
	}

	result := make(map[string][]Dependency)

	for _, file := range files {
		deps, err := p.analyzeFile(ctx, owner, repo, ref, file.Path, config)
		fmt.Println("Founds deps: ", deps)
		if err != nil {
			// Don't fail completely if one file fails, just skip it
			// Caller can check for incomplete results
			fmt.Println("Error: ", err)
			continue
		}
		result[file.Path] = deps
	}

	return result, nil
}

// analyzeFile analyzes a single poetry.lock file
func (p *PoetryAnalyzer) analyzeFile(ctx context.Context, owner, repo, ref, filePath string, config Config) ([]Dependency, error) {
	// Get the file content from the repository
	content, err := config.RepositoryClient.GetFileContent(ctx, owner, repo, ref, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content for %s: %w", filePath, err)
	}
	fmt.Println(content)

	// Parse the poetry.lock file
	dependencies, err := p.parsePoetryLock(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
	}

	return dependencies, nil
}

// poetryLockFile represents the structure of a poetry.lock file
type poetryLockFile struct {
	Package  []poetryPackage `toml:"package"`
	Metadata poetryMetadata  `toml:"metadata"`
}

// poetryPackage represents a single package entry in poetry.lock
type poetryPackage struct {
	Name        string `toml:"name"`
	Version     string `toml:"version"`
	Description string `toml:"description"`
	Category    string `toml:"category"`
	Optional    bool   `toml:"optional"`
}

// poetryMetadata represents the metadata section of poetry.lock
type poetryMetadata struct {
	PythonVersions string `toml:"python-versions"`
	ContentHash    string `toml:"content-hash"`
}

// parsePoetryLock parses the content of a poetry.lock file
func (p *PoetryAnalyzer) parsePoetryLock(content string) ([]Dependency, error) {
	var lockFile poetryLockFile

	if _, err := toml.Decode(content, &lockFile); err != nil {
		return nil, fmt.Errorf("failed to parse poetry.lock: %w", err)
	}

	dependencies := make([]Dependency, 0, len(lockFile.Package))

	for _, pkg := range lockFile.Package {
		depType := "runtime"
		if pkg.Category == "dev" {
			depType = "dev"
		}
		if pkg.Optional {
			depType = "optional"
		}

		dep := Dependency{
			Name:    pkg.Name,
			Version: pkg.Version,
			Type:    depType,
			Source:  "pypi",
		}

		dependencies = append(dependencies, dep)
	}

	return dependencies, nil
}
