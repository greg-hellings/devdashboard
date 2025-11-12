package state

// Package state provides consolidated application state management for the
// DevDashboard ecosystem (CLI + GUI front-ends).
//
// This file contains the GUI-oriented state model and persistence helpers that
// were originally implemented inside the desktop GUI module. Moving it into
// core allows all future front-ends (desktop, web, mobile) to share a single
// canonical definition and logic for:
//   * YAML-backed state persistence
//   * Config merge / denormalized repository cache
//   * Redaction for logging / diagnostics
//   * Future migration handling & credential abstraction
//
// (1) Scope
// --------
// The structures here intentionally mirror the CLI configuration format where
// practical (providers → defaults + repositories) while adding a `GUI` section
// for interactive preferences and runtime metadata.
//
// (2) Versioning
// --------------
// StateVersion enables forward migration. Increment only when a breaking
// structural change is introduced and add a migration function to adjust
// persisted files.
//
// (3) Security
// ------------
// Access tokens should NOT be stored long-term in plain YAML once secure
// keyring integration is added. The CredentialSnapshot is a temporary bridge.
//
// (4) Extensibility
// -----------------
// Additional per-frontend settings should be nested underneath a namespaced
// struct (e.g., GUI) to avoid polluting the top-level.
//
// (5) Thread Safety
// -----------------
// The state objects themselves are *not* synchronized. Callers are responsible
// for guarding concurrent access (e.g., via a runtime-level mutex in GUI).
//
// (6) Future Work
// ---------------
// - Keyring-backed CredentialStore
// - Migration registry
// - History/diff persistence with pruning
// - Validation & schema auditing
//
// Usage Example (GUI):
//   st, _ := state.LoadGUIState("")
//   st.GUI.Theme = "dark"
//   _ = state.SaveGUIState(st, "")
//
// Usage Example (Export CLI-compatible YAML):
//   // Only `Providers` would typically be exported; additional trimming helper
//   // can be added later.
//
// NOTE: All exported symbols are prefixed conservatively to avoid collisions
// with other future state domains (e.g., pipeline, metrics).
//
import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/greg-hellings/devdashboard/core/pkg/config"
)

// GUIState represents the full persisted GUI application state (YAML).
type GUIState struct {
	StateVersion      int                              `yaml:"stateVersion"`
	SavedAt           time.Time                        `yaml:"savedAt"`
	Profile           string                           `yaml:"profile"`
	GUI               GUISection                       `yaml:"gui"`
	Providers         map[string]ProviderConfigWrapper `yaml:"providers"`
	RepositoriesCache []RepoCacheEntry                 `yaml:"repositoriesCache"`
	TrackedPackages   []string                         `yaml:"trackedPackages"`
	Credentials       *CredentialSnapshot              `yaml:"credentials,omitempty"`
	ErrorLog          []ErrorLogEntry                  `yaml:"errorLog,omitempty"`
	ReportHistory     []ReportHistoryEntry             `yaml:"reportHistory,omitempty"`
	Extensions        map[string]map[string]any        `yaml:"extensions,omitempty"` // reserved for future pluggable modules
	Meta              map[string]string                `yaml:"meta,omitempty"`       // arbitrary small string map
}

// GUISection contains desktop/UI specific preferences and metadata.
type GUISection struct {
	LastWindow   WindowGeometry  `yaml:"lastWindow"`
	Theme        string          `yaml:"theme"` // light | dark
	RecentConfig []string        `yaml:"recentConfigFiles"`
	Concurrency  ConcurrencyCfg  `yaml:"concurrency"`
	AutoRefresh  AutoRefreshCfg  `yaml:"autoRefresh"`
	Logging      LoggingCfg      `yaml:"logging"`
	LastReport   *LastReportMeta `yaml:"lastReport,omitempty"`
}

// WindowGeometry tracks last window geometry.
type WindowGeometry struct {
	Width     int  `yaml:"width"`
	Height    int  `yaml:"height"`
	Maximized bool `yaml:"maximized"`
}

// ConcurrencyCfg tuning for async tasks.
type ConcurrencyCfg struct {
	MaxWorkers int `yaml:"maxWorkers"`
}

// AutoRefreshCfg controls periodic dependency report refresh.
type AutoRefreshCfg struct {
	Enabled         bool `yaml:"enabled"`
	IntervalSeconds int  `yaml:"intervalSeconds"`
}

// LoggingCfg controls in-memory logging capture.
type LoggingCfg struct {
	RingBufferSize int    `yaml:"ringBufferSize"`
	Level          string `yaml:"level"` // info | debug | warn | error
}

// LastReportMeta summarises the most recent dependency report.
type LastReportMeta struct {
	GeneratedAt  time.Time `yaml:"generatedAt"`
	RepoCount    int       `yaml:"repoCount"`
	PackageCount int       `yaml:"packageCount"`
}

// ProviderConfigWrapper mirrors CLI provider structure.
type ProviderConfigWrapper struct {
	Default      config.RepoDefaults `yaml:"default"`
	Repositories []config.RepoConfig `yaml:"repositories"`
}

// RepoCacheEntry is a denormalized cache row for fast GUI listing.
type RepoCacheEntry struct {
	Provider   string   `yaml:"provider"`
	Token      string   `yaml:"token"`
	Owner      string   `yaml:"owner"`
	Repository string   `yaml:"repository"`
	Ref        string   `yaml:"ref"`
	Paths      []string `yaml:"paths"`
	Packages   []string `yaml:"packages"`
	Analyzer   string   `yaml:"analyzer"`
}

// CredentialSnapshot is prototype-only. Replace with keyring / secure store.
type CredentialSnapshot struct {
	GitHubToken string `yaml:"githubToken,omitempty"`
	GitLabToken string `yaml:"gitlabToken,omitempty"`
}

// ErrorLogEntry allows structured recent error display.
type ErrorLogEntry struct {
	Time     time.Time `yaml:"time"`
	Source   string    `yaml:"source"`
	Severity string    `yaml:"severity"`
	Message  string    `yaml:"message"`
	Details  string    `yaml:"details,omitempty"`
}

// ReportHistoryEntry supports future diff & history features.
type ReportHistoryEntry struct {
	GeneratedAt  time.Time `yaml:"generatedAt"`
	RepoCount    int       `yaml:"repoCount"`
	PackageCount int       `yaml:"packageCount"`
	SummaryPath  string    `yaml:"summaryPath,omitempty"`
}

// GUIStateStore defines pluggable storage behaviour (filesystem, memory, remote).
type GUIStateStore interface {
	Load(path string) (*GUIState, error)
	Save(st *GUIState, path string) error
}

// FilesystemGUIStateStore is the default disk-backed implementation.
type FilesystemGUIStateStore struct{}

// Load implements GUIStateStore.Load.
func (FilesystemGUIStateStore) Load(path string) (*GUIState, error) {
	return LoadGUIState(path)
}

// Save implements GUIStateStore.Save.
func (FilesystemGUIStateStore) Save(st *GUIState, path string) error {
	return SaveGUIState(st, path)
}

// NewDefaultGUIState creates a new initialized GUIState with sane defaults.
func NewDefaultGUIState() *GUIState {
	return &GUIState{
		StateVersion: 1,
		SavedAt:      time.Now().UTC(),
		Profile:      "default",
		GUI: GUISection{
			LastWindow:   WindowGeometry{Width: 1100, Height: 700, Maximized: false},
			Theme:        "light",
			RecentConfig: []string{},
			Concurrency:  ConcurrencyCfg{MaxWorkers: runtime.NumCPU()},
			AutoRefresh:  AutoRefreshCfg{Enabled: false, IntervalSeconds: 900},
			Logging:      LoggingCfg{RingBufferSize: 5000, Level: "info"},
		},
		Providers: map[string]ProviderConfigWrapper{
			"github": {
				Default:      config.RepoDefaults{Ref: "main", Analyzer: "poetry"},
				Repositories: []config.RepoConfig{},
			},
			"gitlab": {
				Default:      config.RepoDefaults{Ref: "main", Analyzer: "poetry"},
				Repositories: []config.RepoConfig{},
			},
		},
		RepositoriesCache: []RepoCacheEntry{},
		TrackedPackages:   []string{},
		ErrorLog:          []ErrorLogEntry{},
		ReportHistory:     []ReportHistoryEntry{},
		Extensions:        map[string]map[string]any{},
		Meta:              map[string]string{},
	}
}

// LoadGUIState loads a GUIState from disk, returning defaults if file missing.
func LoadGUIState(path string) (*GUIState, error) {
	if path == "" {
		path = DefaultGUIStatePath()
	}
	if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(userConfigDir())+string(os.PathSeparator)) {
		return nil, fmt.Errorf("state: path outside config dir: %s", path)
	}
	// #nosec G304 validated path confined to user config directory above
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewDefaultGUIState(), nil
		}
		return nil, fmt.Errorf("state: read failed: %w", err)
	}
	var st GUIState
	if err := yaml.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("state: parse failed: %w", err)
	}
	normalizeGUIState(&st)
	return &st, nil
}

// SaveGUIState persists the state atomically to disk.
func SaveGUIState(st *GUIState, path string) error {
	if st == nil {
		return errors.New("state: nil GUIState")
	}
	if path == "" {
		path = DefaultGUIStatePath()
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("state: mkdir failed: %w", err)
	}
	st.SavedAt = time.Now().UTC()

	out, err := yaml.Marshal(st)
	if err != nil {
		return fmt.Errorf("state: marshal failed: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".gui_state.tmp-*")
	if err != nil {
		return fmt.Errorf("state: temp create failed: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(out); err != nil {
		return fmt.Errorf("state: temp write failed: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		return fmt.Errorf("state: chmod failed: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("state: sync failed: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("state: atomic rename failed: %w", err)
	}
	return nil
}

// DefaultGUIStatePath returns the OS-specific default path for GUI state.
func DefaultGUIStatePath() string {
	base := userConfigDir()
	return filepath.Join(base, "devdashboard", "gui_state.yaml")
}

// userConfigDir attempts to resolve a configuration directory in a portable way.
func userConfigDir() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".config")
	}
	return "."
}

// normalizeGUIState ensures invariants and fills defaults after load.
func normalizeGUIState(st *GUIState) {
	if st.StateVersion <= 0 {
		st.StateVersion = 1
	}
	if st.Profile == "" {
		st.Profile = "default"
	}
	if st.GUI.Concurrency.MaxWorkers <= 0 {
		st.GUI.Concurrency.MaxWorkers = runtime.NumCPU()
	}
	if st.GUI.Logging.RingBufferSize <= 0 {
		st.GUI.Logging.RingBufferSize = 5000
	}
	if st.GUI.Theme == "" {
		st.GUI.Theme = "light"
	}
	if st.Providers == nil {
		st.Providers = map[string]ProviderConfigWrapper{}
	}
	if st.Extensions == nil {
		st.Extensions = map[string]map[string]any{}
	}
	if st.Meta == nil {
		st.Meta = map[string]string{}
	}
	// Do not automatically rebuild cache here; caller can decide after merges.
}

// AppendRecentConfig adds a file path to MRU list (de-duped, size-limited).
func (s *GUIState) AppendRecentConfig(p string, maxItems int) {
	if p == "" {
		return
	}
	filtered := make([]string, 0, len(s.GUI.RecentConfig)+1)
	for _, existing := range s.GUI.RecentConfig {
		if existing != p {
			filtered = append(filtered, existing)
		}
	}
	s.GUI.RecentConfig = append([]string{p}, filtered...)
	if maxItems > 0 && len(s.GUI.RecentConfig) > maxItems {
		s.GUI.RecentConfig = s.GUI.RecentConfig[:maxItems]
	}
}

// RedactedCopy returns a shallow-copied state with tokens anonymized.
func (s *GUIState) RedactedCopy() *GUIState {
	cp := *s
	if cp.Credentials != nil {
		cp.Credentials = &CredentialSnapshot{
			GitHubToken: redactToken(cp.Credentials.GitHubToken),
			GitLabToken: redactToken(cp.Credentials.GitLabToken),
		}
	}
	for k, prov := range cp.Providers {
		for i := range prov.Repositories {
			prov.Repositories[i].Token = redactToken(prov.Repositories[i].Token)
		}
		prov.Default.Token = redactToken(prov.Default.Token)
		cp.Providers[k] = prov
	}
	for i := range cp.RepositoriesCache {
		cp.RepositoriesCache[i].Token = redactToken(cp.RepositoriesCache[i].Token)
	}
	return &cp
}

func redactToken(t string) string {
	if t == "" {
		return ""
	}
	if len(t) <= 4 {
		return "***"
	}
	return t[:4] + "***"
}

// MergeCLIConfig merges repositories & defaults from a CLI YAML config file.
// Duplicate (provider+owner+repo+ref) entries are skipped.
func (s *GUIState) MergeCLIConfig(path string) error {
	cfg, err := config.LoadFromFile(path)
	if err != nil {
		return err
	}
	if s.Providers == nil {
		s.Providers = map[string]ProviderConfigWrapper{}
	}
	for pname, pc := range cfg.Providers {
		wrapper, ok := s.Providers[pname]
		if !ok {
			wrapper = ProviderConfigWrapper{Default: pc.Default}
		}
		existing := map[string]struct{}{}
		for _, r := range wrapper.Repositories {
			existing[repoCacheKey(pname, r.Owner, r.Repository, r.Ref)] = struct{}{}
		}
		for _, r := range pc.Repositories {
			key := repoCacheKey(pname, r.Owner, r.Repository, r.Ref)
			if _, dup := existing[key]; dup {
				continue
			}
			wrapper.Repositories = append(wrapper.Repositories, r)
		}
		s.Providers[pname] = wrapper
	}
	s.RebuildRepositoriesCache()
	s.AppendRecentConfig(path, 10)
	return nil
}

// RebuildRepositoriesCache regenerates the flattened repository cache.
func (s *GUIState) RebuildRepositoriesCache() {
	cache := make([]RepoCacheEntry, 0, 64)
	for pname, wrapper := range s.Providers {
		for _, r := range wrapper.Repositories {
			cache = append(cache, RepoCacheEntry{
				Provider:   pname,
				Token:      r.Token,
				Owner:      r.Owner,
				Repository: r.Repository,
				Ref:        r.Ref,
				Paths:      r.Paths,
				Packages:   r.Packages,
				Analyzer:   r.Analyzer,
			})
		}
	}
	s.RepositoriesCache = cache
}

func repoCacheKey(provider, owner, repo, ref string) string {
	return fmt.Sprintf("%s:%s/%s@%s", provider, owner, repo, ref)
}

// WriteTo writes the full YAML representation to an arbitrary writer.
func (s *GUIState) WriteTo(w io.Writer) (int64, error) {
	out, err := yaml.Marshal(s)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(out)
	return int64(n), err
}

// TrimForCLIExport returns a reduced copy containing only provider config.
// (Future enhancement: convert back to pure CLI YAML shape exactly.)
func (s *GUIState) TrimForCLIExport() *GUIState {
	cp := *s
	cp.GUI = GUISection{}
	cp.RepositoriesCache = nil
	cp.TrackedPackages = nil
	cp.Credentials = nil
	cp.ErrorLog = nil
	cp.ReportHistory = nil
	cp.Extensions = nil
	cp.Meta = nil
	return &cp
}

// PendingMigrations reports whether migrations are required (future use).
func (s *GUIState) PendingMigrations() bool {
	// Reserved for future version bump logic.
	return false
}

// ApplyMigrations mutates state in-place to latest version (future use).
func (s *GUIState) ApplyMigrations() error {
	// Reserved for future version transformations.
	return nil
}
