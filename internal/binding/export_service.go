package binding

import (
	"context"
	"log/slog"

	"github.com/anofac/markdownia/internal/usecase/export"
)

// ExportUsecase is what ExportService delegates to.
type ExportUsecase interface {
	PrepareExport(ctx context.Context, target export.Target) (export.Payload, error)
	ExportHTML(ctx context.Context, target export.Target, destPath string) error
}

// ExportService is the Wails-bound export surface.
type ExportService struct {
	usecase ExportUsecase
	logger  *slog.Logger
}

// NewExportService constructs the bound export service.
func NewExportService(usecase ExportUsecase, logger *slog.Logger) *ExportService {
	return &ExportService{usecase: usecase, logger: logger}
}

// PrepareExport returns the payload the frontend mounts offscreen and prints.
func (s *ExportService) PrepareExport(ctx context.Context, target ExportTargetDTO) (ExportPayloadDTO, error) {
	payload, err := s.usecase.PrepareExport(ctxFromArgs(ctx), export.Target{
		Kind:         target.Kind,
		DocumentID:   target.DocumentID,
		CollectionID: target.CollectionID,
		Theme:        target.Theme,
		IncludeTOC:   target.IncludeTOC,
		ShowLinkURLs: target.ShowLinkURLs,
	})
	if err != nil {
		return ExportPayloadDTO{}, err
	}
	return ExportPayloadDTO{Title: payload.Title, HTML: payload.HTML, Theme: payload.Theme}, nil
}

// ExportHTML writes a standalone file directly Go-side.
func (s *ExportService) ExportHTML(ctx context.Context, target ExportTargetDTO, destPath string) error {
	return s.usecase.ExportHTML(ctxFromArgs(ctx), export.Target{
		Kind:         target.Kind,
		DocumentID:   target.DocumentID,
		CollectionID: target.CollectionID,
		Theme:        target.Theme,
		IncludeTOC:   target.IncludeTOC,
		ShowLinkURLs: target.ShowLinkURLs,
	}, destPath)
}
