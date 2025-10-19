package dependencies

import (
	"context"
	"errors"
	"testing"

	"github.com/greg-hellings/devdashboard/core/pkg/repository"
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
			name: "handles packages with resolution markers for dev",
			content: `version = 1
requires-python = ">=3.8"

[[package]]
name = "test-pkg"
version = "1.0.0"
resolution-markers = "extra == 'dev'"

[package.source]
type = "registry"
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

func TestUvLockAnalyzer_ParseRealUvLockFile(t *testing.T) {
	// Read the actual uv.lock file from tmp directory
	content := `version = 1
revision = 3
requires-python = ">=3.11"

[[package]]
name = "cfgv"
version = "3.4.0"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/11/74/539e56497d9bd1d484fd863dd69cbbfa653cd2aa27abfe35653494d85e94/cfgv-3.4.0.tar.gz", hash = "sha256:e52591d4c5f5dead8e0f673fb16db7949d2cfb3f7da4582893288f0ded8fe560", size = 7114, upload-time = "2023-08-12T20:38:17.776Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/c5/55/51844dd50c4fc7a33b653bfaba4c2456f06955289ca770a5dbd5fd267374/cfgv-3.4.0-py2.py3-none-any.whl", hash = "sha256:b7265b1f29fd3316bfcd2b330d63d024f2bfd8bcb8b0272f8e19a504856c48f9", size = 7249, upload-time = "2023-08-12T20:38:16.269Z" },
]

[[package]]
name = "distlib"
version = "0.4.0"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/96/8e/709914eb2b5749865801041647dc7f4e6d00b549cfe88b65ca192995f07c/distlib-0.4.0.tar.gz", hash = "sha256:feec40075be03a04501a973d81f633735b4b69f98b05450592310c0f401a4e0d", size = 614605, upload-time = "2025-07-17T16:52:00.465Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/33/6b/e0547afaf41bf2c42e52430072fa5658766e3d65bd4b03a563d1b6336f57/distlib-0.4.0-py2.py3-none-any.whl", hash = "sha256:9659f7d87e46584a30b5780e43ac7a2143098441670ff0a49d5f9034c54a6c16", size = 469047, upload-time = "2025-07-17T16:51:58.613Z" },
]

[[package]]
name = "dnspython"
version = "2.8.0"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/8c/8b/57666417c0f90f08bcafa776861060426765fdb422eb10212086fb811d26/dnspython-2.8.0.tar.gz", hash = "sha256:181d3c6996452cb1189c4046c61599b84a5a86e099562ffde77d26984ff26d0f", size = 368251, upload-time = "2025-09-07T18:58:00.022Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/ba/5a/18ad964b0086c6e62e2e7500f7edc89e3faa45033c71c1893d34eed2b2de/dnspython-2.8.0-py3-none-any.whl", hash = "sha256:01d9bbc4a2d76bf0db7c1f729812ded6d912bd318d3b1cf81d30c0f845dbf3af", size = 331094, upload-time = "2025-09-07T18:57:58.071Z" },
]

[[package]]
name = "filelock"
version = "3.20.0"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/58/46/0028a82567109b5ef6e4d2a1f04a583fb513e6cf9527fcdd09afd817deeb/filelock-3.20.0.tar.gz", hash = "sha256:711e943b4ec6be42e1d4e6690b48dc175c822967466bb31c0c293f34334c13f4", size = 18922, upload-time = "2025-10-08T18:03:50.056Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/76/91/7216b27286936c16f5b4d0c530087e4a54eead683e6b0b73dd0c64844af6/filelock-3.20.0-py3-none-any.whl", hash = "sha256:339b4732ffda5cd79b13f4e2711a31b0365ce445d95d243bb996273d072546a2", size = 16054, upload-time = "2025-10-08T18:03:48.35Z" },
]

[[package]]
name = "identify"
version = "2.6.15"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/ff/e7/685de97986c916a6d93b3876139e00eef26ad5bbbd61925d670ae8013449/identify-2.6.15.tar.gz", hash = "sha256:e4f4864b96c6557ef2a1e1c951771838f4edc9df3a72ec7118b338801b11c7bf", size = 99311, upload-time = "2025-10-02T17:43:40.631Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/0f/1c/e5fd8f973d4f375adb21565739498e2e9a1e54c858a97b9a8ccfdc81da9b/identify-2.6.15-py2.py3-none-any.whl", hash = "sha256:1181ef7608e00704db228516541eb83a88a9f94433a8c80bb9b5bd54b1d81757", size = 99183, upload-time = "2025-10-02T17:43:39.137Z" },
]

[[package]]
name = "nodeenv"
version = "1.9.1"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/43/16/fc88b08840de0e0a72a2f9d8c6bae36be573e475a6326ae854bcc549fc45/nodeenv-1.9.1.tar.gz", hash = "sha256:6ec12890a2dab7946721edbfbcd91f3319c6ccc9aec47be7c7e6b7011ee6645f", size = 47437, upload-time = "2024-06-04T18:44:11.171Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/d2/1d/1b658dbd2b9fa9c4c9f32accbfc0205d532c8c6194dc0f2a4c0428e7128a/nodeenv-1.9.1-py2.py3-none-any.whl", hash = "sha256:ba11c9782d29c27c70ffbdda2d7415098754709be8a7056d79a737cd901155c9", size = 22314, upload-time = "2024-06-04T18:44:08.352Z" },
]

[[package]]
name = "platformdirs"
version = "4.5.0"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/61/33/9611380c2bdb1225fdef633e2a9610622310fed35ab11dac9620972ee088/platformdirs-4.5.0.tar.gz", hash = "sha256:70ddccdd7c99fc5942e9fc25636a8b34d04c24b335100223152c2803e4063312", size = 21632, upload-time = "2025-10-08T17:44:48.791Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/73/cb/ac7874b3e5d58441674fb70742e6c374b28b0c7cb988d37d991cde47166c/platformdirs-4.5.0-py3-none-any.whl", hash = "sha256:e578a81bb873cbb89a41fcc904c7ef523cc18284b7e3b3ccf06aca1403b7ebd3", size = 18651, upload-time = "2025-10-08T17:44:47.223Z" },
]

[[package]]
name = "pre-commit"
version = "4.3.0"
source = { registry = "https://pypidev.ivrtechnology.com/" }
dependencies = [
    { name = "cfgv" },
    { name = "identify" },
    { name = "nodeenv" },
    { name = "pyyaml" },
    { name = "virtualenv" },
]
sdist = { url = "https://files.pythonhosted.org/packages/ff/29/7cf5bbc236333876e4b41f56e06857a87937ce4bf91e117a6991a2dbb02a/pre_commit-4.3.0.tar.gz", hash = "sha256:499fe450cc9d42e9d58e606262795ecb64dd05438943c62b66f6a8673da30b16", size = 193792, upload-time = "2025-08-09T18:56:14.651Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/5b/a5/987a405322d78a73b66e39e4a90e4ef156fd7141bf71df987e50717c321b/pre_commit-4.3.0-py2.py3-none-any.whl", hash = "sha256:2b0747ad7e6e967169136edffee14c16e148a778a54e4f967921aa1ebf2308d8", size = 220965, upload-time = "2025-08-09T18:56:13.192Z" },
]

[[package]]
name = "pyyaml"
version = "6.0.3"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/05/8e/961c0007c59b8dd7729d542c61a4d537767a59645b82a0b521206e1e25c2/pyyaml-6.0.3.tar.gz", hash = "sha256:d76623373421df22fb4cf8817020cbb7ef15c725b9d5e45f17e189bfc384190f", size = 130960, upload-time = "2025-09-25T21:33:16.546Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/6d/16/a95b6757765b7b031c9374925bb718d55e0a9ba8a1b6a12d25962ea44347/pyyaml-6.0.3-cp311-cp311-macosx_10_13_x86_64.whl", hash = "sha256:44edc647873928551a01e7a563d7452ccdebee747728c1080d881d68af7b997e", size = 185826, upload-time = "2025-09-25T21:31:58.655Z" },
    { url = "https://files.pythonhosted.org/packages/16/19/13de8e4377ed53079ee996e1ab0a9c33ec2faf808a4647b7b4c0d46dd239/pyyaml-6.0.3-cp311-cp311-macosx_11_0_arm64.whl", hash = "sha256:652cb6edd41e718550aad172851962662ff2681490a8a711af6a4d288dd96824", size = 175577, upload-time = "2025-09-25T21:32:00.088Z" },
    { url = "https://files.pythonhosted.org/packages/0c/62/d2eb46264d4b157dae1275b573017abec435397aa59cbcdab6fc978a8af4/pyyaml-6.0.3-cp311-cp311-manylinux2014_aarch64.manylinux_2_17_aarch64.manylinux_2_28_aarch64.whl", hash = "sha256:10892704fc220243f5305762e276552a0395f7beb4dbf9b14ec8fd43b57f126c", size = 775556, upload-time = "2025-09-25T21:32:01.31Z" },
    { url = "https://files.pythonhosted.org/packages/10/cb/16c3f2cf3266edd25aaa00d6c4350381c8b012ed6f5276675b9eba8d9ff4/pyyaml-6.0.3-cp311-cp311-manylinux2014_s390x.manylinux_2_17_s390x.manylinux_2_28_s390x.whl", hash = "sha256:850774a7879607d3a6f50d36d04f00ee69e7fc816450e5f7e58d7f17f1ae5c00", size = 882114, upload-time = "2025-09-25T21:32:03.376Z" },
    { url = "https://files.pythonhosted.org/packages/71/60/917329f640924b18ff085ab889a11c763e0b573da888e8404ff486657602/pyyaml-6.0.3-cp311-cp311-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl", hash = "sha256:b8bb0864c5a28024fac8a632c443c87c5aa6f215c0b126c449ae1a150412f31d", size = 806638, upload-time = "2025-09-25T21:32:04.553Z" },
    { url = "https://files.pythonhosted.org/packages/dd/6f/529b0f316a9fd167281a6c3826b5583e6192dba792dd55e3203d3f8e655a/pyyaml-6.0.3-cp311-cp311-musllinux_1_2_aarch64.whl", hash = "sha256:1d37d57ad971609cf3c53ba6a7e365e40660e3be0e5175fa9f2365a379d6095a", size = 767463, upload-time = "2025-09-25T21:32:06.152Z" },
    { url = "https://files.pythonhosted.org/packages/f2/6a/b627b4e0c1dd03718543519ffb2f1deea4a1e6d42fbab8021936a4d22589/pyyaml-6.0.3-cp311-cp311-musllinux_1_2_x86_64.whl", hash = "sha256:37503bfbfc9d2c40b344d06b2199cf0e96e97957ab1c1b546fd4f87e53e5d3e4", size = 794986, upload-time = "2025-09-25T21:32:07.367Z" },
    { url = "https://files.pythonhosted.org/packages/45/91/47a6e1c42d9ee337c4839208f30d9f09caa9f720ec7582917b264defc875/pyyaml-6.0.3-cp311-cp311-win32.whl", hash = "sha256:8098f252adfa6c80ab48096053f512f2321f0b998f98150cea9bd23d83e1467b", size = 142543, upload-time = "2025-09-25T21:32:08.95Z" },
    { url = "https://files.pythonhosted.org/packages/da/e3/ea007450a105ae919a72393cb06f122f288ef60bba2dc64b26e2646fa315/pyyaml-6.0.3-cp311-cp311-win_amd64.whl", hash = "sha256:9f3bfb4965eb874431221a3ff3fdcddc7e74e3b07799e0e84ca4a0f867d449bf", size = 158763, upload-time = "2025-09-25T21:32:09.96Z" },
    { url = "https://files.pythonhosted.org/packages/d1/33/422b98d2195232ca1826284a76852ad5a86fe23e31b009c9886b2d0fb8b2/pyyaml-6.0.3-cp312-cp312-macosx_10_13_x86_64.whl", hash = "sha256:7f047e29dcae44602496db43be01ad42fc6f1cc0d8cd6c83d342306c32270196", size = 182063, upload-time = "2025-09-25T21:32:11.445Z" },
    { url = "https://files.pythonhosted.org/packages/89/a0/6cf41a19a1f2f3feab0e9c0b74134aa2ce6849093d5517a0c550fe37a648/pyyaml-6.0.3-cp312-cp312-macosx_11_0_arm64.whl", hash = "sha256:fc09d0aa354569bc501d4e787133afc08552722d3ab34836a80547331bb5d4a0", size = 173973, upload-time = "2025-09-25T21:32:12.492Z" },
    { url = "https://files.pythonhosted.org/packages/ed/23/7a778b6bd0b9a8039df8b1b1d80e2e2ad78aa04171592c8a5c43a56a6af4/pyyaml-6.0.3-cp312-cp312-manylinux2014_aarch64.manylinux_2_17_aarch64.manylinux_2_28_aarch64.whl", hash = "sha256:9149cad251584d5fb4981be1ecde53a1ca46c891a79788c0df828d2f166bda28", size = 775116, upload-time = "2025-09-25T21:32:13.652Z" },
    { url = "https://files.pythonhosted.org/packages/65/30/d7353c338e12baef4ecc1b09e877c1970bd3382789c159b4f89d6a70dc09/pyyaml-6.0.3-cp312-cp312-manylinux2014_s390x.manylinux_2_17_s390x.manylinux_2_28_s390x.whl", hash = "sha256:5fdec68f91a0c6739b380c83b951e2c72ac0197ace422360e6d5a959d8d97b2c", size = 844011, upload-time = "2025-09-25T21:32:15.21Z" },
    { url = "https://files.pythonhosted.org/packages/8b/9d/b3589d3877982d4f2329302ef98a8026e7f4443c765c46cfecc8858c6b4b/pyyaml-6.0.3-cp312-cp312-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl", hash = "sha256:ba1cc08a7ccde2d2ec775841541641e4548226580ab850948cbfda66a1befcdc", size = 807870, upload-time = "2025-09-25T21:32:16.431Z" },
    { url = "https://files.pythonhosted.org/packages/05/c0/b3be26a015601b822b97d9149ff8cb5ead58c66f981e04fedf4e762f4bd4/pyyaml-6.0.3-cp312-cp312-musllinux_1_2_aarch64.whl", hash = "sha256:8dc52c23056b9ddd46818a57b78404882310fb473d63f17b07d5c40421e47f8e", size = 761089, upload-time = "2025-09-25T21:32:17.56Z" },
    { url = "https://files.pythonhosted.org/packages/be/8e/98435a21d1d4b46590d5459a22d88128103f8da4c2d4cb8f14f2a96504e1/pyyaml-6.0.3-cp312-cp312-musllinux_1_2_x86_64.whl", hash = "sha256:41715c910c881bc081f1e8872880d3c650acf13dfa8214bad49ed4cede7c34ea", size = 790181, upload-time = "2025-09-25T21:32:18.834Z" },
    { url = "https://files.pythonhosted.org/packages/74/93/7baea19427dcfbe1e5a372d81473250b379f04b1bd3c4c5ff825e2327202/pyyaml-6.0.3-cp312-cp312-win32.whl", hash = "sha256:96b533f0e99f6579b3d4d4995707cf36df9100d67e0c8303a0c55b27b5f99bc5", size = 137658, upload-time = "2025-09-25T21:32:20.209Z" },
    { url = "https://files.pythonhosted.org/packages/86/bf/899e81e4cce32febab4fb42bb97dcdf66bc135272882d1987881a4b519e9/pyyaml-6.0.3-cp312-cp312-win_amd64.whl", hash = "sha256:5fcd34e47f6e0b794d17de1b4ff496c00986e1c83f7ab2fb8fcfe9616ff7477b", size = 154003, upload-time = "2025-09-25T21:32:21.167Z" },
    { url = "https://files.pythonhosted.org/packages/1a/08/67bd04656199bbb51dbed1439b7f27601dfb576fb864099c7ef0c3e55531/pyyaml-6.0.3-cp312-cp312-win_arm64.whl", hash = "sha256:64386e5e707d03a7e172c0701abfb7e10f0fb753ee1d773128192742712a98fd", size = 140344, upload-time = "2025-09-25T21:32:22.617Z" },
    { url = "https://files.pythonhosted.org/packages/d1/11/0fd08f8192109f7169db964b5707a2f1e8b745d4e239b784a5a1dd80d1db/pyyaml-6.0.3-cp313-cp313-macosx_10_13_x86_64.whl", hash = "sha256:8da9669d359f02c0b91ccc01cac4a67f16afec0dac22c2ad09f46bee0697eba8", size = 181669, upload-time = "2025-09-25T21:32:23.673Z" },
    { url = "https://files.pythonhosted.org/packages/b1/16/95309993f1d3748cd644e02e38b75d50cbc0d9561d21f390a76242ce073f/pyyaml-6.0.3-cp313-cp313-macosx_11_0_arm64.whl", hash = "sha256:2283a07e2c21a2aa78d9c4442724ec1eb15f5e42a723b99cb3d822d48f5f7ad1", size = 173252, upload-time = "2025-09-25T21:32:25.149Z" },
    { url = "https://files.pythonhosted.org/packages/50/31/b20f376d3f810b9b2371e72ef5adb33879b25edb7a6d072cb7ca0c486398/pyyaml-6.0.3-cp313-cp313-manylinux2014_aarch64.manylinux_2_17_aarch64.manylinux_2_28_aarch64.whl", hash = "sha256:ee2922902c45ae8ccada2c5b501ab86c36525b883eff4255313a253a3160861c", size = 767081, upload-time = "2025-09-25T21:32:26.575Z" },
    { url = "https://files.pythonhosted.org/packages/49/1e/a55ca81e949270d5d4432fbbd19dfea5321eda7c41a849d443dc92fd1ff7/pyyaml-6.0.3-cp313-cp313-manylinux2014_s390x.manylinux_2_17_s390x.manylinux_2_28_s390x.whl", hash = "sha256:a33284e20b78bd4a18c8c2282d549d10bc8408a2a7ff57653c0cf0b9be0afce5", size = 841159, upload-time = "2025-09-25T21:32:27.727Z" },
    { url = "https://files.pythonhosted.org/packages/74/27/e5b8f34d02d9995b80abcef563ea1f8b56d20134d8f4e5e81733b1feceb2/pyyaml-6.0.3-cp313-cp313-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl", hash = "sha256:0f29edc409a6392443abf94b9cf89ce99889a1dd5376d94316ae5145dfedd5d6", size = 801626, upload-time = "2025-09-25T21:32:28.878Z" },
    { url = "https://files.pythonhosted.org/packages/f9/11/ba845c23988798f40e52ba45f34849aa8a1f2d4af4b798588010792ebad6/pyyaml-6.0.3-cp313-cp313-musllinux_1_2_aarch64.whl", hash = "sha256:f7057c9a337546edc7973c0d3ba84ddcdf0daa14533c2065749c9075001090e6", size = 753613, upload-time = "2025-09-25T21:32:30.178Z" },
    { url = "https://files.pythonhosted.org/packages/3d/e0/7966e1a7bfc0a45bf0a7fb6b98ea03fc9b8d84fa7f2229e9659680b69ee3/pyyaml-6.0.3-cp313-cp313-musllinux_1_2_x86_64.whl", hash = "sha256:eda16858a3cab07b80edaf74336ece1f986ba330fdb8ee0d6c0d68fe82bc96be", size = 794115, upload-time = "2025-09-25T21:32:31.353Z" },
    { url = "https://files.pythonhosted.org/packages/de/94/980b50a6531b3019e45ddeada0626d45fa85cbe22300844a7983285bed3b/pyyaml-6.0.3-cp313-cp313-win32.whl", hash = "sha256:d0eae10f8159e8fdad514efdc92d74fd8d682c933a6dd088030f3834bc8e6b26", size = 137427, upload-time = "2025-09-25T21:32:32.58Z" },
    { url = "https://files.pythonhosted.org/packages/97/c9/39d5b874e8b28845e4ec2202b5da735d0199dbe5b8fb85f91398814a9a46/pyyaml-6.0.3-cp313-cp313-win_amd64.whl", hash = "sha256:79005a0d97d5ddabfeeea4cf676af11e647e41d81c9a7722a193022accdb6b7c", size = 154090, upload-time = "2025-09-25T21:32:33.659Z" },
    { url = "https://files.pythonhosted.org/packages/73/e8/2bdf3ca2090f68bb3d75b44da7bbc71843b19c9f2b9cb9b0f4ab7a5a4329/pyyaml-6.0.3-cp313-cp313-win_arm64.whl", hash = "sha256:5498cd1645aa724a7c71c8f378eb29ebe23da2fc0d7a08071d89469bf1d2defb", size = 140246, upload-time = "2025-09-25T21:32:34.663Z" },
    { url = "https://files.pythonhosted.org/packages/9d/8c/f4bd7f6465179953d3ac9bc44ac1a8a3e6122cf8ada906b4f96c60172d43/pyyaml-6.0.3-cp314-cp314-macosx_10_13_x86_64.whl", hash = "sha256:8d1fab6bb153a416f9aeb4b8763bc0f22a5586065f86f7664fc23339fc1c1fac", size = 181814, upload-time = "2025-09-25T21:32:35.712Z" },
    { url = "https://files.pythonhosted.org/packages/bd/9c/4d95bb87eb2063d20db7b60faa3840c1b18025517ae857371c4dd55a6b3a/pyyaml-6.0.3-cp314-cp314-macosx_11_0_arm64.whl", hash = "sha256:34d5fcd24b8445fadc33f9cf348c1047101756fd760b4dacb5c3e99755703310", size = 173809, upload-time = "2025-09-25T21:32:36.789Z" },
    { url = "https://files.pythonhosted.org/packages/92/b5/47e807c2623074914e29dabd16cbbdd4bf5e9b2db9f8090fa64411fc5382/pyyaml-6.0.3-cp314-cp314-manylinux2014_aarch64.manylinux_2_17_aarch64.manylinux_2_28_aarch64.whl", hash = "sha256:501a031947e3a9025ed4405a168e6ef5ae3126c59f90ce0cd6f2bfc477be31b7", size = 766454, upload-time = "2025-09-25T21:32:37.966Z" },
    { url = "https://files.pythonhosted.org/packages/02/9e/e5e9b168be58564121efb3de6859c452fccde0ab093d8438905899a3a483/pyyaml-6.0.3-cp314-cp314-manylinux2014_s390x.manylinux_2_17_s390x.manylinux_2_28_s390x.whl", hash = "sha256:b3bc83488de33889877a0f2543ade9f70c67d66d9ebb4ac959502e12de895788", size = 836355, upload-time = "2025-09-25T21:32:39.178Z" },
    { url = "https://files.pythonhosted.org/packages/88/f9/16491d7ed2a919954993e48aa941b200f38040928474c9e85ea9e64222c3/pyyaml-6.0.3-cp314-cp314-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl", hash = "sha256:c458b6d084f9b935061bc36216e8a69a7e293a2f1e68bf956dcd9e6cbcd143f5", size = 794175, upload-time = "2025-09-25T21:32:40.865Z" },
    { url = "https://files.pythonhosted.org/packages/dd/3f/5989debef34dc6397317802b527dbbafb2b4760878a53d4166579111411e/pyyaml-6.0.3-cp314-cp314-musllinux_1_2_aarch64.whl", hash = "sha256:7c6610def4f163542a622a73fb39f534f8c101d690126992300bf3207eab9764", size = 755228, upload-time = "2025-09-25T21:32:42.084Z" },
    { url = "https://files.pythonhosted.org/packages/d7/ce/af88a49043cd2e265be63d083fc75b27b6ed062f5f9fd6cdc223ad62f03e/pyyaml-6.0.3-cp314-cp314-musllinux_1_2_x86_64.whl", hash = "sha256:5190d403f121660ce8d1d2c1bb2ef1bd05b5f68533fc5c2ea899bd15f4399b35", size = 789194, upload-time = "2025-09-25T21:32:43.362Z" },
    { url = "https://files.pythonhosted.org/packages/23/20/bb6982b26a40bb43951265ba29d4c246ef0ff59c9fdcdf0ed04e0687de4d/pyyaml-6.0.3-cp314-cp314-win_amd64.whl", hash = "sha256:4a2e8cebe2ff6ab7d1050ecd59c25d4c8bd7e6f400f5f82b96557ac0abafd0ac", size = 156429, upload-time = "2025-09-25T21:32:57.844Z" },
    { url = "https://files.pythonhosted.org/packages/f4/f4/a4541072bb9422c8a883ab55255f918fa378ecf083f5b85e87fc2b4eda1b/pyyaml-6.0.3-cp314-cp314-win_arm64.whl", hash = "sha256:93dda82c9c22deb0a405ea4dc5f2d0cda384168e466364dec6255b293923b2f3", size = 143912, upload-time = "2025-09-25T21:32:59.247Z" },
    { url = "https://files.pythonhosted.org/packages/7c/f9/07dd09ae774e4616edf6cda684ee78f97777bdd15847253637a6f052a62f/pyyaml-6.0.3-cp314-cp314t-macosx_10_13_x86_64.whl", hash = "sha256:02893d100e99e03eda1c8fd5c441d8c60103fd175728e23e431db1b589cf5ab3", size = 189108, upload-time = "2025-09-25T21:32:44.377Z" },
    { url = "https://files.pythonhosted.org/packages/4e/78/8d08c9fb7ce09ad8c38ad533c1191cf27f7ae1effe5bb9400a46d9437fcf/pyyaml-6.0.3-cp314-cp314t-macosx_11_0_arm64.whl", hash = "sha256:c1ff362665ae507275af2853520967820d9124984e0f7466736aea23d8611fba", size = 183641, upload-time = "2025-09-25T21:32:45.407Z" },
    { url = "https://files.pythonhosted.org/packages/7b/5b/3babb19104a46945cf816d047db2788bcaf8c94527a805610b0289a01c6b/pyyaml-6.0.3-cp314-cp314t-manylinux2014_aarch64.manylinux_2_17_aarch64.manylinux_2_28_aarch64.whl", hash = "sha256:6adc77889b628398debc7b65c073bcb99c4a0237b248cacaf3fe8a557563ef6c", size = 831901, upload-time = "2025-09-25T21:32:48.83Z" },
    { url = "https://files.pythonhosted.org/packages/8b/cc/dff0684d8dc44da4d22a13f35f073d558c268780ce3c6ba1b87055bb0b87/pyyaml-6.0.3-cp314-cp314t-manylinux2014_s390x.manylinux_2_17_s390x.manylinux_2_28_s390x.whl", hash = "sha256:a80cb027f6b349846a3bf6d73b5e95e782175e52f22108cfa17876aaeff93702", size = 861132, upload-time = "2025-09-25T21:32:50.149Z" },
    { url = "https://files.pythonhosted.org/packages/b1/5e/f77dc6b9036943e285ba76b49e118d9ea929885becb0a29ba8a7c75e29fe/pyyaml-6.0.3-cp314-cp314t-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl", hash = "sha256:00c4bdeba853cc34e7dd471f16b4114f4162dc03e6b7afcc2128711f0eca823c", size = 839261, upload-time = "2025-09-25T21:32:51.808Z" },
    { url = "https://files.pythonhosted.org/packages/ce/88/a9db1376aa2a228197c58b37302f284b5617f56a5d959fd1763fb1675ce6/pyyaml-6.0.3-cp314-cp314t-musllinux_1_2_aarch64.whl", hash = "sha256:66e1674c3ef6f541c35191caae2d429b967b99e02040f5ba928632d9a7f0f065", size = 805272, upload-time = "2025-09-25T21:32:52.941Z" },
    { url = "https://files.pythonhosted.org/packages/da/92/1446574745d74df0c92e6aa4a7b0b3130706a4142b2d1a5869f2eaa423c6/pyyaml-6.0.3-cp314-cp314t-musllinux_1_2_x86_64.whl", hash = "sha256:16249ee61e95f858e83976573de0f5b2893b3677ba71c9dd36b9cf8be9ac6d65", size = 829923, upload-time = "2025-09-25T21:32:54.537Z" },
    { url = "https://files.pythonhosted.org/packages/f0/7a/1c7270340330e575b92f397352af856a8c06f230aa3e76f86b39d01b416a/pyyaml-6.0.3-cp314-cp314t-win_amd64.whl", hash = "sha256:4ad1906908f2f5ae4e5a8ddfce73c320c2a1429ec52eafd27138b7f1cbe341c9", size = 174062, upload-time = "2025-09-25T21:32:55.767Z" },
    { url = "https://files.pythonhosted.org/packages/f1/12/de94a39c2ef588c7e6455cfbe7343d3b2dc9d6b6b2f40c4c6565744c873d/pyyaml-6.0.3-cp314-cp314t-win_arm64.whl", hash = "sha256:ebc55a14a21cb14062aa4162f906cd962b28e2e9ea38f9b4391244cd8de4ae0b", size = 149341, upload-time = "2025-09-25T21:32:56.828Z" },
]

[[package]]
name = "tmp"
version = "0.1.0"
source = { virtual = "." }
dependencies = [
    { name = "dnspython" },
]

[package.dev-dependencies]
dev = [
    { name = "pre-commit" },
]

[package.metadata]
requires-dist = [{ name = "dnspython", specifier = ">=2.8.0" }]

[package.metadata.requires-dev]
dev = [{ name = "pre-commit", specifier = ">=4.3.0" }]

[[package]]
name = "virtualenv"
version = "20.35.3"
source = { registry = "https://pypidev.ivrtechnology.com/" }
dependencies = [
    { name = "distlib" },
    { name = "filelock" },
    { name = "platformdirs" },
]
sdist = { url = "https://files.pythonhosted.org/packages/a4/d5/b0ccd381d55c8f45d46f77df6ae59fbc23d19e901e2d523395598e5f4c93/virtualenv-20.35.3.tar.gz", hash = "sha256:4f1a845d131133bdff10590489610c98c168ff99dc75d6c96853801f7f67af44", size = 6002907, upload-time = "2025-10-10T21:23:33.178Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/27/73/d9a94da0e9d470a543c1b9d3ccbceb0f59455983088e727b8a1824ed90fb/virtualenv-20.35.3-py3-none-any.whl", hash = "sha256:63d106565078d8c8d0b206d48080f938a8b25361e19432d2c9db40d2899c810a", size = 5981061, upload-time = "2025-10-10T21:23:30.433Z" },
]
`

	analyzer := NewUvLockAnalyzer()
	deps, err := analyzer.parseUvLock(content)

	if err != nil {
		t.Fatalf("Failed to parse real uv.lock file: %v", err)
	}

	// Verify we got dependencies
	if len(deps) == 0 {
		t.Error("Expected to find dependencies in real uv.lock file, got 0")
	}

	t.Logf("Successfully parsed real uv.lock file with %d dependencies", len(deps))

	// Verify some expected packages from the file
	expectedPackages := map[string]struct{}{
		"cfgv":       {},
		"distlib":    {},
		"dnspython":  {},
		"filelock":   {},
		"identify":   {},
		"virtualenv": {},
		"tmp":        {},
	}

	foundPackages := make(map[string]bool)
	for _, dep := range deps {
		if _, expected := expectedPackages[dep.Name]; expected {
			foundPackages[dep.Name] = true
			t.Logf("Found expected package: %s v%s (type: %s, source: %s)",
				dep.Name, dep.Version, dep.Type, dep.Source)
		}
	}

	// Verify all expected packages were found
	for pkg := range expectedPackages {
		if !foundPackages[pkg] {
			t.Errorf("Expected package %s not found in parsed dependencies", pkg)
		}
	}

	// Verify the tmp package is marked as dev (it has dev-dependencies)
	var tmpPkg *Dependency
	for i := range deps {
		if deps[i].Name == "tmp" {
			tmpPkg = &deps[i]
			break
		}
	}

	if tmpPkg == nil {
		t.Error("Package 'tmp' not found in dependencies")
	} else {
		if tmpPkg.Type != "dev" {
			t.Errorf("Package 'tmp' should be marked as 'dev' type, got '%s'", tmpPkg.Type)
		}
		if tmpPkg.Source != "pypi" {
			t.Errorf("Package 'tmp' should have source 'pypi', got '%s'", tmpPkg.Source)
		}
		t.Logf("Package 'tmp' correctly identified as dev dependency")
	}

	// Verify runtime packages
	runtimeCount := 0
	devCount := 0
	for _, dep := range deps {
		switch dep.Type {
		case "runtime":
			runtimeCount++
		case "dev":
			devCount++
		}
	}

	t.Logf("Dependency breakdown: %d runtime, %d dev", runtimeCount, devCount)

	if runtimeCount == 0 {
		t.Error("Expected at least one runtime dependency")
	}
}

func TestUvLockAnalyzer_AnalyzeRealUvLockFile(t *testing.T) {
	// This test performs a full integration test with the real uv.lock file
	// It tests the complete flow: finding files, reading content, and parsing

	// Create a mock client that returns the actual file content
	content := `version = 1
revision = 3
requires-python = ">=3.11"

[[package]]
name = "cfgv"
version = "3.4.0"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/11/74/539e56497d9bd1d484fd863dd69cbbfa653cd2aa27abfe35653494d85e94/cfgv-3.4.0.tar.gz", hash = "sha256:e52591d4c5f5dead8e0f673fb16db7949d2cfb3f7da4582893288f0ded8fe560", size = 7114, upload-time = "2023-08-12T20:38:17.776Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/c5/55/51844dd50c4fc7a33b653bfaba4c2456f06955289ca770a5dbd5fd267374/cfgv-3.4.0-py2.py3-none-any.whl", hash = "sha256:b7265b1f29fd3316bfcd2b330d63d024f2bfd8bcb8b0272f8e19a504856c48f9", size = 7249, upload-time = "2023-08-12T20:38:16.269Z" },
]

[[package]]
name = "distlib"
version = "0.4.0"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/96/8e/709914eb2b5749865801041647dc7f4e6d00b549cfe88b65ca192995f07c/distlib-0.4.0.tar.gz", hash = "sha256:feec40075be03a04501a973d81f633735b4b69f98b05450592310c0f401a4e0d", size = 614605, upload-time = "2025-07-17T16:52:00.465Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/33/6b/e0547afaf41bf2c42e52430072fa5658766e3d65bd4b03a563d1b6336f57/distlib-0.4.0-py2.py3-none-any.whl", hash = "sha256:9659f7d87e46584a30b5780e43ac7a2143098441670ff0a49d5f9034c54a6c16", size = 469047, upload-time = "2025-07-17T16:51:58.613Z" },
]

[[package]]
name = "dnspython"
version = "2.8.0"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/8c/8b/57666417c0f90f08bcafa776861060426765fdb422eb10212086fb811d26/dnspython-2.8.0.tar.gz", hash = "sha256:181d3c6996452cb1189c4046c61599b84a5a86e099562ffde77d26984ff26d0f", size = 368251, upload-time = "2025-09-07T18:58:00.022Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/ba/5a/18ad964b0086c6e62e2e7500f7edc89e3faa45033c71c1893d34eed2b2de/dnspython-2.8.0-py3-none-any.whl", hash = "sha256:01d9bbc4a2d76bf0db7c1f729812ded6d912bd318d3b1cf81d30c0f845dbf3af", size = 331094, upload-time = "2025-09-07T18:57:58.071Z" },
]

[[package]]
name = "filelock"
version = "3.20.0"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/58/46/0028a82567109b5ef6e4d2a1f04a583fb513e6cf9527fcdd09afd817deeb/filelock-3.20.0.tar.gz", hash = "sha256:711e943b4ec6be42e1d4e6690b48dc175c822967466bb31c0c293f34334c13f4", size = 18922, upload-time = "2025-10-08T18:03:50.056Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/76/91/7216b27286936c16f5b4d0c530087e4a54eead683e6b0b73dd0c64844af6/filelock-3.20.0-py3-none-any.whl", hash = "sha256:339b4732ffda5cd79b13f4e2711a31b0365ce445d95d243bb996273d072546a2", size = 16054, upload-time = "2025-10-08T18:03:48.35Z" },
]

[[package]]
name = "identify"
version = "2.6.15"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/ff/e7/685de97986c916a6d93b3876139e00eef26ad5bbbd61925d670ae8013449/identify-2.6.15.tar.gz", hash = "sha256:e4f4864b96c6557ef2a1e1c951771838f4edc9df3a72ec7118b338801b11c7bf", size = 99311, upload-time = "2025-10-02T17:43:40.631Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/0f/1c/e5fd8f973d4f375adb21565739498e2e9a1e54c858a97b9a8ccfdc81da9b/identify-2.6.15-py2.py3-none-any.whl", hash = "sha256:1181ef7608e00704db228516541eb83a88a9f94433a8c80bb9b5bd54b1d81757", size = 99183, upload-time = "2025-10-02T17:43:39.137Z" },
]

[[package]]
name = "nodeenv"
version = "1.9.1"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/43/16/fc88b08840de0e0a72a2f9d8c6bae36be573e475a6326ae854bcc549fc45/nodeenv-1.9.1.tar.gz", hash = "sha256:6ec12890a2dab7946721edbfbcd91f3319c6ccc9aec47be7c7e6b7011ee6645f", size = 47437, upload-time = "2024-06-04T18:44:11.171Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/d2/1d/1b658dbd2b9fa9c4c9f32accbfc0205d532c8c6194dc0f2a4c0428e7128a/nodeenv-1.9.1-py2.py3-none-any.whl", hash = "sha256:ba11c9782d29c27c70ffbdda2d7415098754709be8a7056d79a737cd901155c9", size = 22314, upload-time = "2024-06-04T18:44:08.352Z" },
]

[[package]]
name = "platformdirs"
version = "4.5.0"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/61/33/9611380c2bdb1225fdef633e2a9610622310fed35ab11dac9620972ee088/platformdirs-4.5.0.tar.gz", hash = "sha256:70ddccdd7c99fc5942e9fc25636a8b34d04c24b335100223152c2803e4063312", size = 21632, upload-time = "2025-10-08T17:44:48.791Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/73/cb/ac7874b3e5d58441674fb70742e6c374b28b0c7cb988d37d991cde47166c/platformdirs-4.5.0-py3-none-any.whl", hash = "sha256:e578a81bb873cbb89a41fcc904c7ef523cc18284b7e3b3ccf06aca1403b7ebd3", size = 18651, upload-time = "2025-10-08T17:44:47.223Z" },
]

[[package]]
name = "pre-commit"
version = "4.3.0"
source = { registry = "https://pypidev.ivrtechnology.com/" }
dependencies = [
    { name = "cfgv" },
    { name = "identify" },
    { name = "nodeenv" },
    { name = "pyyaml" },
    { name = "virtualenv" },
]
sdist = { url = "https://files.pythonhosted.org/packages/ff/29/7cf5bbc236333876e4b41f56e06857a87937ce4bf91e117a6991a2dbb02a/pre_commit-4.3.0.tar.gz", hash = "sha256:499fe450cc9d42e9d58e606262795ecb64dd05438943c62b66f6a8673da30b16", size = 193792, upload-time = "2025-08-09T18:56:14.651Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/5b/a5/987a405322d78a73b66e39e4a90e4ef156fd7141bf71df987e50717c321b/pre_commit-4.3.0-py2.py3-none-any.whl", hash = "sha256:2b0747ad7e6e967169136edffee14c16e148a778a54e4f967921aa1ebf2308d8", size = 220965, upload-time = "2025-08-09T18:56:13.192Z" },
]

[[package]]
name = "pyyaml"
version = "6.0.3"
source = { registry = "https://pypidev.ivrtechnology.com/" }
sdist = { url = "https://files.pythonhosted.org/packages/05/8e/961c0007c59b8dd7729d542c61a4d537767a59645b82a0b521206e1e25c2/pyyaml-6.0.3.tar.gz", hash = "sha256:d76623373421df22fb4cf8817020cbb7ef15c725b9d5e45f17e189bfc384190f", size = 130960, upload-time = "2025-09-25T21:33:16.546Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/6d/16/a95b6757765b7b031c9374925bb718d55e0a9ba8a1b6a12d25962ea44347/pyyaml-6.0.3-cp311-cp311-macosx_10_13_x86_64.whl", hash = "sha256:44edc647873928551a01e7a563d7452ccdebee747728c1080d881d68af7b997e", size = 185826, upload-time = "2025-09-25T21:31:58.655Z" },
    { url = "https://files.pythonhosted.org/packages/16/19/13de8e4377ed53079ee996e1ab0a9c33ec2faf808a4647b7b4c0d46dd239/pyyaml-6.0.3-cp311-cp311-macosx_11_0_arm64.whl", hash = "sha256:652cb6edd41e718550aad172851962662ff2681490a8a711af6a4d288dd96824", size = 175577, upload-time = "2025-09-25T21:32:00.088Z" },
    { url = "https://files.pythonhosted.org/packages/0c/62/d2eb46264d4b157dae1275b573017abec435397aa59cbcdab6fc978a8af4/pyyaml-6.0.3-cp311-cp311-manylinux2014_aarch64.manylinux_2_17_aarch64.manylinux_2_28_aarch64.whl", hash = "sha256:10892704fc220243f5305762e276552a0395f7beb4dbf9b14ec8fd43b57f126c", size = 775556, upload-time = "2025-09-25T21:32:01.31Z" },
    { url = "https://files.pythonhosted.org/packages/10/cb/16c3f2cf3266edd25aaa00d6c4350381c8b012ed6f5276675b9eba8d9ff4/pyyaml-6.0.3-cp311-cp311-manylinux2014_s390x.manylinux_2_17_s390x.manylinux_2_28_s390x.whl", hash = "sha256:850774a7879607d3a6f50d36d04f00ee69e7fc816450e5f7e58d7f17f1ae5c00", size = 882114, upload-time = "2025-09-25T21:32:03.376Z" },
    { url = "https://files.pythonhosted.org/packages/71/60/917329f640924b18ff085ab889a11c763e0b573da888e8404ff486657602/pyyaml-6.0.3-cp311-cp311-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl", hash = "sha256:b8bb0864c5a28024fac8a632c443c87c5aa6f215c0b126c449ae1a150412f31d", size = 806638, upload-time = "2025-09-25T21:32:04.553Z" },
    { url = "https://files.pythonhosted.org/packages/dd/6f/529b0f316a9fd167281a6c3826b5583e6192dba792dd55e3203d3f8e655a/pyyaml-6.0.3-cp311-cp311-musllinux_1_2_aarch64.whl", hash = "sha256:1d37d57ad971609cf3c53ba6a7e365e40660e3be0e5175fa9f2365a379d6095a", size = 767463, upload-time = "2025-09-25T21:32:06.152Z" },
    { url = "https://files.pythonhosted.org/packages/f2/6a/b627b4e0c1dd03718543519ffb2f1deea4a1e6d42fbab8021936a4d22589/pyyaml-6.0.3-cp311-cp311-musllinux_1_2_x86_64.whl", hash = "sha256:37503bfbfc9d2c40b344d06b2199cf0e96e97957ab1c1b546fd4f87e53e5d3e4", size = 794986, upload-time = "2025-09-25T21:32:07.367Z" },
    { url = "https://files.pythonhosted.org/packages/45/91/47a6e1c42d9ee337c4839208f30d9f09caa9f720ec7582917b264defc875/pyyaml-6.0.3-cp311-cp311-win32.whl", hash = "sha256:8098f252adfa6c80ab48096053f512f2321f0b998f98150cea9bd23d83e1467b", size = 142543, upload-time = "2025-09-25T21:32:08.95Z" },
    { url = "https://files.pythonhosted.org/packages/da/e3/ea007450a105ae919a72393cb06f122f288ef60bba2dc64b26e2646fa315/pyyaml-6.0.3-cp311-cp311-win_amd64.whl", hash = "sha256:9f3bfb4965eb874431221a3ff3fdcddc7e74e3b07799e0e84ca4a0f867d449bf", size = 158763, upload-time = "2025-09-25T21:32:09.96Z" },
    { url = "https://files.pythonhosted.org/packages/d1/33/422b98d2195232ca1826284a76852ad5a86fe23e31b009c9886b2d0fb8b2/pyyaml-6.0.3-cp312-cp312-macosx_10_13_x86_64.whl", hash = "sha256:7f047e29dcae44602496db43be01ad42fc6f1cc0d8cd6c83d342306c32270196", size = 182063, upload-time = "2025-09-25T21:32:11.445Z" },
    { url = "https://files.pythonhosted.org/packages/89/a0/6cf41a19a1f2f3feab0e9c0b74134aa2ce6849093d5517a0c550fe37a648/pyyaml-6.0.3-cp312-cp312-macosx_11_0_arm64.whl", hash = "sha256:fc09d0aa354569bc501d4e787133afc08552722d3ab34836a80547331bb5d4a0", size = 173973, upload-time = "2025-09-25T21:32:12.492Z" },
    { url = "https://files.pythonhosted.org/packages/ed/23/7a778b6bd0b9a8039df8b1b1d80e2e2ad78aa04171592c8a5c43a56a6af4/pyyaml-6.0.3-cp312-cp312-manylinux2014_aarch64.manylinux_2_17_aarch64.manylinux_2_28_aarch64.whl", hash = "sha256:9149cad251584d5fb4981be1ecde53a1ca46c891a79788c0df828d2f166bda28", size = 775116, upload-time = "2025-09-25T21:32:13.652Z" },
    { url = "https://files.pythonhosted.org/packages/65/30/d7353c338e12baef4ecc1b09e877c1970bd3382789c159b4f89d6a70dc09/pyyaml-6.0.3-cp312-cp312-manylinux2014_s390x.manylinux_2_17_s390x.manylinux_2_28_s390x.whl", hash = "sha256:5fdec68f91a0c6739b380c83b951e2c72ac0197ace422360e6d5a959d8d97b2c", size = 844011, upload-time = "2025-09-25T21:32:15.21Z" },
    { url = "https://files.pythonhosted.org/packages/8b/9d/b3589d3877982d4f2329302ef98a8026e7f4443c765c46cfecc8858c6b4b/pyyaml-6.0.3-cp312-cp312-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl", hash = "sha256:ba1cc08a7ccde2d2ec775841541641e4548226580ab850948cbfda66a1befcdc", size = 807870, upload-time = "2025-09-25T21:32:16.431Z" },
    { url = "https://files.pythonhosted.org/packages/05/c0/b3be26a015601b822b97d9149ff8cb5ead58c66f981e04fedf4e762f4bd4/pyyaml-6.0.3-cp312-cp312-musllinux_1_2_aarch64.whl", hash = "sha256:8dc52c23056b9ddd46818a57b78404882310fb473d63f17b07d5c40421e47f8e", size = 761089, upload-time = "2025-09-25T21:32:17.56Z" },
    { url = "https://files.pythonhosted.org/packages/be/8e/98435a21d1d4b46590d5459a22d88128103f8da4c2d4cb8f14f2a96504e1/pyyaml-6.0.3-cp312-cp312-musllinux_1_2_x86_64.whl", hash = "sha256:41715c910c881bc081f1e8872880d3c650acf13dfa8214bad49ed4cede7c34ea", size = 790181, upload-time = "2025-09-25T21:32:18.834Z" },
    { url = "https://files.pythonhosted.org/packages/74/93/7baea19427dcfbe1e5a372d81473250b379f04b1bd3c4c5ff825e2327202/pyyaml-6.0.3-cp312-cp312-win32.whl", hash = "sha256:96b533f0e99f6579b3d4d4995707cf36df9100d67e0c8303a0c55b27b5f99bc5", size = 137658, upload-time = "2025-09-25T21:32:20.209Z" },
    { url = "https://files.pythonhosted.org/packages/86/bf/899e81e4cce32febab4fb42bb97dcdf66bc135272882d1987881a4b519e9/pyyaml-6.0.3-cp312-cp312-win_amd64.whl", hash = "sha256:5fcd34e47f6e0b794d17de1b4ff496c00986e1c83f7ab2fb8fcfe9616ff7477b", size = 154003, upload-time = "2025-09-25T21:32:21.167Z" },
    { url = "https://files.pythonhosted.org/packages/1a/08/67bd04656199bbb51dbed1439b7f27601dfb576fb864099c7ef0c3e55531/pyyaml-6.0.3-cp312-cp312-win_arm64.whl", hash = "sha256:64386e5e707d03a7e172c0701abfb7e10f0fb753ee1d773128192742712a98fd", size = 140344, upload-time = "2025-09-25T21:32:22.617Z" },
    { url = "https://files.pythonhosted.org/packages/d1/11/0fd08f8192109f7169db964b5707a2f1e8b745d4e239b784a5a1dd80d1db/pyyaml-6.0.3-cp313-cp313-macosx_10_13_x86_64.whl", hash = "sha256:8da9669d359f02c0b91ccc01cac4a67f16afec0dac22c2ad09f46bee0697eba8", size = 181669, upload-time = "2025-09-25T21:32:23.673Z" },
    { url = "https://files.pythonhosted.org/packages/b1/16/95309993f1d3748cd644e02e38b75d50cbc0d9561d21f390a76242ce073f/pyyaml-6.0.3-cp313-cp313-macosx_11_0_arm64.whl", hash = "sha256:2283a07e2c21a2aa78d9c4442724ec1eb15f5e42a723b99cb3d822d48f5f7ad1", size = 173252, upload-time = "2025-09-25T21:32:25.149Z" },
    { url = "https://files.pythonhosted.org/packages/50/31/b20f376d3f810b9b2371e72ef5adb33879b25edb7a6d072cb7ca0c486398/pyyaml-6.0.3-cp313-cp313-manylinux2014_aarch64.manylinux_2_17_aarch64.manylinux_2_28_aarch64.whl", hash = "sha256:ee2922902c45ae8ccada2c5b501ab86c36525b883eff4255313a253a3160861c", size = 767081, upload-time = "2025-09-25T21:32:26.575Z" },
    { url = "https://files.pythonhosted.org/packages/49/1e/a55ca81e949270d5d4432fbbd19dfea5321eda7c41a849d443dc92fd1ff7/pyyaml-6.0.3-cp313-cp313-manylinux2014_s390x.manylinux_2_17_s390x.manylinux_2_28_s390x.whl", hash = "sha256:a33284e20b78bd4a18c8c2282d549d10bc8408a2a7ff57653c0cf0b9be0afce5", size = 841159, upload-time = "2025-09-25T21:32:27.727Z" },
    { url = "https://files.pythonhosted.org/packages/74/27/e5b8f34d02d9995b80abcef563ea1f8b56d20134d8f4e5e81733b1feceb2/pyyaml-6.0.3-cp313-cp313-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl", hash = "sha256:0f29edc409a6392443abf94b9cf89ce99889a1dd5376d94316ae5145dfedd5d6", size = 801626, upload-time = "2025-09-25T21:32:28.878Z" },
    { url = "https://files.pythonhosted.org/packages/f9/11/ba845c23988798f40e52ba45f34849aa8a1f2d4af4b798588010792ebad6/pyyaml-6.0.3-cp313-cp313-musllinux_1_2_aarch64.whl", hash = "sha256:f7057c9a337546edc7973c0d3ba84ddcdf0daa14533c2065749c9075001090e6", size = 753613, upload-time = "2025-09-25T21:32:30.178Z" },
    { url = "https://files.pythonhosted.org/packages/3d/e0/7966e1a7bfc0a45bf0a7fb6b98ea03fc9b8d84fa7f2229e9659680b69ee3/pyyaml-6.0.3-cp313-cp313-musllinux_1_2_x86_64.whl", hash = "sha256:eda16858a3cab07b80edaf74336ece1f986ba330fdb8ee0d6c0d68fe82bc96be", size = 794115, upload-time = "2025-09-25T21:32:31.353Z" },
    { url = "https://files.pythonhosted.org/packages/de/94/980b50a6531b3019e45ddeada0626d45fa85cbe22300844a7983285bed3b/pyyaml-6.0.3-cp313-cp313-win32.whl", hash = "sha256:d0eae10f8159e8fdad514efdc92d74fd8d682c933a6dd088030f3834bc8e6b26", size = 137427, upload-time = "2025-09-25T21:32:32.58Z" },
    { url = "https://files.pythonhosted.org/packages/97/c9/39d5b874e8b28845e4ec2202b5da735d0199dbe5b8fb85f91398814a9a46/pyyaml-6.0.3-cp313-cp313-win_amd64.whl", hash = "sha256:79005a0d97d5ddabfeeea4cf676af11e647e41d81c9a7722a193022accdb6b7c", size = 154090, upload-time = "2025-09-25T21:32:33.659Z" },
    { url = "https://files.pythonhosted.org/packages/73/e8/2bdf3ca2090f68bb3d75b44da7bbc71843b19c9f2b9cb9b0f4ab7a5a4329/pyyaml-6.0.3-cp313-cp313-win_arm64.whl", hash = "sha256:5498cd1645aa724a7c71c8f378eb29ebe23da2fc0d7a08071d89469bf1d2defb", size = 140246, upload-time = "2025-09-25T21:32:34.663Z" },
    { url = "https://files.pythonhosted.org/packages/9d/8c/f4bd7f6465179953d3ac9bc44ac1a8a3e6122cf8ada906b4f96c60172d43/pyyaml-6.0.3-cp314-cp314-macosx_10_13_x86_64.whl", hash = "sha256:8d1fab6bb153a416f9aeb4b8763bc0f22a5586065f86f7664fc23339fc1c1fac", size = 181814, upload-time = "2025-09-25T21:32:35.712Z" },
    { url = "https://files.pythonhosted.org/packages/bd/9c/4d95bb87eb2063d20db7b60faa3840c1b18025517ae857371c4dd55a6b3a/pyyaml-6.0.3-cp314-cp314-macosx_11_0_arm64.whl", hash = "sha256:34d5fcd24b8445fadc33f9cf348c1047101756fd760b4dacb5c3e99755703310", size = 173809, upload-time = "2025-09-25T21:32:36.789Z" },
    { url = "https://files.pythonhosted.org/packages/92/b5/47e807c2623074914e29dabd16cbbdd4bf5e9b2db9f8090fa64411fc5382/pyyaml-6.0.3-cp314-cp314-manylinux2014_aarch64.manylinux_2_17_aarch64.manylinux_2_28_aarch64.whl", hash = "sha256:501a031947e3a9025ed4405a168e6ef5ae3126c59f90ce0cd6f2bfc477be31b7", size = 766454, upload-time = "2025-09-25T21:32:37.966Z" },
    { url = "https://files.pythonhosted.org/packages/02/9e/e5e9b168be58564121efb3de6859c452fccde0ab093d8438905899a3a483/pyyaml-6.0.3-cp314-cp314-manylinux2014_s390x.manylinux_2_17_s390x.manylinux_2_28_s390x.whl", hash = "sha256:b3bc83488de33889877a0f2543ade9f70c67d66d9ebb4ac959502e12de895788", size = 836355, upload-time = "2025-09-25T21:32:39.178Z" },
    { url = "https://files.pythonhosted.org/packages/88/f9/16491d7ed2a919954993e48aa941b200f38040928474c9e85ea9e64222c3/pyyaml-6.0.3-cp314-cp314-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl", hash = "sha256:c458b6d084f9b935061bc36216e8a69a7e293a2f1e68bf956dcd9e6cbcd143f5", size = 794175, upload-time = "2025-09-25T21:32:40.865Z" },
    { url = "https://files.pythonhosted.org/packages/dd/3f/5989debef34dc6397317802b527dbbafb2b4760878a53d4166579111411e/pyyaml-6.0.3-cp314-cp314-musllinux_1_2_aarch64.whl", hash = "sha256:7c6610def4f163542a622a73fb39f534f8c101d690126992300bf3207eab9764", size = 755228, upload-time = "2025-09-25T21:32:42.084Z" },
    { url = "https://files.pythonhosted.org/packages/d7/ce/af88a49043cd2e265be63d083fc75b27b6ed062f5f9fd6cdc223ad62f03e/pyyaml-6.0.3-cp314-cp314-musllinux_1_2_x86_64.whl", hash = "sha256:5190d403f121660ce8d1d2c1bb2ef1bd05b5f68533fc5c2ea899bd15f4399b35", size = 789194, upload-time = "2025-09-25T21:32:43.362Z" },
    { url = "https://files.pythonhosted.org/packages/23/20/bb6982b26a40bb43951265ba29d4c246ef0ff59c9fdcdf0ed04e0687de4d/pyyaml-6.0.3-cp314-cp314-win_amd64.whl", hash = "sha256:4a2e8cebe2ff6ab7d1050ecd59c25d4c8bd7e6f400f5f82b96557ac0abafd0ac", size = 156429, upload-time = "2025-09-25T21:32:57.844Z" },
    { url = "https://files.pythonhosted.org/packages/f4/f4/a4541072bb9422c8a883ab55255f918fa378ecf083f5b85e87fc2b4eda1b/pyyaml-6.0.3-cp314-cp314-win_arm64.whl", hash = "sha256:93dda82c9c22deb0a405ea4dc5f2d0cda384168e466364dec6255b293923b2f3", size = 143912, upload-time = "2025-09-25T21:32:59.247Z" },
    { url = "https://files.pythonhosted.org/packages/7c/f9/07dd09ae774e4616edf6cda684ee78f97777bdd15847253637a6f052a62f/pyyaml-6.0.3-cp314-cp314t-macosx_10_13_x86_64.whl", hash = "sha256:02893d100e99e03eda1c8fd5c441d8c60103fd175728e23e431db1b589cf5ab3", size = 189108, upload-time = "2025-09-25T21:32:44.377Z" },
    { url = "https://files.pythonhosted.org/packages/4e/78/8d08c9fb7ce09ad8c38ad533c1191cf27f7ae1effe5bb9400a46d9437fcf/pyyaml-6.0.3-cp314-cp314t-macosx_11_0_arm64.whl", hash = "sha256:c1ff362665ae507275af2853520967820d9124984e0f7466736aea23d8611fba", size = 183641, upload-time = "2025-09-25T21:32:45.407Z" },
    { url = "https://files.pythonhosted.org/packages/7b/5b/3babb19104a46945cf816d047db2788bcaf8c94527a805610b0289a01c6b/pyyaml-6.0.3-cp314-cp314t-manylinux2014_aarch64.manylinux_2_17_aarch64.manylinux_2_28_aarch64.whl", hash = "sha256:6adc77889b628398debc7b65c073bcb99c4a0237b248cacaf3fe8a557563ef6c", size = 831901, upload-time = "2025-09-25T21:32:48.83Z" },
    { url = "https://files.pythonhosted.org/packages/8b/cc/dff0684d8dc44da4d22a13f35f073d558c268780ce3c6ba1b87055bb0b87/pyyaml-6.0.3-cp314-cp314t-manylinux2014_s390x.manylinux_2_17_s390x.manylinux_2_28_s390x.whl", hash = "sha256:a80cb027f6b349846a3bf6d73b5e95e782175e52f22108cfa17876aaeff93702", size = 861132, upload-time = "2025-09-25T21:32:50.149Z" },
    { url = "https://files.pythonhosted.org/packages/b1/5e/f77dc6b9036943e285ba76b49e118d9ea929885becb0a29ba8a7c75e29fe/pyyaml-6.0.3-cp314-cp314t-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl", hash = "sha256:00c4bdeba853cc34e7dd471f16b4114f4162dc03e6b7afcc2128711f0eca823c", size = 839261, upload-time = "2025-09-25T21:32:51.808Z" },
    { url = "https://files.pythonhosted.org/packages/ce/88/a9db1376aa2a228197c58b37302f284b5617f56a5d959fd1763fb1675ce6/pyyaml-6.0.3-cp314-cp314t-musllinux_1_2_aarch64.whl", hash = "sha256:66e1674c3ef6f541c35191caae2d429b967b99e02040f5ba928632d9a7f0f065", size = 805272, upload-time = "2025-09-25T21:32:52.941Z" },
    { url = "https://files.pythonhosted.org/packages/da/92/1446574745d74df0c92e6aa4a7b0b3130706a4142b2d1a5869f2eaa423c6/pyyaml-6.0.3-cp314-cp314t-musllinux_1_2_x86_64.whl", hash = "sha256:16249ee61e95f858e83976573de0f5b2893b3677ba71c9dd36b9cf8be9ac6d65", size = 829923, upload-time = "2025-09-25T21:32:54.537Z" },
    { url = "https://files.pythonhosted.org/packages/f0/7a/1c7270340330e575b92f397352af856a8c06f230aa3e76f86b39d01b416a/pyyaml-6.0.3-cp314-cp314t-win_amd64.whl", hash = "sha256:4ad1906908f2f5ae4e5a8ddfce73c320c2a1429ec52eafd27138b7f1cbe341c9", size = 174062, upload-time = "2025-09-25T21:32:55.767Z" },
    { url = "https://files.pythonhosted.org/packages/f1/12/de94a39c2ef588c7e6455cfbe7343d3b2dc9d6b6b2f40c4c6565744c873d/pyyaml-6.0.3-cp314-cp314t-win_arm64.whl", hash = "sha256:ebc55a14a21cb14062aa4162f906cd962b28e2e9ea38f9b4391244cd8de4ae0b", size = 149341, upload-time = "2025-09-25T21:32:56.828Z" },
]

[[package]]
name = "tmp"
version = "0.1.0"
source = { virtual = "." }
dependencies = [
    { name = "dnspython" },
]

[package.dev-dependencies]
dev = [
    { name = "pre-commit" },
]

[package.metadata]
requires-dist = [{ name = "dnspython", specifier = ">=2.8.0" }]

[package.metadata.requires-dev]
dev = [{ name = "pre-commit", specifier = ">=4.3.0" }]

[[package]]
name = "virtualenv"
version = "20.35.3"
source = { registry = "https://pypidev.ivrtechnology.com/" }
dependencies = [
    { name = "distlib" },
    { name = "filelock" },
    { name = "platformdirs" },
]
sdist = { url = "https://files.pythonhosted.org/packages/a4/d5/b0ccd381d55c8f45d46f77df6ae59fbc23d19e901e2d523395598e5f4c93/virtualenv-20.35.3.tar.gz", hash = "sha256:4f1a845d131133bdff10590489610c98c168ff99dc75d6c96853801f7f67af44", size = 6002907, upload-time = "2025-10-10T21:23:33.178Z" }
wheels = [
    { url = "https://files.pythonhosted.org/packages/27/73/d9a94da0e9d470a543c1b9d3ccbceb0f59455983088e727b8a1824ed90fb/virtualenv-20.35.3-py3-none-any.whl", hash = "sha256:63d106565078d8c8d0b206d48080f938a8b25361e19432d2c9db40d2899c810a", size = 5981061, upload-time = "2025-10-10T21:23:30.433Z" },
]
`

	mockClient := &mockRepoClient{
		files: []repository.FileInfo{
			{Path: "uv.lock", Type: "file"},
		},
		content: content,
	}

	analyzer := NewUvLockAnalyzer()
	config := Config{
		RepositoryPaths:  []string{""},
		RepositoryClient: mockClient,
	}

	ctx := context.Background()

	// Find candidate files
	candidates, err := analyzer.CandidateFiles(ctx, "test-owner", "test-repo", "main", config)
	if err != nil {
		t.Fatalf("Failed to find candidate files: %v", err)
	}

	if len(candidates) != 1 {
		t.Errorf("Expected 1 candidate file, got %d", len(candidates))
	}

	// Analyze dependencies
	results, err := analyzer.AnalyzeDependencies(ctx, "test-owner", "test-repo", "main", candidates, config)
	if err != nil {
		t.Fatalf("Failed to analyze dependencies: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected results for 1 file, got %d", len(results))
	}

	deps := results["uv.lock"]
	if len(deps) == 0 {
		t.Fatal("Expected dependencies, got none")
	}

	t.Logf("Successfully analyzed real uv.lock file with %d dependencies", len(deps))

	// Verify expected packages
	expectedPackages := []string{"cfgv", "distlib", "dnspython", "filelock", "identify", "virtualenv", "tmp"}
	foundPackages := make(map[string]bool)

	for _, dep := range deps {
		foundPackages[dep.Name] = true
	}

	for _, pkg := range expectedPackages {
		if !foundPackages[pkg] {
			t.Errorf("Expected package %s not found in analysis results", pkg)
		}
	}

	// Verify dependency types
	runtimeCount := 0
	devCount := 0
	for _, dep := range deps {
		switch dep.Type {
		case "runtime":
			runtimeCount++
		case "dev":
			devCount++
		}
	}

	t.Logf("Analysis results: %d runtime, %d dev dependencies", runtimeCount, devCount)

	if runtimeCount < 5 {
		t.Errorf("Expected at least 5 runtime dependencies, got %d", runtimeCount)
	}

	if devCount < 1 {
		t.Errorf("Expected at least 1 dev dependency, got %d", devCount)
	}

	// Verify the virtual package with dev-dependencies is correctly identified
	var tmpPkg *Dependency
	for i := range deps {
		if deps[i].Name == "tmp" {
			tmpPkg = &deps[i]
			break
		}
	}

	if tmpPkg == nil {
		t.Error("Expected to find 'tmp' package (virtual package with dev-dependencies)")
	} else if tmpPkg.Type != "dev" {
		t.Errorf("Package 'tmp' should be type 'dev', got '%s'", tmpPkg.Type)
	}
}
