package binding

import (
	"context"
	"log/slog"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/repository/sqlite"
	"github.com/anofac/markdownia/internal/usecase/annotation"
)

// AnnotationUsecase is what AnnotationService delegates to.
type AnnotationUsecase interface {
	AddBookmark(ctx context.Context, docID int64, headingAnchor, note *string) (int64, error)
	RemoveBookmark(ctx context.Context, id int64) error
	ListBookmarks(ctx context.Context, sourceID *int64) ([]sqlite.BookmarkRow, error)
	AddHighlight(ctx context.Context, docID int64, anchor domain.HighlightAnchor, color string, note *string) (int64, error)
	UpdateHighlight(ctx context.Context, id int64, color *string, note *string) error
	RemoveHighlight(ctx context.Context, id int64) error
	ListHighlights(ctx context.Context, docID int64) ([]domain.Highlight, error)
	ListAllAnnotations(ctx context.Context, sourceID *int64) (annotation.Annotations, error)
}

// AnnotationService is the Wails-bound annotation surface.
type AnnotationService struct {
	usecase AnnotationUsecase
	logger  *slog.Logger
}

// NewAnnotationService constructs the bound annotation service.
func NewAnnotationService(usecase AnnotationUsecase, logger *slog.Logger) *AnnotationService {
	return &AnnotationService{usecase: usecase, logger: logger}
}

func ptr[T any](v T) *T { return &v }

func (s *AnnotationService) AddBookmark(ctx context.Context, docID int64, headingAnchor, note string) (int64, error) {
	var ha, n *string
	if headingAnchor != "" {
		ha = ptr(headingAnchor)
	}
	if note != "" {
		n = ptr(note)
	}
	return s.usecase.AddBookmark(ctxFromArgs(ctx), docID, ha, n)
}

func (s *AnnotationService) RemoveBookmark(ctx context.Context, id int64) error {
	return s.usecase.RemoveBookmark(ctxFromArgs(ctx), id)
}

func (s *AnnotationService) ListBookmarks(ctx context.Context) ([]BookmarkDTO, error) {
	rows, err := s.usecase.ListBookmarks(ctxFromArgs(ctx), nil)
	if err != nil {
		return nil, err
	}
	out := make([]BookmarkDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, BookmarkDTO{
			ID: r.ID, DocumentID: r.DocumentID, HeadingAnchor: r.HeadingAnchor,
			Note: r.Note, Title: r.Title, RelPath: r.RelPath, SourceName: r.SourceName,
		})
	}
	return out, nil
}

// AddHighlight validates the anchor Go-side; offsets are relative to the
// block's plain text, computed frontend-side from the rendered DOM.
func (s *AnnotationService) AddHighlight(ctx context.Context, docID int64, anchor AnchorDTO, color, note string) (int64, error) {
	var n *string
	if note != "" {
		n = ptr(note)
	}
	return s.usecase.AddHighlight(ctxFromArgs(ctx), docID, domain.HighlightAnchor{
		BlockHash:   anchor.BlockHash,
		BlockIndex:  anchor.BlockIndex,
		StartOffset: anchor.StartOffset,
		EndOffset:   anchor.EndOffset,
	}, color, n)
}

func (s *AnnotationService) UpdateHighlight(ctx context.Context, id int64, color, note string) error {
	var c, n *string
	if color != "" {
		c = ptr(color)
	}
	if note != "" {
		n = ptr(note)
	}
	return s.usecase.UpdateHighlight(ctxFromArgs(ctx), id, c, n)
}

func (s *AnnotationService) RemoveHighlight(ctx context.Context, id int64) error {
	return s.usecase.RemoveHighlight(ctxFromArgs(ctx), id)
}

func (s *AnnotationService) ListHighlights(ctx context.Context, docID int64) ([]HighlightDTO, error) {
	rows, err := s.usecase.ListHighlights(ctxFromArgs(ctx), docID)
	if err != nil {
		return nil, err
	}
	out := make([]HighlightDTO, 0, len(rows))
	for _, h := range rows {
		out = append(out, toHighlightDTO(h))
	}
	return out, nil
}

// ListAllAnnotations merges bookmarks and highlights for the annotations view.
func (s *AnnotationService) ListAllAnnotations(ctx context.Context) (AnnotationsDTO, error) {
	all, err := s.usecase.ListAllAnnotations(ctxFromArgs(ctx), nil)
	if err != nil {
		return AnnotationsDTO{}, err
	}
	out := AnnotationsDTO{}
	for _, b := range all.Bookmarks {
		out.Bookmarks = append(out.Bookmarks, BookmarkDTO{
			ID: b.ID, DocumentID: b.DocumentID, HeadingAnchor: b.HeadingAnchor,
			Note: b.Note, Title: b.Title, RelPath: b.RelPath, SourceName: b.SourceName,
		})
	}
	for _, h := range all.Highlights {
		out.Highlights = append(out.Highlights, HighlightRowDTO{
			ID: h.ID, DocumentID: h.DocumentID, BlockHash: h.BlockHash, BlockIndex: h.BlockIndex,
			StartOffset: h.StartOffset, EndOffset: h.EndOffset, QuotedText: h.QuotedText,
			Color: string(h.Color), Note: h.Note, Title: h.Title, RelPath: h.RelPath, SourceName: h.SourceName,
		})
	}
	return out, nil
}

// AnnotationsDTO is the merged annotations view.
type AnnotationsDTO struct {
	Bookmarks  []BookmarkDTO      `json:"bookmarks"`
	Highlights []HighlightRowDTO  `json:"highlights"`
}

// HighlightRowDTO is a highlight with its document context.
type HighlightRowDTO struct {
	ID          int64  `json:"id"`
	DocumentID  int64  `json:"documentId"`
	BlockHash   string `json:"blockHash"`
	BlockIndex  int    `json:"blockIndex"`
	StartOffset int    `json:"startOffset"`
	EndOffset   int    `json:"endOffset"`
	QuotedText  string `json:"quotedText"`
	Color       string `json:"color"`
	Note        string `json:"note,omitempty"`
	Title       string `json:"title"`
	RelPath     string `json:"relPath"`
	SourceName  string `json:"sourceName"`
}
