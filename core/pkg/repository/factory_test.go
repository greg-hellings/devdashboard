package repository

import (
	"testing"
)

// TestNewFactory verifies that factory creation works correctly
func TestNewFactory(t *testing.T) {
	config := Config{
		Token:   "test-token",
		BaseURL: "https://example.com",
	}

	factory := NewFactory(config)

	if factory == nil {
		t.Fatal("NewFactory returned nil")
		return
	}

	if factory.config.Token != config.Token {
		t.Errorf("Expected token %s, got %s", config.Token, factory.config.Token)
	}

	if factory.config.BaseURL != config.BaseURL {
		t.Errorf("Expected baseURL %s, got %s", config.BaseURL, factory.config.BaseURL)
	}
}

// TestCreateClientGitHub tests creating a GitHub client via factory
func TestCreateClientGitHub(t *testing.T) {
	config := Config{
		Token: "test-token",
	}

	factory := NewFactory(config)
	client, err := factory.CreateClient("github")

	if err != nil {
		t.Fatalf("Failed to create GitHub client: %v", err)
	}

	if client == nil {
		t.Fatal("CreateClient returned nil client")
	}

	// Verify we got the correct type
	if _, ok := client.(*GitHubClient); !ok {
		t.Errorf("Expected *GitHubClient, got %T", client)
	}
}

// TestCreateClientGitLab tests creating a GitLab client via factory
func TestCreateClientGitLab(t *testing.T) {
	config := Config{
		Token: "test-token",
	}

	factory := NewFactory(config)
	client, err := factory.CreateClient("gitlab")

	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	if client == nil {
		t.Fatal("CreateClient returned nil client")
	}

	// Verify we got the correct type
	if _, ok := client.(*GitLabClient); !ok {
		t.Errorf("Expected *GitLabClient, got %T", client)
	}
}

// TestCreateClientCaseInsensitive verifies provider names are case-insensitive
func TestCreateClientCaseInsensitive(t *testing.T) {
	config := Config{}
	factory := NewFactory(config)

	testCases := []struct {
		provider     string
		expectedType interface{}
	}{
		{"GitHub", &GitHubClient{}},
		{"GITHUB", &GitHubClient{}},
		{"github", &GitHubClient{}},
		{"GitLab", &GitLabClient{}},
		{"GITLAB", &GitLabClient{}},
		{"gitlab", &GitLabClient{}},
	}

	for _, tc := range testCases {
		t.Run(tc.provider, func(t *testing.T) {
			client, err := factory.CreateClient(tc.provider)
			if err != nil {
				t.Fatalf("Failed to create client for provider %s: %v", tc.provider, err)
			}

			if client == nil {
				t.Fatalf("CreateClient returned nil for provider %s", tc.provider)
			}

			// Check the type matches expected
			switch tc.expectedType.(type) {
			case *GitHubClient:
				if _, ok := client.(*GitHubClient); !ok {
					t.Errorf("Expected *GitHubClient for %s, got %T", tc.provider, client)
				}
			case *GitLabClient:
				if _, ok := client.(*GitLabClient); !ok {
					t.Errorf("Expected *GitLabClient for %s, got %T", tc.provider, client)
				}
			}
		})
	}
}

// TestCreateClientUnsupportedProvider tests error handling for unsupported providers
func TestCreateClientUnsupportedProvider(t *testing.T) {
	config := Config{}
	factory := NewFactory(config)

	unsupportedProviders := []string{
		"bitbucket",
		"svn",
		"unknown",
		"",
		"   ",
	}

	for _, provider := range unsupportedProviders {
		t.Run(provider, func(t *testing.T) {
			client, err := factory.CreateClient(provider)

			if err == nil {
				t.Errorf("Expected error for unsupported provider %s, got nil", provider)
			}

			if client != nil {
				t.Errorf("Expected nil client for unsupported provider %s, got %T", provider, client)
			}
		})
	}
}

// TestNewClient tests the convenience function for creating clients
func TestNewClient(t *testing.T) {
	config := Config{
		Token: "test-token",
	}

	// Test GitHub
	githubClient, err := NewClient("github", config)
	if err != nil {
		t.Fatalf("NewClient failed for github: %v", err)
	}
	if githubClient == nil {
		t.Fatal("NewClient returned nil for github")
	}
	if _, ok := githubClient.(*GitHubClient); !ok {
		t.Errorf("Expected *GitHubClient, got %T", githubClient)
	}

	// Test GitLab
	gitlabClient, err := NewClient("gitlab", config)
	if err != nil {
		t.Fatalf("NewClient failed for gitlab: %v", err)
	}
	if gitlabClient == nil {
		t.Fatal("NewClient returned nil for gitlab")
	}
	if _, ok := gitlabClient.(*GitLabClient); !ok {
		t.Errorf("Expected *GitLabClient, got %T", gitlabClient)
	}

	// Test unsupported provider
	_, err = NewClient("unsupported", config)
	if err == nil {
		t.Error("Expected error for unsupported provider, got nil")
	}
}

// TestSupportedProviders verifies the list of supported providers
func TestSupportedProviders(t *testing.T) {
	providers := SupportedProviders()

	if len(providers) == 0 {
		t.Fatal("SupportedProviders returned empty list")
	}

	// Check that expected providers are in the list
	expectedProviders := map[string]bool{
		"github": false,
		"gitlab": false,
	}

	for _, provider := range providers {
		if _, exists := expectedProviders[provider]; exists {
			expectedProviders[provider] = true
		}
	}

	// Verify all expected providers were found
	for provider, found := range expectedProviders {
		if !found {
			t.Errorf("Expected provider %s not found in SupportedProviders()", provider)
		}
	}
}

// TestProviderTypeConstants verifies provider type constants
func TestProviderTypeConstants(t *testing.T) {
	if ProviderGitHub != "github" {
		t.Errorf("Expected ProviderGitHub to be 'github', got '%s'", ProviderGitHub)
	}

	if ProviderGitLab != "gitlab" {
		t.Errorf("Expected ProviderGitLab to be 'gitlab', got '%s'", ProviderGitLab)
	}
}

// TestExtractFileName tests the helper function for extracting filenames from paths
func TestExtractFileName(t *testing.T) {
	testCases := []struct {
		path     string
		expected string
	}{
		{"file.txt", "file.txt"},
		{"path/to/file.txt", "file.txt"},
		{"path/to/directory/", ""},
		{"a/b/c/d/e/file.go", "file.go"},
		{"single", "single"},
		{"", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			result := extractFileName(tc.path)
			if result != tc.expected {
				t.Errorf("extractFileName(%s) = %s, expected %s", tc.path, result, tc.expected)
			}
		})
	}
}
