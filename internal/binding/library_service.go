package binding

import (
	"context"
	"log/slog"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/usecase/library"
)

// LibraryUsecase is what LibraryService delegates to.
type LibraryUsecase interface {
	GetTree(ctx context.Context, sourceID int64) ([]library.TreeNode, error)
	OpenDocument(ctx context.Context, docID int64, ctxType domain.ContextType, ctxID int64) (library.OpenPayload, error)
	GetDocumentMeta(ctx context.Context, docID int64) (domain.Document, error)
	ResolveLink(ctx context.Context, fromDocID int64, href string) (library.LinkTarget, error)
	GetAsset(ctx context.Context, docID int64, relPath string) ([]byte, string, error)
	ListRecent(ctx context.Context, limit int64) ([]library.RecentDoc, error)
}

// LibraryService is the Wails-bound library surface.
type LibraryService struct {
	usecase LibraryUsecase
	logger  *slog.Logger
}

// NewLibraryService constructs the bound library service.
func NewLibraryService(usecase LibraryUsecase, logger *slog.Logger) *LibraryService {
	return &LibraryService{usecase: usecase, logger: logger}
}

func (s *LibraryService) GetTree(ctx context.Context, sourceID int64) ([]TreeNodeDTO, error) {
	nodes, err := s.usecase.GetTree(ctxFromArgs(ctx), sourceID)
	if err != nil {
		return nil, err
	}
	out := make([]TreeNodeDTO, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, TreeNodeDTO{
			ID: n.ID, RelPath: n.RelPath, Title: n.Title, Depth: n.Depth, IsFolder: n.IsFolder,
		})
	}
	return out, nil
}

// TreeNodeDTO is one entry in the source tree.
type TreeNodeDTO struct {
	ID       int64  `json:"id"`
	RelPath  string `json:"relPath"`
	Title    string `json:"title"`
	Depth    int    `json:"depth"`
	IsFolder bool   `json:"isFolder"`
}

func (s *LibraryService) OpenDocument(ctx context.Context, docID int64, contextType string, contextID int64) (DocumentDTO, error) {
	ct := domain.ContextType(contextType)
	if ct == "" {
		ct = domain.ContextLibrary
	}
	payload, err := s.usecase.OpenDocument(ctxFromArgs(ctx), docID, ct, contextID)
	if err != nil {
		return DocumentDTO{}, err
	}
	highlights := make([]HighlightDTO, 0, len(payload.Highlights))
	for _, h := range payload.Highlights {
		highlights = append(highlights, toHighlightDTO(h))
	}
	doc := toDocumentDTO(payload.Document, false)
	doc.Highlights = highlights
	return doc, nil
}

func (s *LibraryService) GetDocumentMeta(ctx context.Context, docID int64) (DocumentMetaDTO, error) {
	d, err := s.usecase.GetDocumentMeta(ctxFromArgs(ctx), docID)
	if err != nil {
		return DocumentMetaDTO{}, err
	}
	return DocumentMetaDTO{
		ID: d.ID, SourceID: d.SourceID, RelPath: d.RelPath, Title: d.Title, WordCount: d.WordCount,
	}, nil
}

func (s *LibraryService) ResolveLink(ctx context.Context, fromDocID int64, href string) (LinkTargetDTO, error) {
	t, err := s.usecase.ResolveLink(ctxFromArgs(ctx), fromDocID, href)
	if err != nil {
		return LinkTargetDTO{}, err
	}
	return LinkTargetDTO{
		Internal: t.Internal, External: t.External, DocumentID: t.DocumentID,
		Anchor: t.Anchor, URL: t.URL, Folder: t.Folder, FolderRel: t.FolderRel,
		NotInLibrary: !t.Internal && !t.External && t.DocumentID == 0 && t.URL == "",
	}, nil
}

func (s *LibraryService) GetAsset(ctx context.Context, docID int64, relPath string) ([]byte, string, error) {
	return s.usecase.GetAsset(ctxFromArgs(ctx), docID, relPath)
}

func (s *LibraryService) ListRecent(ctx context.Context, limit int64) ([]RecentDocDTO, error) {
	recent, err := s.usecase.ListRecent(ctxFromArgs(ctx), limit)
	if err != nil {
		return nil, err
	}
	out := make([]RecentDocDTO, 0, len(recent))
	for _, r := range recent {
		out = append(out, RecentDocDTO{
			DocumentID: r.DocumentID, Title: r.Title, RelPath: r.RelPath, SourceID: r.SourceID,
			LastOpenedAt: r.LastOpenedAt, FurthestScrollPct: r.FurthestScrollPct,
		})
	}
	return out, nil
}

// RecentDocDTO is one recently-read document.
type RecentDocDTO struct {
	DocumentID        int64   `json:"documentId"`
	Title             string  `json:"title"`
	RelPath           string  `json:"relPath"`
	SourceID          int64   `json:"sourceId"`
	LastOpenedAt      string  `json:"lastOpenedAt"`
	FurthestScrollPct float64 `json:"furthestScrollPct"`
}
