package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greg-hellings/devdashboard/core/pkg/config"
)

func TestNewDefaultGUIState(t *testing.T) {
	state := NewDefaultGUIState()
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.StateVersion != 1 {
		t.Errorf("expected StateVersion 1, got %d", state.StateVersion)
	}
	if state.Providers == nil {
		t.Error("expected Providers map to be initialized")
	}
	if state.GUI.Theme != "light" {
		t.Errorf("expected default theme 'light', got %s", state.GUI.Theme)
	}
}

func TestDefaultGUIStatePath(t *testing.T) {
	path := DefaultGUIStatePath()
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	if !strings.Contains(path, "devdashboard") {
		t.Errorf("expected path to contain 'devdashboard', got %s", path)
	}
	if !strings.HasSuffix(path, ".yaml") {
		t.Errorf("expected path to end with .yaml, got %s", path)
	}
}

func TestUserConfigDir(t *testing.T) {
	dir := userConfigDir()
	if dir == "" {
		t.Fatal("expected non-empty config dir")
	}
	// Should return a valid directory path
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %s", dir)
	}
}

func TestSaveGUIState_LoadGUIState(t *testing.T) {
	// Use DefaultGUIStatePath to get a valid path within config dir
	statePath := DefaultGUIStatePath()

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(statePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	defer os.Remove(statePath)

	// Create and save a state
	state := NewDefaultGUIState()
	state.Profile = "integration-test"
	state.GUI.Theme = "dark"
	state.TrackedPackages = []string{"pkg1", "pkg2", "pkg3"}

	err := SaveGUIState(state, statePath)
	if err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	// Load it back
	loaded, err := LoadGUIState(statePath)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	if loaded.Profile != "integration-test" {
		t.Errorf("expected profile 'integration-test', got %s", loaded.Profile)
	}
	if loaded.GUI.Theme != "dark" {
		t.Errorf("expected theme 'dark', got %s", loaded.GUI.Theme)
	}
	if len(loaded.TrackedPackages) != 3 {
		t.Errorf("expected 3 tracked packages, got %d", len(loaded.TrackedPackages))
	}
}

func TestLoadGUIState_NonExistentFile(t *testing.T) {
	_, err := LoadGUIState("/nonexistent/path/to/state.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestNormalizeGUIState(t *testing.T) {
	state := &GUIState{}
	normalizeGUIState(state)

	if state.Providers == nil {
		t.Error("expected Providers to be initialized")
	}
	if state.GUI.Theme == "" {
		t.Error("expected Theme to be set to default")
	}
	if state.GUI.Concurrency.MaxWorkers == 0 {
		t.Error("expected MaxWorkers to be set")
	}
}

func TestAppendRecentConfig(t *testing.T) {
	state := NewDefaultGUIState()

	t.Run("add first config", func(t *testing.T) {
		state.AppendRecentConfig("/path/to/config1.yaml", 10)
		if len(state.GUI.RecentConfig) != 1 {
			t.Errorf("expected 1 recent config, got %d", len(state.GUI.RecentConfig))
		}
		if state.GUI.RecentConfig[0] != "/path/to/config1.yaml" {
			t.Errorf("unexpected config path: %s", state.GUI.RecentConfig[0])
		}
	})

	t.Run("add multiple configs", func(t *testing.T) {
		state.AppendRecentConfig("/path/to/config2.yaml", 10)
		state.AppendRecentConfig("/path/to/config3.yaml", 10)
		if len(state.GUI.RecentConfig) != 3 {
			t.Errorf("expected 3 recent configs, got %d", len(state.GUI.RecentConfig))
		}
	})

	t.Run("duplicate config moves to front", func(t *testing.T) {
		state.AppendRecentConfig("/path/to/config1.yaml", 10)
		if state.GUI.RecentConfig[0] != "/path/to/config1.yaml" {
			t.Error("expected duplicate to be moved to front")
		}
		if len(state.GUI.RecentConfig) != 3 {
			t.Errorf("expected 3 recent configs (no duplicate), got %d", len(state.GUI.RecentConfig))
		}
	})

	t.Run("empty path ignored", func(t *testing.T) {
		initialLen := len(state.GUI.RecentConfig)
		state.AppendRecentConfig("", 10)
		if len(state.GUI.RecentConfig) != initialLen {
			t.Error("expected empty path to be ignored")
		}
	})

	t.Run("max limit enforced", func(t *testing.T) {
		state := NewDefaultGUIState()
		for i := 0; i < 15; i++ {
			state.AppendRecentConfig(string(rune('a'+i)), 5)
		}
		if len(state.GUI.RecentConfig) > 5 {
			t.Errorf("expected max 5 configs, got %d", len(state.GUI.RecentConfig))
		}
	})
}

func TestRedactedCopy(t *testing.T) {
	state := NewDefaultGUIState()
	state.Credentials = &CredentialSnapshot{
		GitHubToken: "ghp_secrettoken123",
		GitLabToken: "glpat_secrettoken456",
	}

	redacted := state.RedactedCopy()

	if redacted == state {
		t.Error("expected a copy, not the same instance")
	}

	if redacted.Credentials == nil {
		t.Fatal("expected credentials to be present")
	}

	if redacted.Credentials.GitHubToken == "ghp_secrettoken123" {
		t.Error("expected GitHub token to be redacted")
	}

	if redacted.Credentials.GitLabToken == "glpat_secrettoken456" {
		t.Error("expected GitLab token to be redacted")
	}

	if !strings.Contains(redacted.Credentials.GitHubToken, "***") {
		t.Error("expected redacted token to contain ***")
	}
}

func TestRedactTokenInternal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"short", "abc", "***"},
		{"exactly 4 chars", "abcd", "***"},
		{"normal token", "ghp_1234567890", "ghp_***"},
		{"long token", "glpat_verylongtoken", "glpa***"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactToken(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMergeCLIConfig(t *testing.T) {
	// Create a temporary config file
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write a simple config
	configContent := `providers:
  github:
    default:
      owner: testowner
      ref: main
    repositories:
      - owner: testowner
        repository: testrepo
        ref: main
        analyzer: go
        packages:
          - pkg1
          - pkg2
`
	if _, err := tmpfile.WriteString(configContent); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	tmpfile.Close()

	state := NewDefaultGUIState()
	err = state.MergeCLIConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("MergeCLIConfig failed: %v", err)
	}

	if len(state.Providers) == 0 {
		t.Fatal("expected providers to be merged")
	}

	if _, ok := state.Providers["github"]; !ok {
		t.Error("expected github provider to be present")
	}

	// Check that recent config was added
	if len(state.GUI.RecentConfig) == 0 {
		t.Error("expected recent config to be updated")
	}
}

func TestRebuildRepositoriesCache(t *testing.T) {
	state := NewDefaultGUIState()
	state.Providers = map[string]ProviderConfigWrapper{
		"github": {
			Default: config.RepoDefaults{
				Owner: "defaultowner",
				Ref:   "main",
			},
			Repositories: []config.RepoConfig{
				{
					Owner:      "owner1",
					Repository: "repo1",
					Ref:        "main",
					Analyzer:   "go",
				},
			},
		},
		"gitlab": {
			Repositories: []config.RepoConfig{
				{
					Owner:      "owner2",
					Repository: "repo2",
					Ref:        "develop",
					Analyzer:   "python",
				},
			},
		},
	}

	state.RebuildRepositoriesCache()

	if len(state.RepositoriesCache) != 2 {
		t.Errorf("expected 2 cached repositories, got %d", len(state.RepositoriesCache))
	}

	// Verify entries have correct data
	for _, entry := range state.RepositoriesCache {
		if entry.Provider == "" {
			t.Error("expected provider to be set")
		}
		if entry.Owner == "" {
			t.Error("expected owner to be set")
		}
		if entry.Repository == "" {
			t.Error("expected repository to be set")
		}
	}
}

func TestRepoCacheKey(t *testing.T) {
	key1 := repoCacheKey("github", "owner", "repo", "main")
	key2 := repoCacheKey("github", "owner", "repo", "develop")
	key3 := repoCacheKey("gitlab", "owner", "repo", "main")

	if key1 == "" {
		t.Error("expected non-empty key")
	}

	if key1 == key2 {
		t.Error("expected different keys for different refs")
	}

	if key1 == key3 {
		t.Error("expected different keys for different providers")
	}

	if !strings.Contains(key1, "github") {
		t.Error("expected key to contain provider")
	}
	if !strings.Contains(key1, "owner") {
		t.Error("expected key to contain owner")
	}
	if !strings.Contains(key1, "repo") {
		t.Error("expected key to contain repo")
	}
}

func TestGUIState_WriteTo(t *testing.T) {
	state := NewDefaultGUIState()
	state.Profile = "test-profile"
	state.TrackedPackages = []string{"pkg1", "pkg2"}

	var buf strings.Builder
	n, err := state.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	if n == 0 {
		t.Error("expected non-zero bytes written")
	}

	output := buf.String()
	if !strings.Contains(output, "test-profile") {
		t.Error("expected output to contain profile")
	}
	if !strings.Contains(output, "pkg1") {
		t.Error("expected output to contain tracked packages")
	}
}

func TestTrimForCLIExport(t *testing.T) {
	state := NewDefaultGUIState()
	state.Profile = "test-profile"
	state.GUI.Theme = "dark"
	state.GUI.RecentConfig = []string{"/path/to/config.yaml"}
	state.Credentials = &CredentialSnapshot{
		GitHubToken: "ghp_secret",
	}
	state.ErrorLog = []ErrorLogEntry{
		{Time: time.Now(), Message: "test error"},
	}

	trimmed := state.TrimForCLIExport()

	if trimmed.GUI.Theme != "" {
		t.Error("expected GUI section to be cleared")
	}
	if len(trimmed.GUI.RecentConfig) != 0 {
		t.Error("expected RecentConfig to be cleared")
	}
	if trimmed.Credentials != nil {
		t.Error("expected Credentials to be cleared")
	}
	if len(trimmed.ErrorLog) != 0 {
		t.Error("expected ErrorLog to be cleared")
	}
	if len(trimmed.RepositoriesCache) != 0 {
		t.Error("expected RepositoriesCache to be cleared")
	}

	// Providers should be preserved
	if len(trimmed.Providers) != len(state.Providers) {
		t.Error("expected Providers to be preserved")
	}
}

func TestPendingMigrations(t *testing.T) {
	t.Run("no migrations needed", func(t *testing.T) {
		state := NewDefaultGUIState()
		state.StateVersion = 1
		pending := state.PendingMigrations()
		if pending {
			t.Error("expected no pending migrations")
		}
	})

	t.Run("check returns bool", func(t *testing.T) {
		state := NewDefaultGUIState()
		state.StateVersion = 0
		pending := state.PendingMigrations()
		// Currently always returns false (reserved for future)
		if pending {
			t.Error("expected false for current implementation")
		}
	})
}

func TestApplyMigrations(t *testing.T) {
	t.Run("apply to old version", func(t *testing.T) {
		state := NewDefaultGUIState()
		state.StateVersion = 0

		err := state.ApplyMigrations()
		if err != nil {
			t.Fatalf("ApplyMigrations failed: %v", err)
		}

		// Currently a no-op, reserved for future migrations
		// So version stays at 0
		if state.StateVersion != 0 {
			t.Errorf("expected StateVersion to remain 0 (no migrations yet), got %d", state.StateVersion)
		}
	})

	t.Run("no-op for current version", func(t *testing.T) {
		state := NewDefaultGUIState()
		originalVersion := state.StateVersion

		err := state.ApplyMigrations()
		if err != nil {
			t.Fatalf("ApplyMigrations failed: %v", err)
		}

		if state.StateVersion != originalVersion {
			t.Error("expected version to remain unchanged")
		}
	})
}

func TestGUIState_SaveTimestamp(t *testing.T) {
	// Use DefaultGUIStatePath to get a valid path within config dir
	statePath := DefaultGUIStatePath()

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(statePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	defer os.Remove(statePath)

	state := NewDefaultGUIState()
	beforeSave := time.Now()

	err := SaveGUIState(state, statePath)
	if err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	afterSave := time.Now()

	// Reload and check timestamp
	loaded, err := LoadGUIState(statePath)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	if loaded.SavedAt.Before(beforeSave) || loaded.SavedAt.After(afterSave) {
		t.Error("SavedAt timestamp not within expected range")
	}
}

func TestGUIState_NilCredentials(t *testing.T) {
	state := NewDefaultGUIState()
	state.Credentials = nil

	// Should not panic when credentials are nil
	redacted := state.RedactedCopy()
	if redacted.Credentials != nil {
		t.Error("expected nil credentials to remain nil")
	}
}

func TestProviderConfigWrapper_Structure(t *testing.T) {
	wrapper := ProviderConfigWrapper{
		Default: config.RepoDefaults{
			Owner: "testowner",
			Ref:   "main",
		},
		Repositories: []config.RepoConfig{
			{
				Owner:      "owner",
				Repository: "repo",
				Ref:        "main",
				Analyzer:   "go",
			},
		},
	}

	if wrapper.Default.Owner != "testowner" {
		t.Error("expected default owner to be set")
	}
	if len(wrapper.Repositories) != 1 {
		t.Error("expected 1 repository")
	}
}

func TestRepoCacheEntry_Structure(t *testing.T) {
	entry := RepoCacheEntry{
		Provider:   "github",
		Owner:      "testowner",
		Repository: "testrepo",
		Ref:        "main",
		Analyzer:   "go",
		Packages:   []string{"pkg1", "pkg2"},
	}

	if entry.Provider != "github" {
		t.Error("expected provider to be set")
	}
	if len(entry.Packages) != 2 {
		t.Error("expected 2 packages")
	}
}

func TestErrorLogEntry_Structure(t *testing.T) {
	now := time.Now()
	entry := ErrorLogEntry{
		Time:     now,
		Source:   "test",
		Severity: "error",
		Message:  "test message",
		Details:  "test details",
	}

	if entry.Source != "test" {
		t.Error("expected source to be set")
	}
	if !entry.Time.Equal(now) {
		t.Error("expected time to match")
	}
}
