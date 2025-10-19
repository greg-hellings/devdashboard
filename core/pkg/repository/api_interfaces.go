package repository

// This file defines narrow interfaces and lightweight wrappers around the
// external GitHub and GitLab API clients. These abstractions make it possible
// to inject deterministic mock implementations in unit tests without relying
// on real HTTP calls or the full surface area of the third‑party SDKs.
//
// Design goals:
// 1. Minimize the number of methods exposed – only what repository clients use.
// 2. Keep interfaces stable so tests don't break when upstream adds new APIs.
// 3. Provide simple wrapper constructors for production usage.
// 4. Allow custom (mock/fake) implementations in tests for higher coverage.
//
// To use in tests:
//   - Create a struct implementing the needed interface(s).
//   - Inject it into a custom client constructor (future change).
//   - Return controlled data for paths, repository metadata, and trees.
//   - Assert transformation logic (e.g., blob -> file, tree -> dir) without network.
//
// Future refactor (optional):
//   The existing GitHubClient / GitLabClient structs can be updated to accept these
//   interface instances (e.g., via NewGitHubClientWithServices or similar) enabling
//   pure unit tests. For now, these interfaces and wrappers exist so that additional
//   test-friendly constructors can be added incrementally.

import (
	"context"

	"github.com/google/go-github/v57/github"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

/////////////////////////
// GitHub API Interfaces
/////////////////////////

// GitHubRepositoriesService abstracts the subset of repository operations used.
type GitHubRepositoriesService interface {
	// Get fetches metadata for a repository.
	Get(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error)
	// GetContents retrieves either a file OR a directory listing depending on path.
	GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error)
}

// GitHubGitService abstracts git tree traversal used for recursive file listing.
type GitHubGitService interface {
	// GetTree retrieves a git tree; when recursive=true it expands the entire tree.
	GetTree(ctx context.Context, owner, repo, sha string, recursive bool) (*github.Tree, *github.Response, error)
}

// githubRepositoriesWrapper is the production wrapper implementing GitHubRepositoriesService.
type githubRepositoriesWrapper struct {
	client *github.Client
}

func (w *githubRepositoriesWrapper) Get(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error) {
	return w.client.Repositories.Get(ctx, owner, repo)
}

func (w *githubRepositoriesWrapper) GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
	return w.client.Repositories.GetContents(ctx, owner, repo, path, opts)
}

// githubGitWrapper is the production wrapper implementing GitHubGitService.
type githubGitWrapper struct {
	client *github.Client
}

func (w *githubGitWrapper) GetTree(ctx context.Context, owner, repo, sha string, recursive bool) (*github.Tree, *github.Response, error) {
	return w.client.Git.GetTree(ctx, owner, repo, sha, recursive)
}

// GitHubAPI groups the narrowed GitHub service interfaces.
type GitHubAPI struct {
	Repositories GitHubRepositoriesService
	Git          GitHubGitService
}

// wrapGitHubClient constructs GitHubAPI from a *github.Client.
func wrapGitHubClient(c *github.Client) GitHubAPI {
	return GitHubAPI{
		Repositories: &githubRepositoriesWrapper{client: c},
		Git:          &githubGitWrapper{client: c},
	}
}

/////////////////////////
// GitLab API Interfaces
/////////////////////////

// GitLabProjectsService abstracts project metadata retrieval.
type GitLabProjectsService interface {
	GetProject(projectID string, opts *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error)
}

// GitLabRepositoriesService abstracts tree listing operations.
type GitLabRepositoriesService interface {
	ListTree(projectID string, opts *gitlab.ListTreeOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error)
}

// GitLabRepositoryFilesService abstracts file content retrieval.
type GitLabRepositoryFilesService interface {
	GetFile(projectID string, filePath string, opts *gitlab.GetFileOptions, options ...gitlab.RequestOptionFunc) (*gitlab.File, *gitlab.Response, error)
}

// gitlabProjectsWrapper is the production wrapper for project metadata.
type gitlabProjectsWrapper struct {
	client *gitlab.Client
}

func (w *gitlabProjectsWrapper) GetProject(projectID string, opts *gitlab.GetProjectOptions, options ...gitlab.RequestOptionFunc) (*gitlab.Project, *gitlab.Response, error) {
	return w.client.Projects.GetProject(projectID, opts, options...)
}

// gitlabRepositoriesWrapper is the production wrapper for listing repository trees.
type gitlabRepositoriesWrapper struct {
	client *gitlab.Client
}

func (w *gitlabRepositoriesWrapper) ListTree(projectID string, opts *gitlab.ListTreeOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.TreeNode, *gitlab.Response, error) {
	return w.client.Repositories.ListTree(projectID, opts, options...)
}

// gitlabRepositoryFilesWrapper is the production wrapper for file content.
type gitlabRepositoryFilesWrapper struct {
	client *gitlab.Client
}

func (w *gitlabRepositoryFilesWrapper) GetFile(projectID string, filePath string, opts *gitlab.GetFileOptions, options ...gitlab.RequestOptionFunc) (*gitlab.File, *gitlab.Response, error) {
	return w.client.RepositoryFiles.GetFile(projectID, filePath, opts, options...)
}

// GitLabAPI groups the narrowed GitLab service interfaces.
type GitLabAPI struct {
	Projects        GitLabProjectsService
	Repositories    GitLabRepositoriesService
	RepositoryFiles GitLabRepositoryFilesService
}

// wrapGitLabClient constructs GitLabAPI from a *gitlab.Client.
func wrapGitLabClient(c *gitlab.Client) GitLabAPI {
	return GitLabAPI{
		Projects:        &gitlabProjectsWrapper{client: c},
		Repositories:    &gitlabRepositoriesWrapper{client: c},
		RepositoryFiles: &gitlabRepositoryFilesWrapper{client: c},
	}
}

/////////////////////////
// Testing Guidance
/////////////////////////

// In unit tests you can define custom types implementing these interfaces:
//
//   type fakeGitHubRepos struct { /* fields */ }
//   func (f *fakeGitHubRepos) Get(...) (*github.Repository, *github.Response, error) { ... }
//
// Then construct a GitHubAPI:
//   ghAPI := GitHubAPI{Repositories: &fakeGitHubRepos{...}, Git: &fakeGitHubGit{...}}
//
// A future enhancement can add alternative constructors:
//   NewGitHubClientWithAPI(config Config, api GitHubAPI) *GitHubClient
//   NewGitLabClientWithAPI(config Config, api GitLabAPI) *GitLabClient
//
// This keeps production code unchanged while enabling pure logic tests that
// exercise transformation, filtering, and fallback behaviors without network.
