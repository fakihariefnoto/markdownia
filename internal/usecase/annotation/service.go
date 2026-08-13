// Package annotation implements bookmarks and highlights, including the
// block-hash anchor validation that keeps the webview (the untrusted side of
// the boundary) honest.
package annotation

import (
	"context"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/repository/sqlite"
)

// Repo is the annotation usecase's persistence view, declared here.
type Repo interface {
	AddBookmark(ctx context.Context, b *domain.Bookmark) (int64, error)
	RemoveBookmark(ctx context.Context, id int64) error
	ListBookmarks(ctx context.Context) ([]sqlite.BookmarkRow, error)
	ListBookmarksBySource(ctx context.Context, sourceID int64) ([]sqlite.BookmarkRow, error)

	AddHighlight(ctx context.Context, h *domain.Highlight) (int64, error)
	UpdateHighlight(ctx context.Context, id int64, color *string, note *string) error
	RemoveHighlight(ctx context.Context, id int64) error
	ListHighlights(ctx context.Context, docID int64) ([]domain.Highlight, error)
	ListAllHighlights(ctx context.Context) ([]sqlite.HighlightRow, error)
	ListHighlightsBySource(ctx context.Context, sourceID int64) ([]sqlite.HighlightRow, error)
	HighlightByID(ctx context.Context, id int64) (domain.Highlight, error)
}

// DocumentRepo validates highlight anchors against the document's stored
// blocks and extracts the quoted text.
type DocumentRepo interface {
	GetByID(ctx context.Context, id int64) (domain.Document, error)
}

// Service is the annotation usecase.
type Service struct {
	repo     Repo
	docRepo  DocumentRepo
	blocks   BlockSource
}

// BlockSource provides the block hashes and lengths for a document, so an
// anchor can be validated against reality.
type BlockSource interface {
	DocumentBlocks(ctx context.Context, docID int64) ([]Block, error)
}

// Block mirrors markdown.Block for the anchor validator.
type Block struct {
	Hash       string
	Index      int
	TextLength int
	Text       string
}

// New constructs the annotation usecase.
func New(repo Repo, docRepo DocumentRepo, blocks BlockSource) *Service {
	return &Service{repo: repo, docRepo: docRepo, blocks: blocks}
}

// ---- Bookmarks ----

func (s *Service) AddBookmark(ctx context.Context, docID int64, headingAnchor, note *string) (int64, error) {
	ha, n := "", ""
	if headingAnchor != nil {
		ha = *headingAnchor
	}
	if note != nil {
		n = *note
	}
	return s.repo.AddBookmark(ctx, &domain.Bookmark{
		DocumentID:    docID,
		HeadingAnchor: ha,
		Note:          n,
	})
}

func (s *Service) RemoveBookmark(ctx context.Context, id int64) error {
	return s.repo.RemoveBookmark(ctx, id)
}

// ListBookmarks returns bookmarks, optionally scoped to a source.
func (s *Service) ListBookmarks(ctx context.Context, sourceID *int64) ([]sqlite.BookmarkRow, error) {
	if sourceID != nil {
		return s.repo.ListBookmarksBySource(ctx, *sourceID)
	}
	return s.repo.ListBookmarks(ctx)
}

// ---- Highlights ----

// AddHighlight validates the frontend-computed anchor Go-side before storing.
// The webview is the untrusted side of the boundary: a hash not present in the
// document, an out-of-range block index, or offsets past the block length are
// rejected at creation, since such an anchor would be swept at the next index.
func (s *Service) AddHighlight(ctx context.Context, docID int64, anchor domain.HighlightAnchor, color string, note *string) (int64, error) {
	if !domain.IsValidHighlightColor(color) {
		return 0, domain.ErrInvalidColor
	}
	if anchor.BlockHash == "" {
		return 0, domain.ErrInvalidAnchor
	}

	blocks, err := s.blocks.DocumentBlocks(ctx, docID)
	if err != nil {
		return 0, err
	}

	var matched *Block
	for i := range blocks {
		b := &blocks[i]
		if b.Hash == anchor.BlockHash && b.Index == anchor.BlockIndex {
			matched = b
			break
		}
	}
	if matched == nil {
		return 0, domain.ErrInvalidAnchor
	}
	if anchor.StartOffset < 0 || anchor.EndOffset > matched.TextLength || anchor.StartOffset >= anchor.EndOffset {
		return 0, domain.ErrInvalidAnchor
	}

	quoted := quotedText(blocks, anchor)
	n := ""
	if note != nil {
		n = *note
	}
	return s.repo.AddHighlight(ctx, &domain.Highlight{
		DocumentID:  docID,
		BlockHash:   anchor.BlockHash,
		BlockIndex:  anchor.BlockIndex,
		StartOffset: anchor.StartOffset,
		EndOffset:   anchor.EndOffset,
		QuotedText:  quoted,
		Color:       domain.HighlightColor(color),
		Note:        n,
	})
}

// quotedText extracts the selected substring from the block's plain text.
func quotedText(blocks []Block, anchor domain.HighlightAnchor) string {
	for _, b := range blocks {
		if b.Hash == anchor.BlockHash && b.Index == anchor.BlockIndex {
			if anchor.StartOffset >= 0 && anchor.EndOffset <= b.TextLength && anchor.EndOffset <= len(b.Text) {
				return b.Text[anchor.StartOffset:anchor.EndOffset]
			}
			return "…"
		}
	}
	return "…"
}

func (s *Service) UpdateHighlight(ctx context.Context, id int64, color *string, note *string) error {
	if color != nil && !domain.IsValidHighlightColor(*color) {
		return domain.ErrInvalidColor
	}
	return s.repo.UpdateHighlight(ctx, id, color, note)
}

func (s *Service) RemoveHighlight(ctx context.Context, id int64) error {
	return s.repo.RemoveHighlight(ctx, id)
}

// ListHighlights returns a document's highlights.
func (s *Service) ListHighlights(ctx context.Context, docID int64) ([]domain.Highlight, error) {
	return s.repo.ListHighlights(ctx, docID)
}

// ListAllAnnotations merges bookmarks and highlights for the annotations view,
// optionally scoped to a source.
func (s *Service) ListAllAnnotations(ctx context.Context, sourceID *int64) (Annotations, error) {
	bm, err := s.ListBookmarks(ctx, sourceID)
	if err != nil {
		return Annotations{}, err
	}
	var hl []sqlite.HighlightRow
	if sourceID != nil {
		hl, err = s.repo.ListHighlightsBySource(ctx, *sourceID)
	} else {
		hl, err = s.repo.ListAllHighlights(ctx)
	}
	if err != nil {
		return Annotations{}, err
	}
	return Annotations{Bookmarks: bm, Highlights: hl}, nil
}

// Annotations is the merged view for the annotations screen.
type Annotations struct {
	Bookmarks  []sqlite.BookmarkRow  `json:"bookmarks"`
	Highlights []sqlite.HighlightRow `json:"highlights"`
}
