// Package state provides consolidated application state management for DevDashboard.
// This includes credential storage, GUI state persistence, and configuration management.
package state

// credentials.go
//
// Credential storage abstraction for DevDashboard.
//
// Goals:
//   * Decouple token persistence from GUI / CLI logic
//   * Allow secure implementations (OS keyring, secrets manager) without changing callers
//   * Provide an in-memory + YAML fallback for development & testing
//
// Design Notes:
//   * Provider identifiers (e.g. "github", "gitlab") are used as keys
//   * Future extension: per-provider multiple credentials (scoped tokens) -> expand key scheme
//   * All interface methods are context-free for simplicity; add context if remote latency appears
//
// Security Guidance:
//   * Avoid logging raw tokens
//   * Use RedactToken (already provided in other code) or similar logic before emitting values
//
// This file intentionally does NOT introduce external dependencies (e.g., keyring libraries).
// A secure implementation can live behind build tags in separate files (e.g. credentials_keyring.go).

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// CredentialStore defines the contract for secure token persistence.
type CredentialStore interface {
	// SetToken stores or updates a token for a given provider.
	SetToken(provider string, token string) error
	// GetToken retrieves a token. Returns ErrCredentialNotFound if missing.
	GetToken(provider string) (string, error)
	// DeleteToken removes a stored token (idempotent).
	DeleteToken(provider string) error
	// ListProviders returns provider IDs that have tokens stored.
	ListProviders() ([]string, error)
}

// ErrCredentialNotFound is returned when a token for a provider does not exist.
var ErrCredentialNotFound = errors.New("credential not found")

// InMemoryCredentialStore is a thread-safe, volatile implementation.
// Useful for tests, ephemeral sessions, or as a fallback when a secure store
// is unavailable.
type InMemoryCredentialStore struct {
	mu     sync.RWMutex
	tokens map[string]string
}

// NewInMemoryCredentialStore creates an empty store.
func NewInMemoryCredentialStore() *InMemoryCredentialStore {
	return &InMemoryCredentialStore{
		tokens: make(map[string]string),
	}
}

// SetToken stores or updates the token for the given provider in the in-memory map.
func (s *InMemoryCredentialStore) SetToken(provider string, token string) error {
	if provider == "" {
		return errors.New("provider cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[provider] = token
	return nil
}

// GetToken returns the token for provider or ErrCredentialNotFound if none is stored.
func (s *InMemoryCredentialStore) GetToken(provider string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.tokens[provider]
	if !ok {
		return "", ErrCredentialNotFound
	}
	return v, nil
}

// DeleteToken removes the token for provider; missing providers are ignored.
func (s *InMemoryCredentialStore) DeleteToken(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, provider)
	return nil
}

// ListProviders returns all provider IDs that currently have tokens.
func (s *InMemoryCredentialStore) ListProviders() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.tokens))
	for k := range s.tokens {
		out = append(out, k)
	}
	return out, nil
}

// FallbackCredentialStore composes two stores: a primary (secure) and a fallback (e.g. in-memory).
// Reads prefer primary; writes attempt primary then fallback if primary fails.
type FallbackCredentialStore struct {
	primary  CredentialStore
	fallback CredentialStore
}

// NewFallbackCredentialStore creates a layered store.
// If primary is nil, fallback is used for all operations.
func NewFallbackCredentialStore(primary, fallback CredentialStore) *FallbackCredentialStore {
	if fallback == nil {
		fallback = NewInMemoryCredentialStore()
	}
	return &FallbackCredentialStore{primary: primary, fallback: fallback}
}

// SetToken writes the token, preferring the primary store when available; falls back if primary fails.
func (f *FallbackCredentialStore) SetToken(provider, token string) error {
	if f.primary != nil {
		if err := f.primary.SetToken(provider, token); err == nil {
			return nil
		}
	}
	return f.fallback.SetToken(provider, token)
}

// GetToken retrieves a token preferring the primary store; non-not-found primary errors are wrapped and returned.
func (f *FallbackCredentialStore) GetToken(provider string) (string, error) {
	if f.primary != nil {
		if v, err := f.primary.GetToken(provider); err == nil {
			return v, nil
		} else if !errors.Is(err, ErrCredentialNotFound) {
			// Non-not-found error: return it early
			return "", fmt.Errorf("primary get token: %w", err)
		}
	}
	return f.fallback.GetToken(provider)
}

// DeleteToken attempts removal in both primary and fallback stores, ignoring not-found conditions.
func (f *FallbackCredentialStore) DeleteToken(provider string) error {
	var primaryErr error
	if f.primary != nil {
		primaryErr = f.primary.DeleteToken(provider)
	}
	fallbackErr := f.fallback.DeleteToken(provider)
	// Prefer a non-not-found error from either layer.
	if primaryErr != nil && !errors.Is(primaryErr, ErrCredentialNotFound) {
		return primaryErr
	}
	if fallbackErr != nil && !errors.Is(fallbackErr, ErrCredentialNotFound) {
		return fallbackErr
	}
	return nil
}

// ListProviders merges provider IDs discovered in both primary and fallback stores, de-duplicated.
func (f *FallbackCredentialStore) ListProviders() ([]string, error) {
	seen := map[string]struct{}{}
	var out []string

	addAll := func(list []string, err error) error {
		if err != nil {
			return err
		}
		for _, p := range list {
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				out = append(out, p)
			}
		}
		return nil
	}

	if f.primary != nil {
		if err := addAll(f.primary.ListProviders()); err != nil {
			return nil, fmt.Errorf("primary list providers: %w", err)
		}
	}
	if err := addAll(f.fallback.ListProviders()); err != nil {
		return nil, fmt.Errorf("fallback list providers: %w", err)
	}

	return out, nil
}

// StubCredentialStore is a no-op implementation returning not-found for all operations.
// Can be used when credential operations are intentionally disabled.
type StubCredentialStore struct{}

// SetToken is a no-op for StubCredentialStore.
func (StubCredentialStore) SetToken(_, _ string) error { return nil }

// GetToken always returns ErrCredentialNotFound for StubCredentialStore.
func (StubCredentialStore) GetToken(_ string) (string, error) {
	return "", ErrCredentialNotFound
}

// DeleteToken is a no-op for StubCredentialStore.
func (StubCredentialStore) DeleteToken(_ string) error { return nil }

// ListProviders returns an empty slice for StubCredentialStore.
func (StubCredentialStore) ListProviders() ([]string, error) { return []string{}, nil }

// (Future) Keyring / secure implementations will live under build tags, e.g.:
//
//	//go:build keyring
//	// +build keyring
//
//	package state
//
//	type KeyringCredentialStore struct { ... }

// ResolveProviderToken returns the credential for the given provider.
// Lookup order:
//  1. Environment variable DEV_DASHBOARD_<PROVIDER>_TOKEN
//  2. GUIState.Credentials snapshot (prototype / YAML storage)
//  3. CredentialStore (if provided)
//
// It returns an empty string if none is found. Always redact tokens before logging.
func ResolveProviderToken(provider string, st *GUIState, cs CredentialStore) (string, error) {
	if provider == "" {
		return "", errors.New("provider cannot be empty")
	}

	envName := fmt.Sprintf("DEV_DASHBOARD_%s_TOKEN", strings.ToUpper(provider))
	if v := strings.TrimSpace(os.Getenv(envName)); v != "" {
		return v, nil
	}

	// YAML / state snapshot (prototype only)
	if st != nil && st.Credentials != nil {
		switch provider {
		case "github":
			if tok := strings.TrimSpace(st.Credentials.GitHubToken); tok != "" {
				return tok, nil
			}
		case "gitlab":
			if tok := strings.TrimSpace(st.Credentials.GitLabToken); tok != "" {
				return tok, nil
			}
		}
	}

	// Credential store (could be secure / keyring-backed)
	if cs != nil {
		if tok, err := cs.GetToken(provider); err == nil && strings.TrimSpace(tok) != "" {
			return tok, nil
		} else if err != nil && !errors.Is(err, ErrCredentialNotFound) {
			return "", fmt.Errorf("credential store failure: %w", err)
		}
	}

	return "", nil
}

// RedactToken safely redacts a token for logging purposes.
func RedactToken(tok string) string {
	if tok == "" {
		return ""
	}
	if len(tok) <= 4 {
		return "***"
	}
	return tok[:4] + "***"
}
