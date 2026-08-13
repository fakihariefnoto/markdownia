package binding

import (
	"context"
	"log/slog"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/usecase/source"
)

// SourceUsecase is what SourceService delegates to.
type SourceUsecase interface {
	List(ctx context.Context) ([]domain.Source, error)
	ImportFolder(ctx context.Context, path string) (int64, error)
	ImportGit(ctx context.Context, url, branch string) (int64, error)
	ImportZip(ctx context.Context, path string) (int64, error)
	RefreshSource(ctx context.Context, id int64) error
	RelocateSource(ctx context.Context, id int64, newPath string) error
	RenameSource(ctx context.Context, id int64, name string) error
	SourceDeletionPreview(ctx context.Context, id int64) (source.DeletionPreview, error)
	DeleteSource(ctx context.Context, id int64) error
	CancelSourceOperation(ctx context.Context, id int64) error
	RebuildAll(ctx context.Context) error
}

// SourceService is the Wails-bound source surface.
type SourceService struct {
	usecase SourceUsecase
	logger  *slog.Logger
}

// NewSourceService constructs the bound source service.
func NewSourceService(usecase SourceUsecase, logger *slog.Logger) *SourceService {
	return &SourceService{usecase: usecase, logger: logger}
}

func (s *SourceService) ListSources(ctx context.Context) ([]SourceDTO, error) {
	list, err := s.usecase.List(ctxFromArgs(ctx))
	if err != nil {
		return nil, err
	}
	out := make([]SourceDTO, 0, len(list))
	for _, src := range list {
		out = append(out, toSourceDTO(src))
	}
	return out, nil
}

// ImportFolder accepts a path from the native folder dialog, never the page.
func (s *SourceService) ImportFolder(ctx context.Context, path string) (int64, error) {
	s.logger.Info("ImportFolder called", "path", path)
	id, err := s.usecase.ImportFolder(ctxFromArgs(ctx), path)
	s.logger.Info("ImportFolder result", "id", id, "error", err)
	return id, err
}

func (s *SourceService) ImportGit(ctx context.Context, url, branch string) (int64, error) {
	return s.usecase.ImportGit(ctxFromArgs(ctx), url, branch)
}

func (s *SourceService) ImportZip(ctx context.Context, path string) (int64, error) {
	return s.usecase.ImportZip(ctxFromArgs(ctx), path)
}

func (s *SourceService) RefreshSource(ctx context.Context, id int64) error {
	return s.usecase.RefreshSource(ctxFromArgs(ctx), id)
}

func (s *SourceService) RelocateSource(ctx context.Context, id int64, newPath string) error {
	return s.usecase.RelocateSource(ctxFromArgs(ctx), id, newPath)
}

func (s *SourceService) RenameSource(ctx context.Context, id int64, name string) error {
	return s.usecase.RenameSource(ctxFromArgs(ctx), id, name)
}

func (s *SourceService) DeleteSource(ctx context.Context, id int64) error {
	return s.usecase.DeleteSource(ctxFromArgs(ctx), id)
}

func (s *SourceService) SourceDeletionPreview(ctx context.Context, id int64) (DeletionPreviewDTO, error) {
	p, err := s.usecase.SourceDeletionPreview(ctxFromArgs(ctx), id)
	if err != nil {
		return DeletionPreviewDTO{}, err
	}
	return DeletionPreviewDTO{
		Documents:          p.Documents,
		Highlights:         p.Highlights,
		Bookmarks:          p.Bookmarks,
		CollectionEntries:  p.CollectionEntries,
		DeletesFilesOnDisk: p.DeletesFilesOnDisk,
	}, nil
}

func (s *SourceService) CancelSourceOperation(ctx context.Context, id int64) error {
	return s.usecase.CancelSourceOperation(ctxFromArgs(ctx), id)
}

func (s *SourceService) RebuildAll(ctx context.Context) error {
	return s.usecase.RebuildAll(ctxFromArgs(ctx))
}
