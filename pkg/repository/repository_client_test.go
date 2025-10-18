package repository

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-github/v57/github"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

///////////////////////////////
// GitHub mock implementations
///////////////////////////////

type mockGitHubRepos struct {
	repo         *github.Repository
	dirContents  map[string][]*github.RepositoryContent
	fileContents map[string]*github.RepositoryContent
}

func (m *mockGitHubRepos) Get(_ context.Context, _, _ string) (*github.Repository, *github.Response, error) {
	return m.repo, &github.Response{Response: &http.Response{Body: io.NopCloser(strings.NewReader(""))}}, nil
}

func (m *mockGitHubRepos) GetContents(_ context.Context, _, _, path string, _ *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
	if fc, ok := m.fileContents[path]; ok {
		return fc, nil, &github.Response{Response: &http.Response{Body: io.NopCloser(strings.NewReader(""))}}, nil
	}
	if dc, ok := m.dirContents[path]; ok {
		return nil, dc, &github.Response{Response: &http.Response{Body: io.NopCloser(strings.NewReader(""))}}, nil
	}
	// Default: empty directory
	return nil, []*github.RepositoryContent{}, &github.Response{Response: &http.Response{Body: io.NopCloser(strings.NewReader(""))}}, nil
}

type mockGitHubGit struct {
	tree *github.Tree
}

func (m *mockGitHubGit) GetTree(_ context.Context, _ string, _ string, _ string, _ bool) (*github.Tree, *github.Response, error) {
	return m.tree, &github.Response{Response: &http.Response{Body: io.NopCloser(strings.NewReader(""))}}, nil
}

///////////////////////////////
// GitLab mock implementations
///////////////////////////////

type mockGitLabProjects struct {
	project *gitlab.Project
}

func (m *mockGitLabProjects) GetProject(_ string, _ *gitlab.GetProjectOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error) {
	return m.project, &gitlab.Response{Response: &http.Response{Body: io.NopCloser(strings.NewReader(""))}}, nil
}

type mockGitLabRepos struct {
	pages    map[int][]*gitlab.TreeNode
	nextPage map[int]int
}

func (m *mockGitLabRepos) ListTree(_ string, opts *gitlab.ListTreeOptions, _ ...gitlab.RequestOptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error) {
	page := opts.Page
	nodes := m.pages[page]
	resp := &gitlab.Response{
		Response: &http.Response{Body: io.NopCloser(strings.NewReader(""))},
		NextPage: m.nextPage[page],
	}
	return nodes, resp, nil
}

type mockGitLabFiles struct {
	files map[string]*gitlab.File
}

func (m *mockGitLabFiles) GetFile(_ string, filePath string, _ *gitlab.GetFileOptions, _ ...gitlab.RequestOptionFunc) (*gitlab.File, *gitlab.Response, error) {
	f := m.files[filePath]
	return f, &gitlab.Response{Response: &http.Response{Body: io.NopCloser(strings.NewReader(""))}}, nil
}

///////////////////////////////
// GitHub Client Tests
///////////////////////////////

func TestGitHubListFilesRecursive_DefaultBranchFallback(t *testing.T) {
	// Prepare mock repository metadata with default branch
	mockRepo := &github.Repository{
		ID:            github.Int64(101),
		Name:          github.String("repo"),
		FullName:      github.String("owner/repo"),
		DefaultBranch: github.String("main"),
	}

	// Git tree includes one blob and one tree (directory); only blob should be returned
	tree := &github.Tree{
		Entries: []*github.TreeEntry{
			{
				Type: github.String("blob"),
				Path: github.String("src/file.go"),
				Size: github.Int(42),
				SHA:  github.String("deadbeef"),
				Mode: github.String("100644"),
			},
			{
				Type: github.String("tree"),
				Path: github.String("src/pkg"),
				SHA:  github.String("cafebabe"),
				Mode: github.String("040000"),
			},
		},
	}

	client := &GitHubClient{
		api: GitHubAPI{
			Repositories: &mockGitHubRepos{repo: mockRepo},
			Git:          &mockGitHubGit{tree: tree},
		},
		config: Config{},
	}

	files, err := client.ListFilesRecursive(context.Background(), "owner", "repo", "")
	if err != nil {
		t.Fatalf("ListFilesRecursive error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file (blob), got %d", len(files))
	}
	if files[0].Path != "src/file.go" {
		t.Errorf("Unexpected file path: %s", files[0].Path)
	}
	if files[0].SHA != "deadbeef" {
		t.Errorf("Unexpected SHA: %s", files[0].SHA)
	}
}

func TestGitHubListFiles_DirectoryListing(t *testing.T) {
	dirContents := map[string][]*github.RepositoryContent{
		"": {
			{
				Type:    github.String("file"),
				Path:    github.String("README.md"),
				Name:    github.String("README.md"),
				SHA:     github.String("abc123"),
				Size:    github.Int(128),
				HTMLURL: github.String("https://example.com/README.md"),
			},
			{
				Type: github.String("dir"),
				Path: github.String("docs"),
				Name: github.String("docs"),
				SHA:  github.String("def456"),
			},
		},
	}

	client := &GitHubClient{
		api: GitHubAPI{
			Repositories: &mockGitHubRepos{
				repo:         &github.Repository{DefaultBranch: github.String("main")},
				dirContents:  dirContents,
				fileContents: map[string]*github.RepositoryContent{},
			},
			Git: &mockGitHubGit{},
		},
	}

	files, err := client.ListFiles(context.Background(), "owner", "repo", "main", "")
	if err != nil {
		t.Fatalf("ListFiles error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(files))
	}

	seen := map[string]bool{}
	for _, f := range files {
		seen[f.Name] = true
	}
	if !seen["README.md"] || !seen["docs"] {
		t.Errorf("Missing expected entries in listing: %+v", seen)
	}
}

func TestGitHubGetFileContent_Base64(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("hello world"))
	fileContent := &github.RepositoryContent{
		Type:     github.String("file"),
		Path:     github.String("file.txt"),
		Name:     github.String("file.txt"),
		Content:  github.String(encoded),
		Encoding: github.String("base64"),
	}

	client := &GitHubClient{
		api: GitHubAPI{
			Repositories: &mockGitHubRepos{
				repo:         &github.Repository{DefaultBranch: github.String("main")},
				dirContents:  map[string][]*github.RepositoryContent{},
				fileContents: map[string]*github.RepositoryContent{"file.txt": fileContent},
			},
			Git: &mockGitHubGit{},
		},
	}

	content, err := client.GetFileContent(context.Background(), "owner", "repo", "main", "file.txt")
	if err != nil {
		t.Fatalf("GetFileContent error: %v", err)
	}
	if content != "hello world" {
		t.Errorf("Expected decoded content 'hello world', got '%s'", content)
	}
}

///////////////////////////////
// GitLab Client Tests
///////////////////////////////

func TestGitLabListFilesRecursive_Pagination(t *testing.T) {
	project := &gitlab.Project{
		ID:                500,
		Name:              "gitlab-repo",
		PathWithNamespace: "group/gitlab-repo",
		DefaultBranch:     "main",
	}

	pages := map[int][]*gitlab.TreeNode{
		1: {
			{Type: "blob", Path: "file1.py", Name: "file1.py", Mode: "100644", ID: "sha1"},
			{Type: "tree", Path: "pkg", Name: "pkg", Mode: "040000", ID: "sha2"},
		},
		2: {
			{Type: "blob", Path: "file2.py", Name: "file2.py", Mode: "100644", ID: "sha3"},
		},
	}
	next := map[int]int{1: 2, 2: 0}

	client := &GitLabClient{
		api: GitLabAPI{
			Projects:        &mockGitLabProjects{project: project},
			Repositories:    &mockGitLabRepos{pages: pages, nextPage: next},
			RepositoryFiles: &mockGitLabFiles{files: map[string]*gitlab.File{}},
		},
		config: Config{},
	}

	files, err := client.ListFilesRecursive(context.Background(), "group", "gitlab-repo", "")
	if err != nil {
		t.Fatalf("ListFilesRecursive error: %v", err)
	}

	// Expect only blob entries across pages: file1.py and file2.py
	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(files))
	}

	paths := map[string]bool{}
	for _, f := range files {
		paths[f.Path] = true
	}
	if !paths["file1.py"] || !paths["file2.py"] {
		t.Errorf("Missing expected files: %+v", paths)
	}
}

func TestGitLabGetFileContent_DefaultBranchAndDecode(t *testing.T) {
	project := &gitlab.Project{
		ID:                600,
		Name:              "sample",
		PathWithNamespace: "org/sample",
		DefaultBranch:     "develop",
	}

	encoded := base64.StdEncoding.EncodeToString([]byte("gitlab content"))
	file := &gitlab.File{
		FileName: "info.txt",
		Content:  encoded,
	}

	client := &GitLabClient{
		api: GitLabAPI{
			Projects:        &mockGitLabProjects{project: project},
			Repositories:    &mockGitLabRepos{pages: map[int][]*gitlab.TreeNode{}, nextPage: map[int]int{}},
			RepositoryFiles: &mockGitLabFiles{files: map[string]*gitlab.File{"info.txt": file}},
		},
		config: Config{},
	}

	content, err := client.GetFileContent(context.Background(), "org", "sample", "", "info.txt")
	if err != nil {
		t.Fatalf("GetFileContent error: %v", err)
	}
	if content != "gitlab content" {
		t.Errorf("Expected 'gitlab content', got '%s'", content)
	}
}
