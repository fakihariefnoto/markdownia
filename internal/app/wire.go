// Package app wires the whole Go layer. This is the ONE place concrete types
// are constructed; every service receives interfaces. Grepping for sqlite.New
// outside this package must return nothing.
package app

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/anofac/markdownia/internal/binding"
	"github.com/anofac/markdownia/internal/gitclient"
	"github.com/anofac/markdownia/internal/indexer"
	"github.com/anofac/markdownia/internal/markdown"
	"github.com/anofac/markdownia/internal/repository/sqlite"
	"github.com/anofac/markdownia/internal/usecase/annotation"
	"github.com/anofac/markdownia/internal/usecase/collection"
	"github.com/anofac/markdownia/internal/usecase/export"
	"github.com/anofac/markdownia/internal/usecase/library"
	"github.com/anofac/markdownia/internal/usecase/reading"
	"github.com/anofac/markdownia/internal/usecase/search"
	sourceusecase "github.com/anofac/markdownia/internal/usecase/source"
)

// Services bundles every bound service for registration with Wails.
type Services struct {
	Source     *binding.SourceService
	Library    *binding.LibraryService
	Search     *binding.SearchService
	Collection *binding.CollectionService
	Annotation *binding.AnnotationService
	Reading    *binding.ReadingService
	Settings   *binding.SettingsService
	Export     *binding.ExportService
	Native     *binding.NativeService
}

// Emitter is the event emission implementation (injected from the Wails layer).
type Emitter interface {
	Emit(name string, data any)
}

// SettingsKV is the narrow settings view the native layer needs.
type SettingsKV interface {
	Get(ctx context.Context, key string) (json.RawMessage, bool, error)
	Set(ctx context.Context, key string, value json.RawMessage) error
}

// Build constructs every usecase and bound service from the database handle
// and the Wails-coupled collaborators.
func Build(db *sqlite.DB, emitter Emitter, native binding.NativeLayer, extractedRoot string, settings SettingsKV, logger *slog.Logger) (*Services, error) {
	// Repositories.
	srcRepo := sqlite.NewSourceRepository(db)
	docRepo := sqlite.NewDocumentRepository(db)
	searchRepo := sqlite.NewSearchRepository(db)
	colRepo := sqlite.NewCollectionRepository(db)
	annRepo := sqlite.NewAnnotationRepository(db)
	readRepo := sqlite.NewReadingRepository(db)
	setRepo := sqlite.NewSettingsRepository(db)

	// Collaborators.
	parser := markdown.NewParser()
	git := gitclient.New()
	events := binding.NewProgressSink(emitter)

	// Indexer: the sweep hook is the annotation repository.
	ix := indexer.New(srcRepo, docRepo, parser, annRepo, events, 0)

	// Usecases.
	srcService := sourceusecase.New(sourceusecase.Options{
		Repo:          srcRepo,
		Git:           git,
		Indexer:       ix,
		Progress:      events,
		ExtractedRoot: extractedRoot,
	})
	libService := library.New(docRepo, srcRepo, annRepo, readRepo, docRepo, parser)
	searchService := search.New(sqlite.NewSearchAdapter(searchRepo))
	colService := collection.New(colRepo)
	annService := annotation.New(annRepo, docRepo, annotation.NewMarkdownBlocks(docRepo, srcRepo, parser))
	readService := reading.New(readRepo, setRepo, docRepo)
	exportService := export.New(docRepo, collectionExportAdapter{repo: colRepo}, parser)

	// Bound services.
	return &Services{
		Source:     binding.NewSourceService(srcService, logger),
		Library:    binding.NewLibraryService(libService, logger),
		Search:     binding.NewSearchService(searchService, logger),
		Collection: binding.NewCollectionService(colService, logger),
		Annotation: binding.NewAnnotationService(annService, logger),
		Reading:    binding.NewReadingService(readService, logger),
		Settings:   binding.NewSettingsService(readService, logger),
		Export:     binding.NewExportService(exportService, logger),
		Native:     binding.NewNativeService(native, logger),
	}, nil
}

// collectionExportAdapter adapts the collection repo to the export usecase's
// narrow DocRow interface.
type collectionExportAdapter struct {
	repo *sqlite.CollectionRepository
}

// ListDocuments returns collection memberships as export DocRows.
func (a collectionExportAdapter) ListDocuments(ctx context.Context, collectionID int64) ([]export.DocRow, error) {
	rows, err := a.repo.ListDocuments(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	out := make([]export.DocRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, export.DocRow{
			DocumentID: r.DocumentID, Title: r.Title, RelPath: r.RelPath,
			SourceName: r.SourceName, SortOrder: r.SortOrder,
		})
	}
	return out, nil
}
