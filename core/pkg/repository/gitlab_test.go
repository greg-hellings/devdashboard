package repository

import (
	"encoding/base64"
	"testing"
)

// TestBase64Decoding verifies that base64 content is properly decoded
func TestBase64Decoding(t *testing.T) {
	// Sample content that might come from GitLab
	originalContent := "Hello, World!\nThis is a test file.\n"

	// Encode to base64 (simulating what GitLab returns)
	encodedContent := base64.StdEncoding.EncodeToString([]byte(originalContent))

	// Decode it (simulating what our client should do)
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedContent)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	decodedContent := string(decodedBytes)

	// Verify the decoded content matches the original
	if decodedContent != originalContent {
		t.Errorf("Decoded content doesn't match original.\nExpected: %q\nGot: %q", originalContent, decodedContent)
	}
}

// TestBase64DecodingMultiline verifies decoding of multi-line content
func TestBase64DecodingMultiline(t *testing.T) {
	originalContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	// Encode to base64
	encodedContent := base64.StdEncoding.EncodeToString([]byte(originalContent))

	// Decode it
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedContent)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	decodedContent := string(decodedBytes)

	// Verify
	if decodedContent != originalContent {
		t.Errorf("Decoded multi-line content doesn't match.\nExpected length: %d\nGot length: %d", len(originalContent), len(decodedContent))
	}
}

// TestBase64DecodingUnicode verifies decoding of content with unicode characters
func TestBase64DecodingUnicode(t *testing.T) {
	originalContent := "Hello 世界! 🌍 Привет мир!"

	// Encode to base64
	encodedContent := base64.StdEncoding.EncodeToString([]byte(originalContent))

	// Decode it
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedContent)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}

	decodedContent := string(decodedBytes)

	// Verify
	if decodedContent != originalContent {
		t.Errorf("Decoded unicode content doesn't match.\nExpected: %q\nGot: %q", originalContent, decodedContent)
	}
}

// TestBase64DecodingEmpty verifies handling of empty content
func TestBase64DecodingEmpty(t *testing.T) {
	originalContent := ""

	// Encode to base64
	encodedContent := base64.StdEncoding.EncodeToString([]byte(originalContent))

	// Decode it
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedContent)
	if err != nil {
		t.Fatalf("Failed to decode empty base64: %v", err)
	}

	decodedContent := string(decodedBytes)

	// Verify
	if decodedContent != originalContent {
		t.Errorf("Decoded empty content doesn't match.\nExpected: %q\nGot: %q", originalContent, decodedContent)
	}
}

// TestNewGitLabClient verifies GitLab client creation
func TestNewGitLabClient(t *testing.T) {
	config := Config{
		Token:   "test-token",
		BaseURL: "https://gitlab.example.com",
	}

	client, err := NewGitLabClient(config)
	if err != nil {
		t.Fatalf("Failed to create GitLab client: %v", err)
	}

	if client == nil {
		t.Fatal("NewGitLabClient returned nil client")
		return
	}

	if client.config.Token != config.Token {
		t.Errorf("Expected token %s, got %s", config.Token, client.config.Token)
	}

	if client.config.BaseURL != config.BaseURL {
		t.Errorf("Expected baseURL %s, got %s", config.BaseURL, client.config.BaseURL)
	}
}

// TestNewGitLabClientWithoutToken verifies client creation without authentication
func TestNewGitLabClientWithoutToken(t *testing.T) {
	config := Config{}

	client, err := NewGitLabClient(config)
	if err != nil {
		t.Fatalf("Failed to create GitLab client without token: %v", err)
	}

	if client == nil {
		t.Fatal("NewGitLabClient returned nil client")
	}
}

// TestGitLabClient_getProjectURL verifies project URL construction
func TestGitLabClient_getProjectURL(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		owner       string
		repo        string
		expectedURL string
	}{
		{
			name:        "gitlab.com default",
			config:      Config{},
			owner:       "gitlab-org",
			repo:        "gitlab",
			expectedURL: "https://gitlab.com/gitlab-org/gitlab",
		},
		{
			name: "self-hosted instance",
			config: Config{
				BaseURL: "https://gitlab.example.com",
			},
			owner:       "myorg",
			repo:        "myproject",
			expectedURL: "https://gitlab.example.com/myorg/myproject",
		},
		{
			name: "self-hosted with port",
			config: Config{
				BaseURL: "https://gitlab.company.com:8080",
			},
			owner:       "team",
			repo:        "app",
			expectedURL: "https://gitlab.company.com:8080/team/app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewGitLabClient(tt.config)
			if err != nil {
				t.Fatalf("Failed to create GitLab client: %v", err)
			}

			url := client.getProjectURL(tt.owner, tt.repo)
			if url != tt.expectedURL {
				t.Errorf("Expected URL %s, got %s", tt.expectedURL, url)
			}
		})
	}
}

// TestNewGitHubClient verifies GitHub client creation
func TestNewGitHubClient(t *testing.T) {
	config := Config{
		Token: "test-token",
	}

	client, err := NewGitHubClient(config)
	if err != nil {
		t.Fatalf("Failed to create GitHub client: %v", err)
	}

	if client == nil {
		t.Fatal("NewGitHubClient returned nil client")
		return
	}

	if client.config.Token != config.Token {
		t.Errorf("Expected token %s, got %s", config.Token, client.config.Token)
	}
}

// TestNewGitHubClientWithoutToken verifies client creation without authentication
func TestNewGitHubClientWithoutToken(t *testing.T) {
	config := Config{}

	client, err := NewGitHubClient(config)
	if err != nil {
		t.Fatalf("Failed to create GitHub client without token: %v", err)
	}

	if client == nil {
		t.Fatal("NewGitHubClient returned nil client")
	}
}
