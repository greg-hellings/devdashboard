// Package report provides types and logic for generating dependency analysis
// reports across multiple repositories. It aggregates analyzer results,
// normalizes package version information, and offers helpers for rendering
// human-readable summaries.
package report

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/greg-hellings/devdashboard/core/pkg/config"
	"github.com/greg-hellings/devdashboard/core/pkg/dependencies"
	"github.com/greg-hellings/devdashboard/core/pkg/repository"
)

// Report contains the results of analyzing dependencies across multiple repositories
type Report struct {
	// Repositories contains the analysis results for each repository
	Repositories []RepositoryReport

	// Packages is the list of packages being tracked across repositories
	Packages []string
}

// RepositoryReport contains dependency information for a single repository
type RepositoryReport struct {
	Provider   string
	Owner      string
	Repository string
	Ref        string
	Analyzer   string

	// Dependencies maps package name to version (empty string if not found)
	Dependencies map[string]string

	// Error contains any error encountered during analysis
	Error error
}

// PackageVersions contains all versions of a package across repositories
type PackageVersions struct {
	PackageName string
	Versions    map[string][]string // version -> list of repo identifiers
}

// Generator generates dependency reports for multiple repositories
type Generator struct {
	depFactory *dependencies.Factory
}

// NewGenerator creates a new report generator
func NewGenerator() *Generator {
	return &Generator{
		depFactory: dependencies.NewFactory(),
	}
}

// Generate creates a dependency report for the given repository configurations
func (g *Generator) Generate(ctx context.Context, repos []config.RepoWithProvider) (*Report, error) {
	slog.Info("Starting dependency report generation", "repoCount", len(repos))

	// Check if context is already canceled
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Collect all unique packages to track
	packageSet := make(map[string]bool)
	for _, repo := range repos {
		for _, pkg := range repo.Config.Packages {
			packageSet[pkg] = true
		}
	}

	packages := make([]string, 0, len(packageSet))
	for pkg := range packageSet {
		packages = append(packages, pkg)
	}
	sort.Strings(packages)

	// Analyze repositories in parallel
	var wg sync.WaitGroup
	repoReports := make([]RepositoryReport, len(repos))

	for i, repo := range repos {
		wg.Add(1)
		go func(index int, r config.RepoWithProvider) {
			defer wg.Done()
			repoReports[index] = g.analyzeRepository(ctx, r)
		}(i, repo)
	}

	wg.Wait()

	// Check if context was canceled during analysis
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	slog.Info("Dependency report generation complete", "repoCount", len(repos))

	return &Report{
		Repositories: repoReports,
		Packages:     packages,
	}, nil
}

// analyzeRepository analyzes a single repository and extracts dependency versions
func (g *Generator) analyzeRepository(ctx context.Context, repo config.RepoWithProvider) RepositoryReport {
	report := RepositoryReport{
		Provider:     repo.Provider,
		Owner:        repo.Config.Owner,
		Repository:   repo.Config.Repository,
		Ref:          repo.Config.Ref,
		Analyzer:     repo.Config.Analyzer,
		Dependencies: make(map[string]string),
	}

	slog.Debug("Analyzing repository",
		"provider", repo.Provider,
		"owner", repo.Config.Owner,
		"repo", repo.Config.Repository,
		"analyzer", repo.Config.Analyzer)

	// Create repository client
	repoFactory := repository.NewFactory(repository.Config{
		Token: repo.Config.Token,
	})
	repoClient, err := repoFactory.CreateClient(repo.Provider)
	if err != nil {
		report.Error = fmt.Errorf("failed to create repository client: %w", err)
		slog.Debug("Failed to create repository client",
			"provider", repo.Provider,
			"error", err)
		return report
	}

	// Create dependency analyzer
	analyzer, err := g.depFactory.CreateAnalyzer(repo.Config.Analyzer)
	if err != nil {
		report.Error = fmt.Errorf("failed to create analyzer: %w", err)
		slog.Debug("Failed to create analyzer",
			"analyzer", repo.Config.Analyzer,
			"error", err)
		return report
	}

	// Configure dependency analyzer
	depConfig := dependencies.Config{
		RepositoryPaths:  repo.Config.Paths,
		RepositoryClient: repoClient,
	}

	// Find dependency files
	var candidates []dependencies.DependencyFile

	if len(repo.Config.Paths) > 0 {
		// User specified explicit paths, use them directly
		slog.Debug("Using user-specified paths",
			"owner", repo.Config.Owner,
			"repo", repo.Config.Repository,
			"paths", repo.Config.Paths)

		for _, path := range repo.Config.Paths {
			candidates = append(candidates, dependencies.DependencyFile{
				Path:     path,
				Type:     repo.Config.Analyzer,
				Analyzer: repo.Config.Analyzer,
			})
		}
	} else {
		// No paths specified, search for candidate files
		slog.Debug("No paths specified, searching for candidate files",
			"owner", repo.Config.Owner,
			"repo", repo.Config.Repository)

		var err error
		candidates, err = analyzer.CandidateFiles(ctx, repo.Config.Owner, repo.Config.Repository, repo.Config.Ref, depConfig)
		if err != nil {
			report.Error = fmt.Errorf("failed to find dependency files: %w", err)
			slog.Debug("Failed to find dependency files",
				"owner", repo.Config.Owner,
				"repo", repo.Config.Repository,
				"error", err)
			return report
		}

		if len(candidates) == 0 {
			report.Error = fmt.Errorf("no dependency files found")
			slog.Debug("No dependency files found",
				"owner", repo.Config.Owner,
				"repo", repo.Config.Repository)
			return report
		}
	}

	slog.Debug("Found dependency files",
		"owner", repo.Config.Owner,
		"repo", repo.Config.Repository,
		"count", len(candidates))

	// Analyze dependencies
	results, err := analyzer.AnalyzeDependencies(ctx, repo.Config.Owner, repo.Config.Repository, repo.Config.Ref, candidates, depConfig)
	if err != nil {
		report.Error = fmt.Errorf("failed to analyze dependencies: %w", err)
		slog.Debug("Failed to analyze dependencies",
			"owner", repo.Config.Owner,
			"repo", repo.Config.Repository,
			"error", err)
		return report
	}

	// Extract versions for requested packages
	for _, deps := range results {
		for _, dep := range deps {
			// Check if this is a package we're tracking
			for _, pkg := range repo.Config.Packages {
				if dep.Name == pkg {
					report.Dependencies[pkg] = dep.Version
					slog.Debug("Found tracked package",
						"package", pkg,
						"version", dep.Version,
						"repo", repo.Config.Repository)
					break
				}
			}
		}
	}

	slog.Debug("Repository analysis complete",
		"owner", repo.Config.Owner,
		"repo", repo.Config.Repository,
		"foundPackages", len(report.Dependencies))

	return report
}

// GetPackageVersions returns version information grouped by package
func (r *Report) GetPackageVersions() []PackageVersions {
	result := make([]PackageVersions, len(r.Packages))

	for i, pkg := range r.Packages {
		pv := PackageVersions{
			PackageName: pkg,
			Versions:    make(map[string][]string),
		}

		for _, repoReport := range r.Repositories {
			repoID := fmt.Sprintf("%s/%s", repoReport.Owner, repoReport.Repository)

			if version, found := repoReport.Dependencies[pkg]; found {
				pv.Versions[version] = append(pv.Versions[version], repoID)
			} else {
				// Package not found in this repository
				pv.Versions[""] = append(pv.Versions[""], repoID)
			}
		}

		result[i] = pv
	}

	return result
}

// GetRepoIdentifier returns a human-readable identifier for a repository report
func (r *RepositoryReport) GetRepoIdentifier() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Repository)
}

// HasErrors returns true if any repository analysis encountered an error
func (r *Report) HasErrors() bool {
	for _, repo := range r.Repositories {
		if repo.Error != nil {
			return true
		}
	}
	return false
}

// GetErrors returns all errors encountered during analysis
func (r *Report) GetErrors() map[string]error {
	errors := make(map[string]error)
	for _, repo := range r.Repositories {
		if repo.Error != nil {
			errors[repo.GetRepoIdentifier()] = repo.Error
		}
	}
	return errors
}
