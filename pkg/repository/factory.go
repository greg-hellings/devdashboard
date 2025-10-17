package repository

import (
	"fmt"
	"strings"
)

// ProviderType represents the type of repository provider
type ProviderType string

const (
	// ProviderGitHub represents GitHub as the repository provider
	ProviderGitHub ProviderType = "github"
	// ProviderGitLab represents GitLab as the repository provider
	ProviderGitLab ProviderType = "gitlab"
)

// Factory creates repository clients based on the provider type
type Factory struct {
	config Config
}

// NewFactory creates a new factory instance with the provided configuration
// The configuration will be applied to all clients created by this factory
func NewFactory(config Config) *Factory {
	return &Factory{
		config: config,
	}
}

// CreateClient creates a new repository client based on the provider name
// The provider parameter is case-insensitive and supports the following values:
//   - "github" or "GitHub" - Creates a GitHub client
//   - "gitlab" or "GitLab" - Creates a GitLab client
//
// Returns an error if the provider name is not recognized or client creation fails
func (f *Factory) CreateClient(provider string) (Client, error) {
	// Normalize provider name to lowercase for comparison
	normalizedProvider := strings.ToLower(strings.TrimSpace(provider))

	switch ProviderType(normalizedProvider) {
	case ProviderGitHub:
		return NewGitHubClient(f.config)
	case ProviderGitLab:
		return NewGitLabClient(f.config)
	default:
		return nil, fmt.Errorf("unsupported provider: %s (supported: github, gitlab)", provider)
	}
}

// NewClient is a convenience function that creates a repository client
// without needing to instantiate a Factory first
// This is useful for simple use cases where you only need one client
func NewClient(provider string, config Config) (Client, error) {
	factory := NewFactory(config)
	return factory.CreateClient(provider)
}

// SupportedProviders returns a list of all supported provider types
func SupportedProviders() []string {
	return []string{
		string(ProviderGitHub),
		string(ProviderGitLab),
	}
}
