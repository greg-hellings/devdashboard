package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		validateFn  func(*testing.T, *Config)
		description string
	}{
		{
			name: "valid config with defaults",
			content: `
providers:
  github:
    default:
      token: "test-token"
      owner: "test-owner"
      repository: "default-repo"
      ref: "main"
      analyzer: "poetry"
      paths:
        - "src"
      packages:
        - "requests"
        - "pytest"
    repositories:
      - repository: "repo1"
      - repository: "repo2"
        owner: "other-owner"
`,
			wantErr:     false,
			description: "Should load valid config and apply defaults",
			validateFn: func(t *testing.T, cfg *Config) {
				if len(cfg.Providers) != 1 {
					t.Errorf("Expected 1 provider, got %d", len(cfg.Providers))
				}

				github, ok := cfg.Providers["github"]
				if !ok {
					t.Fatal("GitHub provider not found")
				}

				if len(github.Repositories) != 2 {
					t.Errorf("Expected 2 repositories, got %d", len(github.Repositories))
				}

				// Check first repo inherited defaults
				repo1 := github.Repositories[0]
				if repo1.Repository != "repo1" {
					t.Errorf("Expected repository 'repo1', got '%s'", repo1.Repository)
				}
				if repo1.Owner != "test-owner" {
					t.Errorf("Expected owner 'test-owner', got '%s'", repo1.Owner)
				}
				if repo1.Token != "test-token" {
					t.Errorf("Expected token 'test-token', got '%s'", repo1.Token)
				}
				if repo1.Ref != "main" {
					t.Errorf("Expected ref 'main', got '%s'", repo1.Ref)
				}
				if repo1.Analyzer != "poetry" {
					t.Errorf("Expected analyzer 'poetry', got '%s'", repo1.Analyzer)
				}
				if len(repo1.Paths) != 1 || repo1.Paths[0] != "src" {
					t.Errorf("Expected paths ['src'], got %v", repo1.Paths)
				}
				if len(repo1.Packages) != 2 {
					t.Errorf("Expected 2 packages, got %d", len(repo1.Packages))
				}

				// Check second repo overrode owner
				repo2 := github.Repositories[1]
				if repo2.Owner != "other-owner" {
					t.Errorf("Expected owner 'other-owner', got '%s'", repo2.Owner)
				}
				// But still inherits other defaults
				if repo2.Token != "test-token" {
					t.Errorf("Expected token 'test-token', got '%s'", repo2.Token)
				}
			},
		},
		{
			name: "multiple providers",
			content: `
providers:
  github:
    default:
      owner: "gh-owner"
      analyzer: "poetry"
      packages: ["requests"]
    repositories:
      - repository: "gh-repo"
  gitlab:
    default:
      owner: "gl-owner"
      analyzer: "pipfile"
      packages: ["django"]
    repositories:
      - repository: "gl-repo"
`,
			wantErr:     false,
			description: "Should handle multiple providers",
			validateFn: func(t *testing.T, cfg *Config) {
				if len(cfg.Providers) != 2 {
					t.Errorf("Expected 2 providers, got %d", len(cfg.Providers))
				}

				if _, ok := cfg.Providers["github"]; !ok {
					t.Error("GitHub provider not found")
				}
				if _, ok := cfg.Providers["gitlab"]; !ok {
					t.Error("GitLab provider not found")
				}
			},
		},
		{
			name: "repository overrides all defaults",
			content: `
providers:
  github:
    default:
      token: "default-token"
      owner: "default-owner"
      ref: "main"
      analyzer: "poetry"
      paths: ["src"]
      packages: ["requests"]
    repositories:
      - repository: "custom-repo"
        token: "custom-token"
        owner: "custom-owner"
        ref: "develop"
        analyzer: "pipfile"
        paths: ["lib"]
        packages: ["django"]
`,
			wantErr:     false,
			description: "Repository should be able to override all defaults",
			validateFn: func(t *testing.T, cfg *Config) {
				repo := cfg.Providers["github"].Repositories[0]
				if repo.Token != "custom-token" {
					t.Errorf("Expected token 'custom-token', got '%s'", repo.Token)
				}
				if repo.Owner != "custom-owner" {
					t.Errorf("Expected owner 'custom-owner', got '%s'", repo.Owner)
				}
				if repo.Ref != "develop" {
					t.Errorf("Expected ref 'develop', got '%s'", repo.Ref)
				}
				if repo.Analyzer != "pipfile" {
					t.Errorf("Expected analyzer 'pipfile', got '%s'", repo.Analyzer)
				}
				if len(repo.Paths) != 1 || repo.Paths[0] != "lib" {
					t.Errorf("Expected paths ['lib'], got %v", repo.Paths)
				}
				if len(repo.Packages) != 1 || repo.Packages[0] != "django" {
					t.Errorf("Expected packages ['django'], got %v", repo.Packages)
				}
			},
		},
		{
			name: "missing required owner",
			content: `
providers:
  github:
    default:
      analyzer: "poetry"
    repositories:
      - repository: "repo1"
`,
			wantErr:     true,
			description: "Should error when owner is missing",
		},
		{
			name: "missing required repository",
			content: `
providers:
  github:
    default:
      owner: "test-owner"
      analyzer: "poetry"
    repositories:
      - owner: "test-owner"
`,
			wantErr:     true,
			description: "Should error when repository name is missing",
		},
		{
			name: "missing required analyzer",
			content: `
providers:
  github:
    default:
      owner: "test-owner"
    repositories:
      - repository: "repo1"
`,
			wantErr:     true,
			description: "Should error when analyzer is missing",
		},
		{
			name:        "invalid yaml",
			content:     `invalid: yaml: content: {{{`,
			wantErr:     true,
			description: "Should error on invalid YAML",
		},
		{
			name: "empty paths and packages",
			content: `
providers:
  github:
    default:
      owner: "test-owner"
      analyzer: "poetry"
    repositories:
      - repository: "repo1"
`,
			wantErr:     false,
			description: "Should handle empty paths and packages",
			validateFn: func(t *testing.T, cfg *Config) {
				repo := cfg.Providers["github"].Repositories[0]
				if len(repo.Paths) > 0 {
					t.Errorf("Expected empty paths, got %v", repo.Paths)
				}
				if len(repo.Packages) > 0 {
					t.Errorf("Expected empty packages, got %v", repo.Packages)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "config.yaml")

			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			// Load config
			cfg, err := LoadFromFile(tmpFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validateFn != nil {
				tt.validateFn(t, cfg)
			}
		})
	}
}

func TestLoadFromFile_FileNotFound(t *testing.T) {
	_, err := LoadFromFile("nonexistent-file.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestGetAllRepos(t *testing.T) {
	cfg := &Config{
		Providers: map[string]ProviderConfig{
			"github": {
				Default: RepoDefaults{
					Owner:    "gh-owner",
					Analyzer: "poetry",
				},
				Repositories: []RepoConfig{
					{Repository: "repo1"},
					{Repository: "repo2"},
				},
			},
			"gitlab": {
				Default: RepoDefaults{
					Owner:    "gl-owner",
					Analyzer: "pipfile",
				},
				Repositories: []RepoConfig{
					{Repository: "repo3"},
				},
			},
		},
	}

	// Apply defaults
	if err := cfg.ApplyDefaults(); err != nil {
		t.Fatalf("Failed to apply defaults: %v", err)
	}

	repos := cfg.GetAllRepos()

	if len(repos) != 3 {
		t.Errorf("Expected 3 repos, got %d", len(repos))
	}

	// Check providers are set correctly
	providerCount := make(map[string]int)
	for _, repo := range repos {
		providerCount[repo.Provider]++
	}

	if providerCount["github"] != 2 {
		t.Errorf("Expected 2 GitHub repos, got %d", providerCount["github"])
	}
	if providerCount["gitlab"] != 1 {
		t.Errorf("Expected 1 GitLab repo, got %d", providerCount["gitlab"])
	}
}

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		check   func(*testing.T, *Config)
	}{
		{
			name: "applies all defaults",
			config: &Config{
				Providers: map[string]ProviderConfig{
					"github": {
						Default: RepoDefaults{
							Token:    "token",
							Owner:    "owner",
							Ref:      "main",
							Paths:    []string{"src"},
							Packages: []string{"pkg1"},
							Analyzer: "poetry",
						},
						Repositories: []RepoConfig{
							{Repository: "repo1"},
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				repo := cfg.Providers["github"].Repositories[0]
				if repo.Token != "token" {
					t.Error("Token not applied")
				}
				if repo.Owner != "owner" {
					t.Error("Owner not applied")
				}
				if repo.Ref != "main" {
					t.Error("Ref not applied")
				}
				if len(repo.Paths) != 1 {
					t.Error("Paths not applied")
				}
				if len(repo.Packages) != 1 {
					t.Error("Packages not applied")
				}
				if repo.Analyzer != "poetry" {
					t.Error("Analyzer not applied")
				}
			},
		},
		{
			name: "preserves repository values over defaults",
			config: &Config{
				Providers: map[string]ProviderConfig{
					"github": {
						Default: RepoDefaults{
							Token:    "default-token",
							Owner:    "default-owner",
							Analyzer: "poetry",
						},
						Repositories: []RepoConfig{
							{
								Repository: "repo1",
								Token:      "repo-token",
								Owner:      "repo-owner",
							},
						},
					},
				},
			},
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				repo := cfg.Providers["github"].Repositories[0]
				if repo.Token != "repo-token" {
					t.Error("Repository token should not be overridden")
				}
				if repo.Owner != "repo-owner" {
					t.Error("Repository owner should not be overridden")
				}
				if repo.Analyzer != "poetry" {
					t.Error("Default analyzer should be applied")
				}
			},
		},
		{
			name: "error on missing owner",
			config: &Config{
				Providers: map[string]ProviderConfig{
					"github": {
						Default: RepoDefaults{
							Analyzer: "poetry",
						},
						Repositories: []RepoConfig{
							{Repository: "repo1"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error on missing repository name",
			config: &Config{
				Providers: map[string]ProviderConfig{
					"github": {
						Default: RepoDefaults{
							Owner:    "owner",
							Analyzer: "poetry",
						},
						Repositories: []RepoConfig{
							{Owner: "owner"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "error on missing analyzer",
			config: &Config{
				Providers: map[string]ProviderConfig{
					"github": {
						Default: RepoDefaults{
							Owner: "owner",
						},
						Repositories: []RepoConfig{
							{Repository: "repo1"},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ApplyDefaults()

			if (err != nil) != tt.wantErr {
				t.Errorf("ApplyDefaults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, tt.config)
			}
		})
	}
}

func TestRepoWithProvider_Structure(t *testing.T) {
	rwp := RepoWithProvider{
		Provider: "github",
		Config: RepoConfig{
			Owner:      "test-owner",
			Repository: "test-repo",
			Analyzer:   "poetry",
		},
	}

	if rwp.Provider != "github" {
		t.Errorf("Expected provider 'github', got '%s'", rwp.Provider)
	}
	if rwp.Config.Owner != "test-owner" {
		t.Errorf("Expected owner 'test-owner', got '%s'", rwp.Config.Owner)
	}
}
