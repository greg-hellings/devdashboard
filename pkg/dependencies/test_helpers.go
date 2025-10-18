package dependencies

import (
	"context"

	"github.com/greg-hellings/devdashboard/pkg/repository"
)

// mockRepoClient is a mock implementation of repository.Client for testing
type mockRepoClient struct {
	files   []repository.FileInfo
	content string
	err     error
}

func (m *mockRepoClient) GetRepositoryInfo(ctx context.Context, owner, repo string) (*repository.RepositoryInfo, error) {
	if m.err != nil {
		return nil, m.err
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

func (m *mockRepoClient) ListFiles(ctx context.Context, owner, repo, ref, path string) ([]repository.FileInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.files, nil
}

func (m *mockRepoClient) ListFilesRecursive(ctx context.Context, owner, repo, ref string) ([]repository.FileInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.files, nil
}

func (m *mockRepoClient) GetFileContent(ctx context.Context, owner, repo, ref, path string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.content, nil
}
