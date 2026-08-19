// Package source implements the source domain: importing and re-scanning a
// folder, git repo, or zip; source lifecycle; and deletion. The physical
// browsing axis (PRD decision D5).
package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anofac/markdownia/internal/archive"
	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/gitclient"
	"github.com/anofac/markdownia/internal/pathguard"
)

// Service is the source domain usecase.
type Service struct {
	repo          Repository
	git           GitClient
	indexer       Indexer
	progress      ProgressSink
	extractedRoot string
	now           func() time.Time
}

// Options configures the service.
type Options struct {
	Repo          Repository
	Git           GitClient
	Indexer       Indexer
	Progress      ProgressSink
	ExtractedRoot string
	Now           func() time.Time
}

// New constructs the source usecase.
func New(opts Options) *Service {
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{
		repo:          opts.Repo,
		git:           opts.Git,
		indexer:       opts.Indexer,
		progress:      opts.Progress,
		extractedRoot: opts.ExtractedRoot,
		now:           opts.Now,
	}
}

// List returns all sources ordered by name.
func (s *Service) List(ctx context.Context) ([]domain.Source, error) {
	return s.repo.List(ctx)
}

// ImportFolder creates a source referencing an existing folder in place and
// kicks off an async index. The app never copies or moves user content.
func (s *Service) ImportFolder(ctx context.Context, path string) (int64, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", domain.ErrInvalidArgument, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return 0, fmt.Errorf("%w: folder %s: %v", domain.ErrInvalidArgument, abs, err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("%w: %s is not a directory", domain.ErrInvalidArgument, abs)
	}

	kind := domain.SourceKindFolder
	var branch, commit string
	if s.git.IsRepository(abs) {
		branch, commit, _ = s.git.HeadInfo(abs)
		kind = domain.SourceKindFolder // referenced in place, git metadata recorded
	}

	id, err := s.repo.Create(ctx, &domain.Source{
		Kind:      kind,
		Name:      filepath.Base(abs),
		RootPath:  abs,
		GitBranch: branch,
		GitCommit: commit,
		Status:    domain.StatusPending,
		CreatedAt: s.now().Format(time.RFC3339),
		UpdatedAt: s.now().Format(time.RFC3339),
	})
	if err != nil {
		return 0, err
	}

	go func() {
		_ = s.runIndex(context.Background(), id, func(ctx context.Context) error {
			return s.indexer.Index(ctx, id)
		})
	}()
	return id, nil
}

// ImportGit clones a remote repo and indexes it. Returns the source id
// immediately; clone and index run async with progress events.
func (s *Service) ImportGit(ctx context.Context, url, branch string) (int64, error) {
	remoteURL, branchFromURL, err := gitclient.NormalizeURL(url)
	if err != nil {
		return 0, err
	}
	if branch == "" {
		branch = branchFromURL
	}

	id, err := s.repo.Create(ctx, &domain.Source{
		Kind:      domain.SourceKindGit,
		Name:      repoNameFromURL(remoteURL),
		RootPath:  filepath.Join(s.extractedRoot, "git"),
		OriginURL: remoteURL,
		GitBranch: branch,
		Status:    domain.StatusCloning,
		CreatedAt: s.now().Format(time.RFC3339),
		UpdatedAt: s.now().Format(time.RFC3339),
	})
	if err != nil {
		return 0, err
	}

	dest := filepath.Join(s.extractedRoot, "git", fmt.Sprintf("src-%d", id))
	go s.runClone(id, dest, remoteURL, branch)
	return id, nil
}

func (s *Service) runClone(id int64, dest, url, branch string) {
	ctx := context.Background()
	src, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return
	}
	src.RootPath = dest
	s.progress.SourceStatus(id, domain.StatusCloning, "")
	_ = os.RemoveAll(dest)

	err = s.git.Clone(ctx, url, branch, dest, func(t string) {
		s.progress.SourceProgress(id, "cloning", 0, 0)
	})
	if err != nil {
		// A failed clone creates no source — remove the row entirely.
		_ = s.repo.Delete(ctx, id)
		_ = os.RemoveAll(dest)
		s.progress.SourceStatus(id, domain.StatusError, err.Error())
		return
	}

	src.RootPath = dest
	if br, cm, err := s.git.HeadInfo(dest); err == nil {
		src.GitBranch = br
		src.GitCommit = cm
	}
	_ = s.repo.Update(ctx, &src)

	if err := s.runIndex(ctx, id, func(ctx context.Context) error {
		return s.indexer.Index(ctx, id)
	}); err != nil {
		_ = s.repo.SetStatus(ctx, id, domain.StatusError, err.Error())
	}
}

// ImportZip extracts a zip into app-managed storage and indexes it.
func (s *Service) ImportZip(ctx context.Context, path string) (int64, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return 0, err
	}
	name := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))

	id, err := s.repo.Create(ctx, &domain.Source{
		Kind:      domain.SourceKindZip,
		Name:      name,
		IsManaged: true,
		Status:    domain.StatusExtracting,
		CreatedAt: s.now().Format(time.RFC3339),
		UpdatedAt: s.now().Format(time.RFC3339),
	})
	if err != nil {
		return 0, err
	}

	dest := filepath.Join(s.extractedRoot, "zip", fmt.Sprintf("src-%d", id))
	src := domain.Source{
		ID: id, Kind: domain.SourceKindZip, Name: name,
		RootPath: dest, IsManaged: true, Status: domain.StatusExtracting,
		CreatedAt: s.now().Format(time.RFC3339), UpdatedAt: s.now().Format(time.RFC3339),
	}
	if err := s.repo.Update(ctx, &src); err != nil {
		return 0, err
	}

	go s.runZip(id, abs, dest)
	return id, nil
}

func (s *Service) runZip(id int64, srcPath, dest string) {
	ctx := context.Background()
	s.progress.SourceStatus(id, domain.StatusExtracting, "")

	err := archive.Extract(ctx, srcPath, dest, func(cur, total int) {
		s.progress.SourceProgress(id, "extracting", cur, total)
	})
	if err != nil {
		_ = s.repo.Delete(ctx, id)
		_ = os.RemoveAll(dest)
		s.progress.SourceStatus(id, domain.StatusError, err.Error())
		return
	}

	if err := s.runIndex(ctx, id, func(ctx context.Context) error {
		return s.indexer.Index(ctx, id)
	}); err != nil {
		_ = s.repo.SetStatus(ctx, id, domain.StatusError, err.Error())
	}
}

// runIndex runs the indexer and reports completion/status.
func (s *Service) runIndex(ctx context.Context, id int64, fn func(context.Context) error) error {
	s.progress.SourceStatus(id, domain.StatusIndexing, "")
	if err := fn(ctx); err != nil {
		s.progress.SourceStatus(id, domain.StatusError, err.Error())
		return err
	}
	return nil
}

// RefreshSource re-scans a source: folder → mtime scan, git → pull then scan,
// zip → rejected (a zip has no origin to refresh from).
func (s *Service) RefreshSource(ctx context.Context, id int64) error {
	src, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	switch src.Kind {
	case domain.SourceKindZip:
		return domain.ErrRefreshNotAllowed
	case domain.SourceKindGit:
		dest := src.RootPath
		if err := s.git.Pull(ctx, dest, nil); err != nil {
			// A failed pull leaves the source at its previous good state.
			return err
		}
		if br, cm, err := s.git.HeadInfo(dest); err == nil {
			src.GitBranch = br
			src.GitCommit = cm
			_ = s.repo.Update(ctx, &src)
		}
	}
	return s.runIndex(ctx, id, func(ctx context.Context) error {
		return s.indexer.Index(ctx, id)
	})
}

// RelocateSource re-points a source's root_path and re-indexes. Documents,
// highlights, and bookmarks survive because they key on rel_path.
func (s *Service) RelocateSource(ctx context.Context, id int64, newPath string) error {
	abs, err := filepath.Abs(newPath)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInvalidArgument, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s is not a directory", domain.ErrInvalidArgument, abs)
	}
	src, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	src.RootPath = abs
	src.Status = domain.StatusPending
	src.ErrorMessage = ""
	if err := s.repo.Update(ctx, &src); err != nil {
		return err
	}

	// Re-pointing means the old cached documents no longer match the new
	// location. Re-index now so the library reflects the new folder; failures
	// surface via the source status.
	go func() {
		_ = s.runIndex(context.Background(), id, func(ctx context.Context) error {
			return s.indexer.Index(ctx, id)
		})
	}()
	return nil
}

// RenameSource changes only the display name.
func (s *Service) RenameSource(ctx context.Context, id int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.ErrInvalidArgument
	}
	src, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	src.Name = name
	return s.repo.Update(ctx, &src)
}

// SourceDeletionPreview returns the counts the confirm dialog needs.
func (s *Service) SourceDeletionPreview(ctx context.Context, id int64) (DeletionPreview, error) {
	src, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return DeletionPreview{}, err
	}
	docs, hl, bm, col, err := s.repo.Counts(ctx, id)
	if err != nil {
		return DeletionPreview{}, err
	}
	return DeletionPreview{
		Documents:          docs,
		Highlights:         hl,
		Bookmarks:          bm,
		CollectionEntries:  col,
		DeletesFilesOnDisk: src.IsManaged,
	}, nil
}

// DeleteSource removes a source and everything hanging off it. Extracted files
// (managed sources) are removed path-guarded to the app-data root; referenced
// sources are never touched on disk.
func (s *Service) DeleteSource(ctx context.Context, id int64) error {
	src, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if src.IsManaged {
		// pathguard against the app-data root so a corrupted root_path cannot
		// delete somewhere else.
		contained := pathguard.Join(s.extractedRoot, relTo(src.RootPath))
		if contained != "" && strings.HasPrefix(contained, s.extractedRoot) {
			_ = os.RemoveAll(contained)
		}
	}
	return nil
}

// relTo returns rootPath relative to the extracted root, for the delete guard.
func relTo(rootPath string) string {
	rel, err := filepath.Rel("/", rootPath)
	if err != nil {
		return ""
	}
	return strings.TrimLeft(rel, "/")
}

// CancelSourceOperation cancels an in-flight operation. The source usecase
// relies on ctx cancellation reaching the indexer; a fully-running clone is
// cancelled by the caller passing a cancellable context.
func (s *Service) CancelSourceOperation(ctx context.Context, id int64) error {
	// Deletion of the partial source is the caller's job (import flows cancel
	// via context). Here we only refuse when the source is mid-operation and
	// report that it must be deleted rather than left visible.
	src, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if src.Status != domain.StatusReady && src.Status != domain.StatusError &&
		src.Status != domain.StatusUnavailable {
		_ = s.repo.Delete(ctx, id)
	}
	return nil
}

// RebuildAll re-indexes every source in the library, sequentially, reporting
// progress per source via the progress sink. Returns immediately after kicking
// off the async pass.
func (s *Service) RebuildAll(ctx context.Context) error {
	all, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	go func() {
		for _, src := range all {
			// RefreshSource: folder → rescan, git → pull+scan, zip → rejected.
			_ = s.RefreshSource(context.Background(), src.ID)
		}
	}()
	return nil
}

func repoNameFromURL(remoteURL string) string {
	u := strings.TrimSuffix(remoteURL, "/")
	if i := strings.LastIndex(u, "/"); i >= 0 {
		u = u[i+1:]
	}
	u = strings.TrimSuffix(u, ".git")
	if u == "" {
		return "git repo"
	}
	return u
}
