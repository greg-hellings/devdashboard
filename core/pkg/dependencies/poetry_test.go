package dependencies

import (
	"context"
	"errors"
	"testing"

	"github.com/greg-hellings/devdashboard/core/pkg/repository"
)

func TestPoetryAnalyzer_Name(t *testing.T) {
	analyzer := NewPoetryAnalyzer()
	if analyzer.Name() != string(AnalyzerPoetry) {
		t.Errorf("Expected name %s, got %s", AnalyzerPoetry, analyzer.Name())
	}
}

func TestPoetryAnalyzer_CandidateFiles(t *testing.T) {
	tests := []struct {
		name        string
		mockFiles   []repository.FileInfo
		mockError   error
		searchPaths []string
		want        []DependencyFile
		wantErr     bool
	}{
		{
			name: "finds poetry.lock in root",
			mockFiles: []repository.FileInfo{
				{Path: "poetry.lock", Type: "file"},
				{Path: "README.md", Type: "file"},
			},
			searchPaths: []string{""},
			want: []DependencyFile{
				{Path: "poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
			},
			wantErr: false,
		},
		{
			name: "finds multiple poetry.lock files",
			mockFiles: []repository.FileInfo{
				{Path: "poetry.lock", Type: "file"},
				{Path: "backend/poetry.lock", Type: "file"},
				{Path: "frontend/poetry.lock", Type: "file"},
			},
			searchPaths: []string{""},
			want: []DependencyFile{
				{Path: "poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
				{Path: "backend/poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
				{Path: "frontend/poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
			},
			wantErr: false,
		},
		{
			name: "filters by search path",
			mockFiles: []repository.FileInfo{
				{Path: "poetry.lock", Type: "file"},
				{Path: "backend/poetry.lock", Type: "file"},
				{Path: "frontend/poetry.lock", Type: "file"},
			},
			searchPaths: []string{"backend"},
			want: []DependencyFile{
				{Path: "backend/poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
			},
			wantErr: false,
		},
		{
			name: "ignores directories",
			mockFiles: []repository.FileInfo{
				{Path: "poetry.lock", Type: "dir"},
				{Path: "real/poetry.lock", Type: "file"},
			},
			searchPaths: []string{""},
			want: []DependencyFile{
				{Path: "real/poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
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
			analyzer := NewPoetryAnalyzer()

			var config Config
			if tt.name != "returns error when client is nil" {
				config = Config{
					RepositoryPaths: tt.searchPaths,
					RepositoryClient: &mockRepoClient{
						files: tt.mockFiles,
						err:   tt.mockError,
					},
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
					if got[i].Path != tt.want[i].Path {
						t.Errorf("CandidateFiles() got[%d].Path = %v, want %v", i, got[i].Path, tt.want[i].Path)
					}
					if got[i].Type != tt.want[i].Type {
						t.Errorf("CandidateFiles() got[%d].Type = %v, want %v", i, got[i].Type, tt.want[i].Type)
					}
					if got[i].Analyzer != tt.want[i].Analyzer {
						t.Errorf("CandidateFiles() got[%d].Analyzer = %v, want %v", i, got[i].Analyzer, tt.want[i].Analyzer)
					}
				}
			}
		})
	}
}

func TestPoetryAnalyzer_AnalyzeDependencies(t *testing.T) {
	validPoetryLock := `[[package]]
name = "django"
version = "4.2.0"
description = "A high-level Python web framework"
category = "main"
optional = false

[[package]]
name = "requests"
version = "2.28.0"
description = "HTTP library"
category = "main"
optional = false

[metadata]
python-versions = "^3.8"
content-hash = "abc123"
`

	devDepsPoetryLock := `[[package]]
name = "pytest"
version = "7.3.0"
description = "Testing framework"
category = "dev"
optional = false

[[package]]
name = "django"
version = "4.2.0"
description = "Web framework"
category = "main"
optional = false

[metadata]
python-versions = "^3.8"
content-hash = "abc123"
`

	optionalDepsPoetryLock := `[[package]]
name = "extra-package"
version = "1.0.0"
description = "Optional package"
category = "main"
optional = true

[metadata]
python-versions = "^3.8"
content-hash = "abc123"
`

	emptyPoetryLock := `[metadata]
python-versions = "^3.8"
content-hash = "abc123"
`

	invalidPoetryLock := `this is not valid TOML`

	tests := []struct {
		name         string
		files        []DependencyFile
		mockContent  string
		mockError    error
		wantErr      bool
		wantDepCount int
	}{
		{
			name: "parses basic poetry.lock",
			files: []DependencyFile{
				{Path: "poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
			},
			mockContent:  validPoetryLock,
			wantErr:      false,
			wantDepCount: 2,
		},
		{
			name: "handles dev dependencies",
			files: []DependencyFile{
				{Path: "poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
			},
			mockContent:  devDepsPoetryLock,
			wantErr:      false,
			wantDepCount: 2,
		},
		{
			name: "handles optional dependencies",
			files: []DependencyFile{
				{Path: "poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
			},
			mockContent:  optionalDepsPoetryLock,
			wantErr:      false,
			wantDepCount: 1,
		},
		{
			name: "handles empty dependencies",
			files: []DependencyFile{
				{Path: "poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
			},
			mockContent:  emptyPoetryLock,
			wantErr:      false,
			wantDepCount: 0,
		},
		{
			name: "skips invalid files",
			files: []DependencyFile{
				{Path: "invalid.lock", Type: "poetry.lock", Analyzer: "poetry"},
			},
			mockContent: invalidPoetryLock,
			wantErr:     false, // Should not error, just skip the file
		},
		{
			name: "returns error when client is nil",
			files: []DependencyFile{
				{Path: "poetry.lock", Type: "poetry.lock", Analyzer: "poetry"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewPoetryAnalyzer()

			var config Config
			if tt.name != "returns error when client is nil" {
				config = Config{
					RepositoryClient: &mockRepoClient{
						content: tt.mockContent,
						err:     tt.mockError,
					},
				}
			}

			got, err := analyzer.AnalyzeDependencies(context.Background(), "owner", "repo", "main", tt.files, config)

			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeDependencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantDepCount > 0 {
				deps, ok := got[tt.files[0].Path]
				if !ok {
					t.Errorf("AnalyzeDependencies() missing results for %s", tt.files[0].Path)
					return
				}
				if len(deps) != tt.wantDepCount {
					t.Errorf("AnalyzeDependencies() got %d dependencies, want %d", len(deps), tt.wantDepCount)
				}
			}
		})
	}
}

func TestPoetryAnalyzer_ParsePoetryLock(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantErr      bool
		wantCount    int
		wantPackages map[string]string
		wantTypes    map[string]string
	}{
		{
			name: "parses runtime and dev dependencies",
			content: `[[package]]
name = "django"
version = "4.2.0"
description = "Web framework"
category = "main"
optional = false

[[package]]
name = "pytest"
version = "7.3.0"
description = "Testing framework"
category = "dev"
optional = false

[metadata]
python-versions = "^3.8"
content-hash = "abc123"
`,
			wantErr:   false,
			wantCount: 2,
			wantPackages: map[string]string{
				"django": "4.2.0",
				"pytest": "7.3.0",
			},
			wantTypes: map[string]string{
				"django": "runtime",
				"pytest": "dev",
			},
		},
		{
			name: "handles optional dependencies",
			content: `[[package]]
name = "optional-dep"
version = "1.0.0"
description = "Optional package"
category = "main"
optional = true

[metadata]
python-versions = "^3.8"
content-hash = "abc123"
`,
			wantErr:   false,
			wantCount: 1,
			wantPackages: map[string]string{
				"optional-dep": "1.0.0",
			},
			wantTypes: map[string]string{
				"optional-dep": "optional",
			},
		},
		{
			name:      "handles invalid TOML",
			content:   `this is not { valid TOML`,
			wantErr:   true,
			wantCount: 0,
		},
		{
			name: "handles empty package list",
			content: `[metadata]
python-versions = "^3.8"
content-hash = "abc123"
`,
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "handles complex packages",
			content: `[[package]]
name = "sqlalchemy"
version = "2.0.0"
description = "Database toolkit"
category = "main"
optional = false

[[package]]
name = "alembic"
version = "1.10.0"
description = "Database migration tool"
category = "main"
optional = false

[[package]]
name = "black"
version = "23.1.0"
description = "Code formatter"
category = "dev"
optional = false

[metadata]
python-versions = "^3.8"
content-hash = "xyz789"
`,
			wantErr:   false,
			wantCount: 3,
			wantPackages: map[string]string{
				"sqlalchemy": "2.0.0",
				"alembic":    "1.10.0",
				"black":      "23.1.0",
			},
			wantTypes: map[string]string{
				"sqlalchemy": "runtime",
				"alembic":    "runtime",
				"black":      "dev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewPoetryAnalyzer()
			deps, err := analyzer.parsePoetryLock(tt.content)

			if (err != nil) != tt.wantErr {
				t.Errorf("parsePoetryLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if len(deps) != tt.wantCount {
				t.Errorf("parsePoetryLock() got %d dependencies, want %d", len(deps), tt.wantCount)
			}

			// Verify package versions and types
			for _, dep := range deps {
				if expectedVersion, found := tt.wantPackages[dep.Name]; found {
					if dep.Version != expectedVersion {
						t.Errorf("Expected version %s for %s, got %s", expectedVersion, dep.Name, dep.Version)
					}
				}

				if expectedType, found := tt.wantTypes[dep.Name]; found {
					if dep.Type != expectedType {
						t.Errorf("Expected type %s for %s, got %s", expectedType, dep.Name, dep.Type)
					}
				}

				// All poetry packages should have pypi as source
				if dep.Source != "pypi" {
					t.Errorf("Expected source 'pypi' for %s, got '%s'", dep.Name, dep.Source)
				}
			}
		})
	}
}
