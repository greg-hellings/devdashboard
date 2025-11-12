package state

import (
	"errors"
	"os"
	"testing"
)

func TestInMemoryCredentialStore_SetToken(t *testing.T) {
	store := NewInMemoryCredentialStore()

	t.Run("set valid token", func(t *testing.T) {
		err := store.SetToken("github", "ghp_test123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("set empty provider", func(t *testing.T) {
		err := store.SetToken("", "token")
		if err == nil {
			t.Fatal("expected error for empty provider")
		}
	})

	t.Run("update existing token", func(t *testing.T) {
		err := store.SetToken("github", "ghp_old")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = store.SetToken("github", "ghp_new")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		token, err := store.GetToken("github")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghp_new" {
			t.Errorf("expected ghp_new, got %s", token)
		}
	})
}

func TestInMemoryCredentialStore_GetToken(t *testing.T) {
	store := NewInMemoryCredentialStore()

	t.Run("get non-existent token", func(t *testing.T) {
		_, err := store.GetToken("github")
		if !errors.Is(err, ErrCredentialNotFound) {
			t.Errorf("expected ErrCredentialNotFound, got %v", err)
		}
	})

	t.Run("get existing token", func(t *testing.T) {
		err := store.SetToken("gitlab", "glpat_test456")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		token, err := store.GetToken("gitlab")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "glpat_test456" {
			t.Errorf("expected glpat_test456, got %s", token)
		}
	})
}

func TestInMemoryCredentialStore_DeleteToken(t *testing.T) {
	store := NewInMemoryCredentialStore()

	t.Run("delete non-existent token", func(t *testing.T) {
		err := store.DeleteToken("github")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("delete existing token", func(t *testing.T) {
		err := store.SetToken("github", "ghp_test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = store.DeleteToken("github")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, err = store.GetToken("github")
		if !errors.Is(err, ErrCredentialNotFound) {
			t.Errorf("expected ErrCredentialNotFound after delete, got %v", err)
		}
	})
}

func TestInMemoryCredentialStore_ListProviders(t *testing.T) {
	store := NewInMemoryCredentialStore()

	t.Run("empty store", func(t *testing.T) {
		providers, err := store.ListProviders()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(providers) != 0 {
			t.Errorf("expected 0 providers, got %d", len(providers))
		}
	})

	t.Run("with multiple tokens", func(t *testing.T) {
		store.SetToken("github", "ghp_test")
		store.SetToken("gitlab", "glpat_test")
		providers, err := store.ListProviders()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(providers) != 2 {
			t.Errorf("expected 2 providers, got %d", len(providers))
		}
		// Check both providers are present
		hasGitHub := false
		hasGitLab := false
		for _, p := range providers {
			if p == "github" {
				hasGitHub = true
			}
			if p == "gitlab" {
				hasGitLab = true
			}
		}
		if !hasGitHub || !hasGitLab {
			t.Error("expected both github and gitlab providers")
		}
	})
}

func TestFallbackCredentialStore_SetToken(t *testing.T) {
	t.Run("primary succeeds", func(t *testing.T) {
		primary := NewInMemoryCredentialStore()
		fallback := NewInMemoryCredentialStore()
		store := NewFallbackCredentialStore(primary, fallback)

		err := store.SetToken("github", "ghp_test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify token is in primary
		token, err := primary.GetToken("github")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghp_test" {
			t.Errorf("expected ghp_test, got %s", token)
		}
	})

	t.Run("primary fails, fallback succeeds", func(t *testing.T) {
		primary := &failingCredentialStore{}
		fallback := NewInMemoryCredentialStore()
		store := NewFallbackCredentialStore(primary, fallback)

		err := store.SetToken("github", "ghp_test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify token is in fallback
		token, err := fallback.GetToken("github")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghp_test" {
			t.Errorf("expected ghp_test, got %s", token)
		}
	})

	t.Run("nil primary uses fallback", func(t *testing.T) {
		fallback := NewInMemoryCredentialStore()
		store := NewFallbackCredentialStore(nil, fallback)

		err := store.SetToken("github", "ghp_test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		token, err := fallback.GetToken("github")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghp_test" {
			t.Errorf("expected ghp_test, got %s", token)
		}
	})
}

func TestFallbackCredentialStore_GetToken(t *testing.T) {
	t.Run("primary has token", func(t *testing.T) {
		primary := NewInMemoryCredentialStore()
		fallback := NewInMemoryCredentialStore()
		store := NewFallbackCredentialStore(primary, fallback)

		primary.SetToken("github", "ghp_primary")
		fallback.SetToken("github", "ghp_fallback")

		token, err := store.GetToken("github")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghp_primary" {
			t.Errorf("expected primary token, got %s", token)
		}
	})

	t.Run("primary not found, fallback has token", func(t *testing.T) {
		primary := NewInMemoryCredentialStore()
		fallback := NewInMemoryCredentialStore()
		store := NewFallbackCredentialStore(primary, fallback)

		fallback.SetToken("github", "ghp_fallback")

		token, err := store.GetToken("github")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghp_fallback" {
			t.Errorf("expected fallback token, got %s", token)
		}
	})

	t.Run("both not found", func(t *testing.T) {
		primary := NewInMemoryCredentialStore()
		fallback := NewInMemoryCredentialStore()
		store := NewFallbackCredentialStore(primary, fallback)

		_, err := store.GetToken("github")
		if !errors.Is(err, ErrCredentialNotFound) {
			t.Errorf("expected ErrCredentialNotFound, got %v", err)
		}
	})

	t.Run("primary error non-not-found", func(t *testing.T) {
		primary := &failingCredentialStore{getErr: errors.New("primary error")}
		fallback := NewInMemoryCredentialStore()
		store := NewFallbackCredentialStore(primary, fallback)

		_, err := store.GetToken("github")
		if err == nil {
			t.Fatal("expected error")
		}
		if err.Error() != "primary get token: primary error" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFallbackCredentialStore_DeleteToken(t *testing.T) {
	t.Run("delete from both stores", func(t *testing.T) {
		primary := NewInMemoryCredentialStore()
		fallback := NewInMemoryCredentialStore()
		store := NewFallbackCredentialStore(primary, fallback)

		primary.SetToken("github", "ghp_primary")
		fallback.SetToken("github", "ghp_fallback")

		err := store.DeleteToken("github")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify deleted from both
		_, err = primary.GetToken("github")
		if !errors.Is(err, ErrCredentialNotFound) {
			t.Error("expected token deleted from primary")
		}
		_, err = fallback.GetToken("github")
		if !errors.Is(err, ErrCredentialNotFound) {
			t.Error("expected token deleted from fallback")
		}
	})
}

func TestFallbackCredentialStore_ListProviders(t *testing.T) {
	t.Run("merge providers from both stores", func(t *testing.T) {
		primary := NewInMemoryCredentialStore()
		fallback := NewInMemoryCredentialStore()
		store := NewFallbackCredentialStore(primary, fallback)

		primary.SetToken("github", "ghp_primary")
		fallback.SetToken("gitlab", "glpat_fallback")
		fallback.SetToken("bitbucket", "bb_fallback")

		providers, err := store.ListProviders()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(providers) != 3 {
			t.Errorf("expected 3 providers, got %d", len(providers))
		}
	})

	t.Run("deduplicate providers", func(t *testing.T) {
		primary := NewInMemoryCredentialStore()
		fallback := NewInMemoryCredentialStore()
		store := NewFallbackCredentialStore(primary, fallback)

		primary.SetToken("github", "ghp_primary")
		fallback.SetToken("github", "ghp_fallback")

		providers, err := store.ListProviders()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(providers) != 1 {
			t.Errorf("expected 1 provider (deduplicated), got %d", len(providers))
		}
	})
}

func TestStubCredentialStore(t *testing.T) {
	store := StubCredentialStore{}

	t.Run("SetToken is no-op", func(t *testing.T) {
		err := store.SetToken("github", "token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("GetToken always returns not found", func(t *testing.T) {
		_, err := store.GetToken("github")
		if !errors.Is(err, ErrCredentialNotFound) {
			t.Errorf("expected ErrCredentialNotFound, got %v", err)
		}
	})

	t.Run("DeleteToken is no-op", func(t *testing.T) {
		err := store.DeleteToken("github")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ListProviders returns empty", func(t *testing.T) {
		providers, err := store.ListProviders()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(providers) != 0 {
			t.Errorf("expected 0 providers, got %d", len(providers))
		}
	})
}

func TestResolveProviderToken(t *testing.T) {
	t.Run("empty provider", func(t *testing.T) {
		_, err := ResolveProviderToken("", nil, nil)
		if err == nil {
			t.Fatal("expected error for empty provider")
		}
	})

	t.Run("from environment variable", func(t *testing.T) {
		os.Setenv("DEV_DASHBOARD_GITHUB_TOKEN", "ghp_env")
		defer os.Unsetenv("DEV_DASHBOARD_GITHUB_TOKEN")

		token, err := ResolveProviderToken("github", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghp_env" {
			t.Errorf("expected ghp_env, got %s", token)
		}
	})

	t.Run("from GUIState github", func(t *testing.T) {
		state := &GUIState{
			Credentials: &CredentialSnapshot{
				GitHubToken: "ghp_state",
			},
		}

		token, err := ResolveProviderToken("github", state, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghp_state" {
			t.Errorf("expected ghp_state, got %s", token)
		}
	})

	t.Run("from GUIState gitlab", func(t *testing.T) {
		state := &GUIState{
			Credentials: &CredentialSnapshot{
				GitLabToken: "glpat_state",
			},
		}

		token, err := ResolveProviderToken("gitlab", state, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "glpat_state" {
			t.Errorf("expected glpat_state, got %s", token)
		}
	})

	t.Run("from credential store", func(t *testing.T) {
		store := NewInMemoryCredentialStore()
		store.SetToken("github", "ghp_store")

		token, err := ResolveProviderToken("github", nil, store)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghp_store" {
			t.Errorf("expected ghp_store, got %s", token)
		}
	})

	t.Run("priority order: env > state > store", func(t *testing.T) {
		os.Setenv("DEV_DASHBOARD_GITHUB_TOKEN", "ghp_env")
		defer os.Unsetenv("DEV_DASHBOARD_GITHUB_TOKEN")

		state := &GUIState{
			Credentials: &CredentialSnapshot{
				GitHubToken: "ghp_state",
			},
		}

		store := NewInMemoryCredentialStore()
		store.SetToken("github", "ghp_store")

		token, err := ResolveProviderToken("github", state, store)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghp_env" {
			t.Errorf("expected env token to take priority, got %s", token)
		}
	})

	t.Run("not found returns empty string", func(t *testing.T) {
		token, err := ResolveProviderToken("github", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "" {
			t.Errorf("expected empty string, got %s", token)
		}
	})

	t.Run("credential store error", func(t *testing.T) {
		store := &failingCredentialStore{getErr: errors.New("store error")}

		_, err := ResolveProviderToken("github", nil, store)
		if err == nil {
			t.Fatal("expected error from failing store")
		}
	})

	t.Run("whitespace trimming", func(t *testing.T) {
		state := &GUIState{
			Credentials: &CredentialSnapshot{
				GitHubToken: "  ghp_state  ",
			},
		}

		token, err := ResolveProviderToken("github", state, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "ghp_state" {
			t.Errorf("expected trimmed token, got '%s'", token)
		}
	})
}

func TestRedactToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"short token", "abc", "***"},
		{"4 char token", "abcd", "***"},
		{"normal token", "ghp_1234567890", "ghp_***"},
		{"long token", "glpat_abcdefghijklmnop", "glpa***"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactToken(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestNewInMemoryCredentialStore(t *testing.T) {
	store := NewInMemoryCredentialStore()
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.tokens == nil {
		t.Error("expected tokens map to be initialized")
	}
}

func TestNewFallbackCredentialStore_NilFallback(t *testing.T) {
	primary := NewInMemoryCredentialStore()
	store := NewFallbackCredentialStore(primary, nil)
	if store == nil {
		t.Fatal("expected non-nil store")
	}
	if store.fallback == nil {
		t.Error("expected fallback to be created when nil")
	}
}

// failingCredentialStore is a test helper that simulates failures
type failingCredentialStore struct {
	getErr error
}

func (f *failingCredentialStore) SetToken(provider, token string) error {
	return errors.New("set failed")
}

func (f *failingCredentialStore) GetToken(provider string) (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	return "", errors.New("get failed")
}

func (f *failingCredentialStore) DeleteToken(provider string) error {
	return errors.New("delete failed")
}

func (f *failingCredentialStore) ListProviders() ([]string, error) {
	return nil, errors.New("list failed")
}
