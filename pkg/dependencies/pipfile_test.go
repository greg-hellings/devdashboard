package dependencies

import (
	"context"
	"errors"
	"testing"

	"github.com/greg-hellings/devdashboard/pkg/repository"
)

func TestPipfileAnalyzer_Name(t *testing.T) {
	analyzer := NewPipfileAnalyzer()
	if analyzer.Name() != string(AnalyzerPipfile) {
		t.Errorf("Expected name %s, got %s", AnalyzerPipfile, analyzer.Name())
	}
}

func TestPipfileAnalyzer_CandidateFiles(t *testing.T) {
	tests := []struct {
		name        string
		mockFiles   []repository.FileInfo
		mockError   error
		searchPaths []string
		want        []DependencyFile
		wantErr     bool
	}{
		{
			name: "finds Pipfile.lock in root",
			mockFiles: []repository.FileInfo{
				{Path: "Pipfile.lock", Type: "file"},
				{Path: "README.md", Type: "file"},
			},
			searchPaths: []string{""},
			want: []DependencyFile{
				{Path: "Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
			},
			wantErr: false,
		},
		{
			name: "finds multiple Pipfile.lock files",
			mockFiles: []repository.FileInfo{
				{Path: "Pipfile.lock", Type: "file"},
				{Path: "api/Pipfile.lock", Type: "file"},
				{Path: "workers/Pipfile.lock", Type: "file"},
			},
			searchPaths: []string{""},
			want: []DependencyFile{
				{Path: "Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
				{Path: "api/Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
				{Path: "workers/Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
			},
			wantErr: false,
		},
		{
			name: "filters by search path",
			mockFiles: []repository.FileInfo{
				{Path: "Pipfile.lock", Type: "file"},
				{Path: "api/Pipfile.lock", Type: "file"},
				{Path: "workers/Pipfile.lock", Type: "file"},
			},
			searchPaths: []string{"api"},
			want: []DependencyFile{
				{Path: "api/Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
			},
			wantErr: false,
		},
		{
			name: "ignores directories",
			mockFiles: []repository.FileInfo{
				{Path: "Pipfile.lock", Type: "dir"},
				{Path: "real/Pipfile.lock", Type: "file"},
			},
			searchPaths: []string{""},
			want: []DependencyFile{
				{Path: "real/Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
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
		{
			name: "handles multiple search paths",
			mockFiles: []repository.FileInfo{
				{Path: "backend/Pipfile.lock", Type: "file"},
				{Path: "frontend/Pipfile.lock", Type: "file"},
				{Path: "services/api/Pipfile.lock", Type: "file"},
			},
			searchPaths: []string{"backend", "frontend"},
			want: []DependencyFile{
				{Path: "backend/Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
				{Path: "frontend/Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
			},
			wantErr: false,
		},
		{
			name: "no files when wrong extension",
			mockFiles: []repository.FileInfo{
				{Path: "Pipfile", Type: "file"},
				{Path: "requirements.txt", Type: "file"},
				{Path: "setup.py", Type: "file"},
			},
			searchPaths: []string{""},
			want:        []DependencyFile{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewPipfileAnalyzer()

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

func TestPipfileAnalyzer_AnalyzeDependencies(t *testing.T) {
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
			name: "parses basic Pipfile.lock",
			files: []DependencyFile{
				{Path: "Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
			},
			mockContent: `{
				"_meta": {
					"hash": {
						"sha256": "abc123"
					},
					"pipfile-spec": 6,
					"requires": {
						"python_version": "3.9"
					},
					"sources": []
				},
				"default": {
					"requests": {
						"version": "==2.28.1",
						"hashes": ["sha256:abc"]
					},
					"urllib3": {
						"version": "==1.26.12",
						"hashes": ["sha256:def"]
					}
				},
				"develop": {
					"pytest": {
						"version": "==7.2.0",
						"hashes": ["sha256:ghi"]
					}
				}
			}`,
			wantNumFiles:   1,
			wantNumDeps:    3,
			checkFirstDep:  true,
			expectedDepKey: "Pipfile.lock",
			expectedDep: Dependency{
				Name:    "requests",
				Version: "2.28.1",
				Type:    "runtime",
				Source:  "pypi",
			},
		},
		{
			name: "handles empty dependencies",
			files: []DependencyFile{
				{Path: "Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
			},
			mockContent: `{
				"_meta": {
					"hash": {
						"sha256": "abc123"
					},
					"pipfile-spec": 6,
					"requires": {},
					"sources": []
				},
				"default": {},
				"develop": {}
			}`,
			wantNumFiles: 1,
			wantNumDeps:  0,
		},
		{
			name: "handles multiple files",
			files: []DependencyFile{
				{Path: "Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
				{Path: "api/Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
			},
			mockContent: `{
				"_meta": {
					"hash": {
						"sha256": "abc123"
					},
					"pipfile-spec": 6,
					"requires": {},
					"sources": []
				},
				"default": {
					"requests": {
						"version": "==2.28.1",
						"hashes": ["sha256:abc"]
					}
				},
				"develop": {}
			}`,
			wantNumFiles: 2,
			wantNumDeps:  1,
		},
		{
			name: "skips invalid files",
			files: []DependencyFile{
				{Path: "Pipfile.lock", Type: "Pipfile.lock", Analyzer: "pipfile"},
			},
			mockContent:  `invalid json content {{{`,
			wantNumFiles: 0,
			wantNumDeps:  0,
		},
		{
			name:         "returns error when client is nil",
			files:        []DependencyFile{{Path: "Pipfile.lock", Type: "Pipfile.lock"}},
			wantNumFiles: 0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewPipfileAnalyzer()

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

func TestPipfileAnalyzer_ParsePipfileLock(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantNumDeps int
		wantErr     bool
		checkDeps   []Dependency
	}{
		{
			name: "parses runtime and dev dependencies",
			content: `{
				"_meta": {
					"hash": {
						"sha256": "abc123"
					},
					"pipfile-spec": 6,
					"requires": {
						"python_version": "3.9"
					},
					"sources": []
				},
				"default": {
					"django": {
						"version": "==4.1.0",
						"hashes": ["sha256:abc"]
					},
					"requests": {
						"version": "==2.28.1",
						"hashes": ["sha256:def"]
					}
				},
				"develop": {
					"pytest": {
						"version": "==7.2.0",
						"hashes": ["sha256:ghi"]
					},
					"black": {
						"version": "==22.10.0",
						"hashes": ["sha256:jkl"]
					}
				}
			}`,
			wantNumDeps: 4,
			checkDeps: []Dependency{
				{Name: "django", Version: "4.1.0", Type: "runtime", Source: "pypi"},
				{Name: "requests", Version: "2.28.1", Type: "runtime", Source: "pypi"},
				{Name: "pytest", Version: "7.2.0", Type: "dev", Source: "pypi"},
				{Name: "black", Version: "22.10.0", Type: "dev", Source: "pypi"},
			},
		},
		{
			name: "handles version with == prefix",
			content: `{
				"_meta": {
					"hash": {
						"sha256": "abc123"
					},
					"pipfile-spec": 6,
					"requires": {},
					"sources": []
				},
				"default": {
					"package": {
						"version": "==1.0.0",
						"hashes": []
					}
				},
				"develop": {}
			}`,
			wantNumDeps: 1,
			checkDeps: []Dependency{
				{Name: "package", Version: "1.0.0", Type: "runtime", Source: "pypi"},
			},
		},
		{
			name:        "handles invalid JSON",
			content:     `invalid {{{ json`,
			wantNumDeps: 0,
			wantErr:     true,
		},
		{
			name: "handles empty sections",
			content: `{
				"_meta": {
					"hash": {
						"sha256": "abc123"
					},
					"pipfile-spec": 6,
					"requires": {},
					"sources": []
				},
				"default": {},
				"develop": {}
			}`,
			wantNumDeps: 0,
			wantErr:     false,
		},
		{
			name: "handles packages with markers",
			content: `{
				"_meta": {
					"hash": {
						"sha256": "abc123"
					},
					"pipfile-spec": 6,
					"requires": {},
					"sources": []
				},
				"default": {
					"colorama": {
						"version": "==0.4.6",
						"hashes": ["sha256:abc"],
						"markers": "sys_platform == 'win32'"
					}
				},
				"develop": {}
			}`,
			wantNumDeps: 1,
			checkDeps: []Dependency{
				{Name: "colorama", Version: "0.4.6", Type: "runtime", Source: "pypi"},
			},
		},
		{
			name: "handles packages with extras",
			content: `{
				"_meta": {
					"hash": {
						"sha256": "abc123"
					},
					"pipfile-spec": 6,
					"requires": {},
					"sources": []
				},
				"default": {
					"requests": {
						"version": "==2.28.1",
						"hashes": ["sha256:abc"],
						"extras": ["security", "socks"]
					}
				},
				"develop": {}
			}`,
			wantNumDeps: 1,
			checkDeps: []Dependency{
				{Name: "requests", Version: "2.28.1", Type: "runtime", Source: "pypi"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewPipfileAnalyzer()
			deps, err := analyzer.parsePipfileLock(tt.content)

			if (err != nil) != tt.wantErr {
				t.Errorf("parsePipfileLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(deps) != tt.wantNumDeps {
					t.Errorf("parsePipfileLock() got %d deps, want %d", len(deps), tt.wantNumDeps)
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
