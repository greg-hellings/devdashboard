package report

import (
	"context"
	"errors"
	"testing"

	"github.com/greg-hellings/devdashboard/pkg/config"
	"github.com/greg-hellings/devdashboard/pkg/repository"
)

// mockRepoClient is a mock repository client for testing
type mockRepoClient struct {
	files       map[string][]repository.FileInfo
	fileContent map[string]string
	shouldError bool
}

func (m *mockRepoClient) ListFiles(ctx context.Context, owner, repo, ref, path string) ([]repository.FileInfo, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	key := owner + "/" + repo + "/" + path
	if files, ok := m.files[key]; ok {
		return files, nil
	}
	return []repository.FileInfo{}, nil
}

func (m *mockRepoClient) ListFilesRecursive(ctx context.Context, owner, repo, ref string) ([]repository.FileInfo, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	return m.files[owner+"/"+repo+"/"], nil
}

func (m *mockRepoClient) GetFileContent(ctx context.Context, owner, repo, ref, path string) (string, error) {
	if m.shouldError {
		return "", errors.New("mock error")
	}
	if content, ok := m.fileContent[path]; ok {
		return content, nil
	}
	return "", errors.New("file not found")
}

func (m *mockRepoClient) GetRepositoryInfo(ctx context.Context, owner, repo string) (*repository.RepositoryInfo, error) {
	if m.shouldError {
		return nil, errors.New("mock error")
	}
	return &repository.RepositoryInfo{
		ID:            "test-repo",
		Name:          repo,
		FullName:      owner + "/" + repo,
		Description:   "Test repository",
		DefaultBranch: "main",
		URL:           "https://example.com/" + owner + "/" + repo,
	}, nil
}

func TestNewGenerator(t *testing.T) {
	gen := NewGenerator()
	if gen == nil {
		t.Fatal("NewGenerator returned nil")
	}
	if gen.depFactory == nil {
		t.Error("Generator should have a dependency factory")
	}
}

func TestGenerate_EmptyRepos(t *testing.T) {
	gen := NewGenerator()
	ctx := context.Background()

	report, err := gen.Generate(ctx, []config.RepoWithProvider{})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if report == nil {
		t.Fatal("Report should not be nil")
	}

	if len(report.Repositories) != 0 {
		t.Errorf("Expected 0 repositories, got %d", len(report.Repositories))
	}

	if len(report.Packages) != 0 {
		t.Errorf("Expected 0 packages, got %d", len(report.Packages))
	}
}

func TestGenerate_WithPackages(t *testing.T) {
	gen := NewGenerator()
	ctx := context.Background()

	repos := []config.RepoWithProvider{
		{
			Provider: "github",
			Config: config.RepoConfig{
				Owner:      "test-owner",
				Repository: "test-repo",
				Ref:        "main",
				Analyzer:   "pipfile",
				Packages:   []string{"django", "requests"},
				Paths:      []string{"Pipfile.lock"},
			},
		},
	}

	report, err := gen.Generate(ctx, repos)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(report.Packages) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(report.Packages))
	}

	// Packages should be sorted
	if len(report.Packages) >= 2 {
		if report.Packages[0] != "django" || report.Packages[1] != "requests" {
			t.Errorf("Expected packages to be sorted: django, requests; got %v", report.Packages)
		}
	}

	if len(report.Repositories) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(report.Repositories))
	}

	repoReport := report.Repositories[0]
	if repoReport.Provider != "github" {
		t.Errorf("Expected provider 'github', got '%s'", repoReport.Provider)
	}
	if repoReport.Owner != "test-owner" {
		t.Errorf("Expected owner 'test-owner', got '%s'", repoReport.Owner)
	}
	if repoReport.Repository != "test-repo" {
		t.Errorf("Expected repository 'test-repo', got '%s'", repoReport.Repository)
	}
}

func TestGenerate_MultipleRepos(t *testing.T) {
	gen := NewGenerator()
	ctx := context.Background()

	repos := []config.RepoWithProvider{
		{
			Provider: "github",
			Config: config.RepoConfig{
				Owner:      "owner1",
				Repository: "repo1",
				Ref:        "main",
				Analyzer:   "pipfile",
				Packages:   []string{"django"},
				Paths:      []string{"Pipfile.lock"},
			},
		},
		{
			Provider: "gitlab",
			Config: config.RepoConfig{
				Owner:      "owner2",
				Repository: "repo2",
				Ref:        "main",
				Analyzer:   "poetry",
				Packages:   []string{"requests"},
				Paths:      []string{"poetry.lock"},
			},
		},
	}

	report, err := gen.Generate(ctx, repos)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(report.Repositories) != 2 {
		t.Fatalf("Expected 2 repositories, got %d", len(report.Repositories))
	}

	// Check both repositories were analyzed
	if report.Repositories[0].Owner != "owner1" {
		t.Errorf("Expected first repo owner 'owner1', got '%s'", report.Repositories[0].Owner)
	}
	if report.Repositories[1].Owner != "owner2" {
		t.Errorf("Expected second repo owner 'owner2', got '%s'", report.Repositories[1].Owner)
	}
}

func TestGenerate_InvalidProvider(t *testing.T) {
	gen := NewGenerator()
	ctx := context.Background()

	repos := []config.RepoWithProvider{
		{
			Provider: "invalid-provider",
			Config: config.RepoConfig{
				Owner:      "test-owner",
				Repository: "test-repo",
				Ref:        "main",
				Analyzer:   "pipfile",
				Packages:   []string{"django"},
			},
		},
	}

	report, err := gen.Generate(ctx, repos)
	if err != nil {
		t.Fatalf("Generate should not fail on invalid provider: %v", err)
	}

	if len(report.Repositories) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(report.Repositories))
	}

	if report.Repositories[0].Error == nil {
		t.Error("Expected error for invalid provider")
	}
}

func TestGenerate_InvalidAnalyzer(t *testing.T) {
	gen := NewGenerator()
	ctx := context.Background()

	repos := []config.RepoWithProvider{
		{
			Provider: "github",
			Config: config.RepoConfig{
				Owner:      "test-owner",
				Repository: "test-repo",
				Ref:        "main",
				Analyzer:   "invalid-analyzer",
				Packages:   []string{"django"},
			},
		},
	}

	report, err := gen.Generate(ctx, repos)
	if err != nil {
		t.Fatalf("Generate should not fail on invalid analyzer: %v", err)
	}

	if len(report.Repositories) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(report.Repositories))
	}

	if report.Repositories[0].Error == nil {
		t.Error("Expected error for invalid analyzer")
	}
}

func TestGetPackageVersions_NoPackages(t *testing.T) {
	report := &Report{
		Packages:     []string{},
		Repositories: []RepositoryReport{},
	}

	versions := report.GetPackageVersions()
	if len(versions) != 0 {
		t.Errorf("Expected 0 package versions, got %d", len(versions))
	}
}

func TestGetPackageVersions_SinglePackage(t *testing.T) {
	report := &Report{
		Packages: []string{"django"},
		Repositories: []RepositoryReport{
			{
				Owner:      "owner1",
				Repository: "repo1",
				Dependencies: map[string]string{
					"django": "4.2.0",
				},
			},
			{
				Owner:      "owner2",
				Repository: "repo2",
				Dependencies: map[string]string{
					"django": "3.2.0",
				},
			},
		},
	}

	versions := report.GetPackageVersions()
	if len(versions) != 1 {
		t.Fatalf("Expected 1 package version, got %d", len(versions))
	}

	pv := versions[0]
	if pv.PackageName != "django" {
		t.Errorf("Expected package name 'django', got '%s'", pv.PackageName)
	}

	if len(pv.Versions) != 2 {
		t.Errorf("Expected 2 different versions, got %d", len(pv.Versions))
	}

	if len(pv.Versions["4.2.0"]) != 1 {
		t.Errorf("Expected 1 repo with version 4.2.0, got %d", len(pv.Versions["4.2.0"]))
	}

	if len(pv.Versions["3.2.0"]) != 1 {
		t.Errorf("Expected 1 repo with version 3.2.0, got %d", len(pv.Versions["3.2.0"]))
	}
}

func TestGetPackageVersions_PackageNotFound(t *testing.T) {
	report := &Report{
		Packages: []string{"django"},
		Repositories: []RepositoryReport{
			{
				Owner:      "owner1",
				Repository: "repo1",
				Dependencies: map[string]string{
					"django": "4.2.0",
				},
			},
			{
				Owner:        "owner2",
				Repository:   "repo2",
				Dependencies: map[string]string{
					// django not present
				},
			},
		},
	}

	versions := report.GetPackageVersions()
	if len(versions) != 1 {
		t.Fatalf("Expected 1 package version, got %d", len(versions))
	}

	pv := versions[0]
	if pv.PackageName != "django" {
		t.Errorf("Expected package name 'django', got '%s'", pv.PackageName)
	}

	// Should have version 4.2.0 and empty string for not found
	if len(pv.Versions) != 2 {
		t.Errorf("Expected 2 version entries (including empty), got %d", len(pv.Versions))
	}

	if len(pv.Versions[""]) != 1 {
		t.Errorf("Expected 1 repo with missing package, got %d", len(pv.Versions[""]))
	}

	if pv.Versions[""][0] != "owner2/repo2" {
		t.Errorf("Expected owner2/repo2 in missing packages, got %s", pv.Versions[""][0])
	}
}

func TestGetPackageVersions_MultipleReposSameVersion(t *testing.T) {
	report := &Report{
		Packages: []string{"requests"},
		Repositories: []RepositoryReport{
			{
				Owner:      "owner1",
				Repository: "repo1",
				Dependencies: map[string]string{
					"requests": "2.28.0",
				},
			},
			{
				Owner:      "owner2",
				Repository: "repo2",
				Dependencies: map[string]string{
					"requests": "2.28.0",
				},
			},
			{
				Owner:      "owner3",
				Repository: "repo3",
				Dependencies: map[string]string{
					"requests": "2.28.0",
				},
			},
		},
	}

	versions := report.GetPackageVersions()
	if len(versions) != 1 {
		t.Fatalf("Expected 1 package version, got %d", len(versions))
	}

	pv := versions[0]
	if len(pv.Versions) != 1 {
		t.Errorf("Expected 1 version, got %d", len(pv.Versions))
	}

	if len(pv.Versions["2.28.0"]) != 3 {
		t.Errorf("Expected 3 repos with version 2.28.0, got %d", len(pv.Versions["2.28.0"]))
	}
}

func TestGetRepoIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		repo     RepositoryReport
		expected string
	}{
		{
			name: "simple identifier",
			repo: RepositoryReport{
				Owner:      "myorg",
				Repository: "myrepo",
			},
			expected: "myorg/myrepo",
		},
		{
			name: "with special characters",
			repo: RepositoryReport{
				Owner:      "my-org",
				Repository: "my.repo",
			},
			expected: "my-org/my.repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.repo.GetRepoIdentifier()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestHasErrors(t *testing.T) {
	tests := []struct {
		name     string
		report   *Report
		expected bool
	}{
		{
			name: "no errors",
			report: &Report{
				Repositories: []RepositoryReport{
					{Owner: "owner1", Repository: "repo1", Error: nil},
					{Owner: "owner2", Repository: "repo2", Error: nil},
				},
			},
			expected: false,
		},
		{
			name: "one error",
			report: &Report{
				Repositories: []RepositoryReport{
					{Owner: "owner1", Repository: "repo1", Error: nil},
					{Owner: "owner2", Repository: "repo2", Error: errors.New("test error")},
				},
			},
			expected: true,
		},
		{
			name: "all errors",
			report: &Report{
				Repositories: []RepositoryReport{
					{Owner: "owner1", Repository: "repo1", Error: errors.New("error1")},
					{Owner: "owner2", Repository: "repo2", Error: errors.New("error2")},
				},
			},
			expected: true,
		},
		{
			name: "empty repositories",
			report: &Report{
				Repositories: []RepositoryReport{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.report.HasErrors()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetErrors(t *testing.T) {
	tests := []struct {
		name          string
		report        *Report
		expectedCount int
	}{
		{
			name: "no errors",
			report: &Report{
				Repositories: []RepositoryReport{
					{Owner: "owner1", Repository: "repo1", Error: nil},
					{Owner: "owner2", Repository: "repo2", Error: nil},
				},
			},
			expectedCount: 0,
		},
		{
			name: "one error",
			report: &Report{
				Repositories: []RepositoryReport{
					{Owner: "owner1", Repository: "repo1", Error: nil},
					{Owner: "owner2", Repository: "repo2", Error: errors.New("test error")},
				},
			},
			expectedCount: 1,
		},
		{
			name: "multiple errors",
			report: &Report{
				Repositories: []RepositoryReport{
					{Owner: "owner1", Repository: "repo1", Error: errors.New("error1")},
					{Owner: "owner2", Repository: "repo2", Error: errors.New("error2")},
					{Owner: "owner3", Repository: "repo3", Error: errors.New("error3")},
				},
			},
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.report.GetErrors()
			if len(errors) != tt.expectedCount {
				t.Errorf("Expected %d errors, got %d", tt.expectedCount, len(errors))
			}

			// Verify error keys match repository identifiers
			for _, repo := range tt.report.Repositories {
				if repo.Error != nil {
					key := repo.GetRepoIdentifier()
					if _, found := errors[key]; !found {
						t.Errorf("Expected error for repository %s", key)
					}
				}
			}
		})
	}
}

func TestGetErrors_VerifyErrorContent(t *testing.T) {
	expectedError := errors.New("specific error message")
	report := &Report{
		Repositories: []RepositoryReport{
			{
				Owner:      "test-owner",
				Repository: "test-repo",
				Error:      expectedError,
			},
		},
	}

	errors := report.GetErrors()
	if len(errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errors))
	}

	key := "test-owner/test-repo"
	if err, found := errors[key]; !found {
		t.Errorf("Expected error for key %s", key)
	} else if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}
}

func TestRepositoryReport_Structure(t *testing.T) {
	report := RepositoryReport{
		Provider:   "github",
		Owner:      "test-owner",
		Repository: "test-repo",
		Ref:        "main",
		Analyzer:   "poetry",
		Dependencies: map[string]string{
			"package1": "1.0.0",
			"package2": "2.0.0",
		},
		Error: nil,
	}

	if report.Provider != "github" {
		t.Errorf("Expected provider 'github', got '%s'", report.Provider)
	}

	if len(report.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(report.Dependencies))
	}

	if report.Dependencies["package1"] != "1.0.0" {
		t.Errorf("Expected package1 version '1.0.0', got '%s'", report.Dependencies["package1"])
	}
}
