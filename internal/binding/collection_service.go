package binding

import (
	"context"
	"log/slog"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/repository/sqlite"
	"github.com/anofac/markdownia/internal/usecase/collection"
)

// CollectionUsecase is what CollectionService delegates to.
type CollectionUsecase interface {
	ListCollections(ctx context.Context) ([]collection.Collection, error)
	CreateCollection(ctx context.Context, name, description string) (int64, error)
	RenameCollection(ctx context.Context, id int64, name string) error
	UpdateDescription(ctx context.Context, id int64, description string) error
	DeleteCollection(ctx context.Context, id int64) error
	AddDocuments(ctx context.Context, collectionID int64, docIDs []int64) error
	RemoveDocuments(ctx context.Context, collectionID int64, docIDs []int64) error
	ReorderDocuments(ctx context.Context, collectionID int64, orderedIDs []int64) error
	ListCollectionDocuments(ctx context.Context, id int64) ([]sqlite.CollectionDocumentRow, error)
	CollectionsForDocument(ctx context.Context, docID int64) ([]domain.Collection, error)
}

// CollectionService is the Wails-bound collection surface.
type CollectionService struct {
	usecase CollectionUsecase
	logger  *slog.Logger
}

// NewCollectionService constructs the bound collection service.
func NewCollectionService(usecase CollectionUsecase, logger *slog.Logger) *CollectionService {
	return &CollectionService{usecase: usecase, logger: logger}
}

func (s *CollectionService) ListCollections(ctx context.Context) ([]CollectionDTO, error) {
	list, err := s.usecase.ListCollections(ctxFromArgs(ctx))
	if err != nil {
		return nil, err
	}
	out := make([]CollectionDTO, 0, len(list))
	for _, c := range list {
		out = append(out, CollectionDTO{
			ID: c.ID, Name: c.Name, Description: c.Description, Icon: c.Icon, DocumentCount: c.DocumentCount,
		})
	}
	return out, nil
}

func (s *CollectionService) CreateCollection(ctx context.Context, name, description string) (int64, error) {
	return s.usecase.CreateCollection(ctxFromArgs(ctx), name, description)
}

func (s *CollectionService) RenameCollection(ctx context.Context, id int64, name string) error {
	return s.usecase.RenameCollection(ctxFromArgs(ctx), id, name)
}

func (s *CollectionService) DeleteCollection(ctx context.Context, id int64) error {
	return s.usecase.DeleteCollection(ctxFromArgs(ctx), id)
}

func (s *CollectionService) AddDocuments(ctx context.Context, collectionID int64, docIDs []int64) error {
	return s.usecase.AddDocuments(ctxFromArgs(ctx), collectionID, docIDs)
}

func (s *CollectionService) RemoveDocuments(ctx context.Context, collectionID int64, docIDs []int64) error {
	return s.usecase.RemoveDocuments(ctxFromArgs(ctx), collectionID, docIDs)
}

func (s *CollectionService) ReorderDocuments(ctx context.Context, collectionID int64, orderedIDs []int64) error {
	return s.usecase.ReorderDocuments(ctxFromArgs(ctx), collectionID, orderedIDs)
}

func (s *CollectionService) ListCollectionDocuments(ctx context.Context, id int64) ([]CollectionDocumentDTO, error) {
	rows, err := s.usecase.ListCollectionDocuments(ctxFromArgs(ctx), id)
	if err != nil {
		return nil, err
	}
	out := make([]CollectionDocumentDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, CollectionDocumentDTO{
			DocumentID: r.DocumentID, Title: r.Title, RelPath: r.RelPath,
			SourceName: r.SourceName, SortOrder: r.SortOrder,
		})
	}
	return out, nil
}

func (s *CollectionService) CollectionsForDocument(ctx context.Context, docID int64) ([]CollectionDTO, error) {
	list, err := s.usecase.CollectionsForDocument(ctxFromArgs(ctx), docID)
	if err != nil {
		return nil, err
	}
	out := make([]CollectionDTO, 0, len(list))
	for _, c := range list {
		out = append(out, CollectionDTO{ID: c.ID, Name: c.Name, Description: c.Description, Icon: c.Icon})
	}
	return out, nil
}
