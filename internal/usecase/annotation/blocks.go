package annotation

import (
	"context"
	"os"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/markdown"
	"github.com/anofac/markdownia/internal/pathguard"
)

// DocSource resolves a document's source root so blocks can be re-derived from
// the file.
type DocSource interface {
	GetByID(ctx context.Context, id int64) (domain.Source, error)
}

// MarkdownBlocks is a BlockSource that re-derives block identities from the
// document's source file. The frontend computes anchors from the rendered DOM;
// Go validates them against this same derivation.
type MarkdownBlocks struct {
	docs   DocumentRepo
	source DocSource
	parser *markdown.Parser
}

// NewMarkdownBlocks constructs the block source.
func NewMarkdownBlocks(docs DocumentRepo, source DocSource, parser *markdown.Parser) *MarkdownBlocks {
	return &MarkdownBlocks{docs: docs, source: source, parser: parser}
}

// DocumentBlocks re-derives the document's current block identities.
func (m *MarkdownBlocks) DocumentBlocks(ctx context.Context, docID int64) ([]Block, error) {
	doc, err := m.docs.GetByID(ctx, docID)
	if err != nil {
		return nil, err
	}
	src, err := m.source.GetByID(ctx, doc.SourceID)
	if err != nil {
		return nil, err
	}
	abs := pathguard.Join(src.RootPath, doc.RelPath)
	if abs == "" {
		return nil, domain.ErrPathEscapesRoot
	}
	// #nosec G304 -- abs was produced by pathguard.Join against the source root.
	content, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	res, err := m.parser.Render(content)
	if err != nil {
		return nil, err
	}
	out := make([]Block, 0, len(res.Blocks))
	for _, b := range res.Blocks {
		out = append(out, Block{
			Hash: b.Hash, Index: b.Index, TextLength: b.TextLength, Text: b.Text,
		})
	}
	return out, nil
}
