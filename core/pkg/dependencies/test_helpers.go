package dependencies

import (
	"context"

	"github.com/greg-hellings/devdashboard/core/pkg/repository"
)

// mockRepoClient is a mock implementation of repository.Client for testing
type mockRepoClient struct {
	files   []repository.FileInfo
	content string
	err     error
}

func (m *mockRepoClient) GetRepositoryInfo(_ context.Context, owner, repo string) (*repository.Info, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &repository.Info{
		ID:            "test-repo",
		Name:          repo,
		FullName:      owner + "/" + repo,
		Description:   "Test repository",
		DefaultBranch: "main",
		URL:           "https://example.com/" + owner + "/" + repo,
	}, nil
}

func (m *mockRepoClient) ListFiles(_ context.Context, _, _, _, _ string) ([]repository.FileInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.files, nil
}

func (m *mockRepoClient) ListFilesRecursive(_ context.Context, _, _, _ string) ([]repository.FileInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.files, nil
}

func (m *mockRepoClient) GetFileContent(_ context.Context, _, _, _, _ string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.content, nil
}
