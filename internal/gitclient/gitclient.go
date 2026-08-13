// Package gitclient wraps go-git for clone, pull, and metadata reads. It is
// the only package that touches git remotes — the app's one network integration.
package gitclient

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/anofac/markdownia/internal/domain"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// ProgressCallback receives clone/pull progress updates.
type ProgressCallback = func(text string)

// Client clones, pulls, and reads git metadata. Interface is declared here for
// the source usecase to consume.
type Client interface {
	Clone(ctx context.Context, url, branch, dest string, progress ProgressCallback) error
	Pull(ctx context.Context, dest string, progress ProgressCallback) error
	HeadInfo(dest string) (branch, commit string, err error)
	IsRepository(path string) bool
}

// GitClient is the go-git backed implementation.
type GitClient struct{}

// New constructs the git client.
func New() *GitClient { return &GitClient{} }

// Clone clones a repository to dest. Cancellation via ctx aborts the clone.
func (c *GitClient) Clone(ctx context.Context, remoteURL, branch, dest string, progress ProgressCallback) error {
	opts := &gogit.CloneOptions{
		URL:      remoteURL,
		Progress: &progressWriter{cb: progress, ctx: ctx},
	}
	if branch != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(branch)
		opts.SingleBranch = true
	}
	if _, err := gogit.PlainCloneContext(ctx, dest, false, opts); err != nil {
		return classifyCloneErr(err)
	}
	return nil
}

// Pull fetches and merges upstream changes for an existing checkout.
func (c *GitClient) Pull(ctx context.Context, dest string, progress ProgressCallback) error {
	repo, err := gogit.PlainOpen(dest)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrNotAGitRepo, err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	if err := wt.PullContext(ctx, &gogit.PullOptions{
		Progress: &progressWriter{cb: progress, ctx: ctx},
	}); err != nil {
		if err == gogit.NoErrAlreadyUpToDate {
			return nil
		}
		return classifyPullErr(err)
	}
	return nil
}

// HeadInfo returns the current branch and HEAD commit SHA of a checkout.
func (c *GitClient) HeadInfo(dest string) (string, string, error) {
	repo, err := gogit.PlainOpen(dest)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", domain.ErrNotAGitRepo, err)
	}
	head, err := repo.Head()
	if err != nil {
		return "", "", err
	}
	branch := ""
	if head.Name().IsBranch() {
		branch = head.Name().Short()
	}
	return branch, head.Hash().String(), nil
}

// IsRepository reports whether path contains a git repository.
func (c *GitClient) IsRepository(path string) bool {
	repo, err := gogit.PlainOpen(path)
	return err == nil && repo != nil
}

// NormalizeURL normalizes git URL forms: HTTPS, SSH, owner/repo shorthand, and
// a pasted /tree/<branch> URL (returning the extracted branch).
func NormalizeURL(input string) (remoteURL, branch string, err error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", "", fmt.Errorf("url is empty")
	}

	// owner/repo shorthand → GitHub HTTPS.
	if !strings.Contains(s, "://") && !strings.HasPrefix(s, "git@") && strings.Count(s, "/") == 1 {
		return "https://github.com/" + s, "", nil
	}

	// Pasted browser URL with a tree path: /tree/<branch>.
	if strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") {
		u, perr := url.Parse(s)
		if perr == nil {
			parts := strings.Split(strings.Trim(u.Path, "/"), "/")
			for i, p := range parts {
				if p == "tree" && i+1 < len(parts) {
					branch = parts[i+1]
					u.Path = "/" + strings.Join(parts[:i], "/")
					return u.String(), branch, nil
				}
			}
		}
	}

	return s, "", nil
}

// classifyCloneErr maps a go-git clone failure to a domain error.
func classifyCloneErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case isAuthError(err):
		return domain.ErrAuthRequired
	case isNotFoundError(err):
		return domain.ErrRepoNotFound
	case isTransportError(err):
		return domain.ErrHostUnreachable
	default:
		return fmt.Errorf("%w: %v", domain.ErrCloneFailed, err)
	}
}

func classifyPullErr(err error) error {
	switch {
	case isAuthError(err):
		return domain.ErrAuthRequired
	case isNotFoundError(err):
		return domain.ErrRepoNotFound
	case isTransportError(err):
		return domain.ErrHostUnreachable
	default:
		return fmt.Errorf("%w: %v", domain.ErrPullFailed, err)
	}
}

func isAuthError(err error) bool {
	return strings.Contains(err.Error(), "authentication required") ||
		strings.Contains(err.Error(), "invalid credentials") ||
		strings.Contains(err.Error(), "could not read Username")
}

func isNotFoundError(err error) bool {
	return strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "repository not found") ||
		strings.Contains(err.Error(), "404")
}

func isTransportError(err error) bool {
	return strings.Contains(err.Error(), "dial tcp") ||
		strings.Contains(err.Error(), "no such host") ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "could not resolve host") ||
		strings.Contains(err.Error(), "timeout")
}

// progressWriter adapts a ProgressCallback to go-git's io.Writer.
type progressWriter struct {
	cb  ProgressCallback
	ctx context.Context
}

func (p *progressWriter) Write(b []byte) (int, error) {
	if p.ctx.Err() != nil {
		return 0, p.ctx.Err()
	}
	if p.cb != nil {
		p.cb(string(b))
	}
	return len(b), nil
}
