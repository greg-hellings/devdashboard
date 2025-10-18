package dependencies

import (
	"context"
	"errors"
	"testing"

	"github.com/greg-hellings/devdashboard/pkg/repository"
)

func TestUvLockAnalyzer_Name(t *testing.T) {
	analyzer := NewUvLockAnalyzer()
	if analyzer.Name() != string(AnalyzerUvLock) {
		t.Errorf("Expected name %s, got %s", AnalyzerUvLock, analyzer.Name())
	}
}

func TestUvLockAnalyzer_CandidateFiles(t *testing.T) {
	tests := []struct {
		name        string
		mockFiles   []repository.FileInfo
		mockError   error
		searchPaths []string
		want        []DependencyFile
		wantErr     bool
	}{
		{
			name: "finds uv.lock in root",
			mockFiles: []repository.FileInfo{
				{Path: "uv.lock", Type: "file"},
				{Path: "README.md", Type: "file"},
			},
			searchPaths: []string{""},
			want: []DependencyFile{
				{Path: "uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
			},
			wantErr: false,
		},
		{
			name: "finds multiple uv.lock files",
			mockFiles: []repository.FileInfo{
				{Path: "uv.lock", Type: "file"},
				{Path: "api/uv.lock", Type: "file"},
				{Path: "workers/uv.lock", Type: "file"},
			},
			searchPaths: []string{""},
			want: []DependencyFile{
				{Path: "uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
				{Path: "api/uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
				{Path: "workers/uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
			},
			wantErr: false,
		},
		{
			name: "filters by search path",
			mockFiles: []repository.FileInfo{
				{Path: "uv.lock", Type: "file"},
				{Path: "api/uv.lock", Type: "file"},
				{Path: "workers/uv.lock", Type: "file"},
			},
			searchPaths: []string{"api"},
			want: []DependencyFile{
				{Path: "api/uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
			},
			wantErr: false,
		},
		{
			name: "ignores directories",
			mockFiles: []repository.FileInfo{
				{Path: "uv.lock", Type: "dir"},
				{Path: "real/uv.lock", Type: "file"},
			},
			searchPaths: []string{""},
			want: []DependencyFile{
				{Path: "real/uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
			},
			wantErr: false,
		},
		{
			name:        "handles repository error",
			mockError:   errors.New("repository error"),
			searchPaths: []string{""},
			want:        nil,
			wantErr:     true,
		},
		{
			name:        "returns error when client is nil",
			mockFiles:   []repository.FileInfo{},
			searchPaths: []string{""},
			want:        nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewUvLockAnalyzer()

			var config Config
			if tt.name != "returns error when client is nil" {
				config = Config{
					RepositoryPaths:  tt.searchPaths,
					RepositoryClient: &mockRepoClient{files: tt.mockFiles, err: tt.mockError},
				}
			} else {
				config = Config{
					RepositoryPaths:  tt.searchPaths,
					RepositoryClient: nil,
				}
			}

			got, err := analyzer.CandidateFiles(context.Background(), "owner", "repo", "main", config)

			if (err != nil) != tt.wantErr {
				t.Errorf("CandidateFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("CandidateFiles() got %d files, want %d", len(got), len(tt.want))
					return
				}

				for i := range got {
					if got[i].Path != tt.want[i].Path ||
						got[i].Type != tt.want[i].Type ||
						got[i].Analyzer != tt.want[i].Analyzer {
						t.Errorf("CandidateFiles()[%d] = %+v, want %+v", i, got[i], tt.want[i])
					}
				}
			}
		})
	}
}

func TestUvLockAnalyzer_AnalyzeDependencies(t *testing.T) {
	tests := []struct {
		name           string
		files          []DependencyFile
		mockContent    string
		mockError      error
		wantNumFiles   int
		wantNumDeps    int
		wantErr        bool
		checkFirstDep  bool
		expectedDepKey string
		expectedDep    Dependency
	}{
		{
			name: "parses basic uv.lock",
			files: []DependencyFile{
				{Path: "uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
			},
			mockContent: `version = 1
requires-python = ">=3.8"

[[package]]
name = "requests"
version = "2.28.1"

[package.source]
type = "registry"
registry = "pypi"

[[package]]
name = "urllib3"
version = "1.26.12"

[package.source]
type = "registry"
registry = "pypi"
`,
			wantNumFiles:   1,
			wantNumDeps:    2,
			checkFirstDep:  true,
			expectedDepKey: "uv.lock",
			expectedDep: Dependency{
				Name:    "requests",
				Version: "2.28.1",
				Type:    "runtime",
				Source:  "pypi",
			},
		},
		{
			name: "handles dev dependencies with markers",
			files: []DependencyFile{
				{Path: "uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
			},
			mockContent: `version = 1
requires-python = ">=3.8"

[[package]]
name = "pytest"
version = "7.2.0"
marker = "extra == 'dev'"

[package.source]
type = "registry"
`,
			wantNumFiles:   1,
			wantNumDeps:    1,
			checkFirstDep:  true,
			expectedDepKey: "uv.lock",
			expectedDep: Dependency{
				Name:    "pytest",
				Version: "7.2.0",
				Type:    "dev",
				Source:  "pypi",
			},
		},
		{
			name: "handles git sources",
			files: []DependencyFile{
				{Path: "uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
			},
			mockContent: `version = 1
requires-python = ">=3.8"

[[package]]
name = "mypackage"
version = "0.1.0"

[package.source]
type = "git"
git = "https://github.com/user/repo.git"
rev = "abc123"
`,
			wantNumFiles:   1,
			wantNumDeps:    1,
			checkFirstDep:  true,
			expectedDepKey: "uv.lock",
			expectedDep: Dependency{
				Name:    "mypackage",
				Version: "0.1.0",
				Type:    "runtime",
				Source:  "git",
			},
		},
		{
			name: "handles path sources",
			files: []DependencyFile{
				{Path: "uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
			},
			mockContent: `version = 1
requires-python = ">=3.8"

[[package]]
name = "local-package"
version = "1.0.0"

[package.source]
type = "path"
path = "./packages/local"
`,
			wantNumFiles:   1,
			wantNumDeps:    1,
			checkFirstDep:  true,
			expectedDepKey: "uv.lock",
			expectedDep: Dependency{
				Name:    "local-package",
				Version: "1.0.0",
				Type:    "runtime",
				Source:  "path",
			},
		},
		{
			name: "handles empty dependencies",
			files: []DependencyFile{
				{Path: "uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
			},
			mockContent: `version = 1
requires-python = ">=3.8"
`,
			wantNumFiles: 1,
			wantNumDeps:  0,
		},
		{
			name: "handles multiple files",
			files: []DependencyFile{
				{Path: "uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
				{Path: "api/uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
			},
			mockContent: `version = 1
requires-python = ">=3.8"

[[package]]
name = "requests"
version = "2.28.1"

[package.source]
type = "registry"
`,
			wantNumFiles: 2,
			wantNumDeps:  1,
		},
		{
			name: "skips invalid files",
			files: []DependencyFile{
				{Path: "uv.lock", Type: "uv.lock", Analyzer: "uvlock"},
			},
			mockContent:  `invalid toml content {{{`,
			wantNumFiles: 0,
			wantNumDeps:  0,
		},
		{
			name:         "returns error when client is nil",
			files:        []DependencyFile{{Path: "uv.lock", Type: "uv.lock"}},
			wantNumFiles: 0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewUvLockAnalyzer()

			var config Config
			if tt.name != "returns error when client is nil" {
				config = Config{
					RepositoryClient: &mockRepoClient{content: tt.mockContent, err: tt.mockError},
				}
			} else {
				config = Config{
					RepositoryClient: nil,
				}
			}

			got, err := analyzer.AnalyzeDependencies(context.Background(), "owner", "repo", "main", tt.files, config)

			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeDependencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(got) != tt.wantNumFiles {
					t.Errorf("AnalyzeDependencies() got %d files, want %d", len(got), tt.wantNumFiles)
					return
				}

				if tt.wantNumFiles > 0 {
					for _, deps := range got {
						if len(deps) != tt.wantNumDeps {
							t.Errorf("AnalyzeDependencies() got %d deps, want %d", len(deps), tt.wantNumDeps)
						}
					}

					if tt.checkFirstDep && tt.wantNumDeps > 0 {
						deps := got[tt.expectedDepKey]
						if len(deps) == 0 {
							t.Errorf("No dependencies found for key %s", tt.expectedDepKey)
							return
						}

						// Find the expected dependency by name
						found := false
						for _, dep := range deps {
							if dep.Name == tt.expectedDep.Name {
								found = true
								if dep.Version != tt.expectedDep.Version ||
									dep.Type != tt.expectedDep.Type ||
									dep.Source != tt.expectedDep.Source {
									t.Errorf("Dependency mismatch: got %+v, want %+v", dep, tt.expectedDep)
								}
								break
							}
						}
						if !found {
							t.Errorf("Expected dependency %s not found", tt.expectedDep.Name)
						}
					}
				}
			}
		})
	}
}

func TestUvLockAnalyzer_ParseUvLock(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantNumDeps int
		wantErr     bool
		checkDeps   []Dependency
	}{
		{
			name: "parses runtime and dev dependencies",
			content: `version = 1
requires-python = ">=3.8"

[[package]]
name = "django"
version = "4.1.0"

[package.source]
type = "registry"

[[package]]
name = "requests"
version = "2.28.1"

[package.source]
type = "registry"

[[package]]
name = "pytest"
version = "7.2.0"
marker = "extra == 'dev'"

[package.source]
type = "registry"

[[package]]
name = "black"
version = "22.10.0"
resolution-markers = "extra == 'test'"

[package.source]
type = "registry"
`,
			wantNumDeps: 4,
			checkDeps: []Dependency{
				{Name: "django", Version: "4.1.0", Type: "runtime", Source: "pypi"},
				{Name: "requests", Version: "2.28.1", Type: "runtime", Source: "pypi"},
				{Name: "pytest", Version: "7.2.0", Type: "dev", Source: "pypi"},
				{Name: "black", Version: "22.10.0", Type: "dev", Source: "pypi"},
			},
		},
		{
			name: "handles different source types",
			content: `version = 1
requires-python = ">=3.8"

[[package]]
name = "pypi-pkg"
version = "1.0.0"

[package.source]
type = "registry"

[[package]]
name = "git-pkg"
version = "2.0.0"

[package.source]
type = "git"
git = "https://github.com/user/repo.git"

[[package]]
name = "path-pkg"
version = "3.0.0"

[package.source]
type = "path"
path = "./local"

[[package]]
name = "url-pkg"
version = "4.0.0"

[package.source]
type = "url"
url = "https://example.com/pkg.whl"

[[package]]
name = "directory-pkg"
version = "5.0.0"

[package.source]
type = "directory"
path = "./mydir"
`,
			wantNumDeps: 5,
			checkDeps: []Dependency{
				{Name: "pypi-pkg", Version: "1.0.0", Type: "runtime", Source: "pypi"},
				{Name: "git-pkg", Version: "2.0.0", Type: "runtime", Source: "git"},
				{Name: "path-pkg", Version: "3.0.0", Type: "runtime", Source: "path"},
				{Name: "url-pkg", Version: "4.0.0", Type: "runtime", Source: "url"},
				{Name: "directory-pkg", Version: "5.0.0", Type: "runtime", Source: "path"},
			},
		},
		{
			name: "handles packages without explicit source",
			content: `version = 1
requires-python = ">=3.8"

[[package]]
name = "package"
version = "1.0.0"

[package.source]
type = "registry"
`,
			wantNumDeps: 1,
			checkDeps: []Dependency{
				{Name: "package", Version: "1.0.0", Type: "runtime", Source: "pypi"},
			},
		},
		{
			name: "handles packages with dev-dependencies field",
			content: `version = 1
requires-python = ">=3.8"

[[package]]
name = "test-pkg"
version = "1.0.0"

[package.source]
type = "registry"

[[package.dev-dependencies]]
name = "pytest"
version = ">=7.0"
`,
			wantNumDeps: 1,
			checkDeps: []Dependency{
				{Name: "test-pkg", Version: "1.0.0", Type: "dev", Source: "pypi"},
			},
		},
		{
			name:        "handles invalid TOML",
			content:     `invalid {{{ toml`,
			wantNumDeps: 0,
			wantErr:     true,
		},
		{
			name: "handles empty package list",
			content: `version = 1
requires-python = ">=3.8"
`,
			wantNumDeps: 0,
			wantErr:     false,
		},
		{
			name: "handles complex package with wheels and sdist",
			content: `version = 1
requires-python = ">=3.8"

[[package]]
name = "complex-pkg"
version = "2.5.0"

[package.source]
type = "registry"
registry = "pypi"

[[package.wheels]]
url = "https://files.pythonhosted.org/packages/..."
hash = "sha256:abc123"
size = 1234567

[package.sdist]
url = "https://files.pythonhosted.org/packages/..."
hash = "sha256:def456"
size = 234567
`,
			wantNumDeps: 1,
			checkDeps: []Dependency{
				{Name: "complex-pkg", Version: "2.5.0", Type: "runtime", Source: "pypi"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewUvLockAnalyzer()
			deps, err := analyzer.parseUvLock(tt.content)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseUvLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(deps) != tt.wantNumDeps {
					t.Errorf("parseUvLock() got %d deps, want %d", len(deps), tt.wantNumDeps)
					return
				}

				if len(tt.checkDeps) > 0 {
					for _, expected := range tt.checkDeps {
						found := false
						for _, got := range deps {
							if got.Name == expected.Name {
								found = true
								if got.Version != expected.Version ||
									got.Type != expected.Type ||
									got.Source != expected.Source {
									t.Errorf("Dependency %s: got %+v, want %+v", expected.Name, got, expected)
								}
								break
							}
						}
						if !found {
							t.Errorf("Expected dependency %s not found", expected.Name)
						}
					}
				}
			}
		})
	}
}
