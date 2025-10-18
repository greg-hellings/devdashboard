package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level configuration file structure
type Config struct {
	Providers map[string]ProviderConfig `yaml:"providers"`
}

// ProviderConfig contains configuration for a specific repository provider
type ProviderConfig struct {
	Default      RepoDefaults `yaml:"default"`
	Repositories []RepoConfig `yaml:"repositories"`
}

// RepoDefaults contains default values that can be inherited by repositories
type RepoDefaults struct {
	Token      string   `yaml:"token"`
	Owner      string   `yaml:"owner"`
	Repository string   `yaml:"repository"`
	Ref        string   `yaml:"ref"`
	Paths      []string `yaml:"paths"`
	Packages   []string `yaml:"packages"`
	Analyzer   string   `yaml:"analyzer"`
}

// RepoConfig contains configuration for a single repository
type RepoConfig struct {
	Token      string   `yaml:"token"`
	Owner      string   `yaml:"owner"`
	Repository string   `yaml:"repository"`
	Ref        string   `yaml:"ref"`
	Paths      []string `yaml:"paths"`
	Packages   []string `yaml:"packages"`
	Analyzer   string   `yaml:"analyzer"`
}

// LoadFromFile reads a YAML configuration file and returns the parsed Config
func LoadFromFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults to repositories
	if err := config.ApplyDefaults(); err != nil {
		return nil, fmt.Errorf("failed to apply defaults: %w", err)
	}

	return &config, nil
}

// ApplyDefaults applies default values to repositories that don't have them set
func (c *Config) ApplyDefaults() error {
	for providerName, providerConfig := range c.Providers {
		for i := range providerConfig.Repositories {
			repo := &providerConfig.Repositories[i]
			defaults := providerConfig.Default

			// Apply defaults for each field if not set
			if repo.Token == "" {
				repo.Token = defaults.Token
			}
			if repo.Owner == "" {
				repo.Owner = defaults.Owner
			}
			if repo.Ref == "" {
				repo.Ref = defaults.Ref
			}
			if len(repo.Paths) == 0 {
				repo.Paths = defaults.Paths
			}
			if len(repo.Packages) == 0 {
				repo.Packages = defaults.Packages
			}
			if repo.Analyzer == "" {
				repo.Analyzer = defaults.Analyzer
			}

			// Validate required fields
			if repo.Owner == "" {
				return fmt.Errorf("provider %s: repository at index %d missing required field 'owner'", providerName, i)
			}
			if repo.Repository == "" {
				return fmt.Errorf("provider %s: repository at index %d missing required field 'repository'", providerName, i)
			}
			if repo.Analyzer == "" {
				return fmt.Errorf("provider %s: repository at index %d missing required field 'analyzer'", providerName, i)
			}
		}
		c.Providers[providerName] = providerConfig
	}

	return nil
}

// GetAllRepos returns a flat list of all repositories with their provider name
func (c *Config) GetAllRepos() []RepoWithProvider {
	var repos []RepoWithProvider
	for providerName, providerConfig := range c.Providers {
		for _, repo := range providerConfig.Repositories {
			repos = append(repos, RepoWithProvider{
				Provider: providerName,
				Config:   repo,
			})
		}
	}
	return repos
}

// RepoWithProvider combines a repository configuration with its provider name
type RepoWithProvider struct {
	Provider string
	Config   RepoConfig
}
