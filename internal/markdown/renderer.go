package markdown

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/yuin/goldmark"
	goldmarkmeta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Block is one block-level element's anchor identity: a stable hash of its
// source text plus an index disambiguating identical blocks.
type Block struct {
	Hash       string
	Index      int
	TextLength int
	Text       string
}

// ParseResult is the output of one parse pass: the cached HTML plus everything
// that is a byproduct of the same walk.
type ParseResult struct {
	HTML        string
	PlainText   string
	CodeText    string
	Outline     []domain.OutlineHeading
	HasMermaid  bool
	Blocks      []Block
	Frontmatter map[string]any
	Title       string
	TitleSource domain.TitleSource
	WordCount   int
}

// Parser renders markdown once at index time. Safe for concurrent use.
type Parser struct {
	// build returns a fresh goldmark instance with its own block-anchor
	// renderer. Goldmark renderers keep per-document anchor maps; sharing one
	// instance across goroutines would race on those maps.
	build func() (goldmark.Markdown, *blockAnchorRenderer)
}

// NewParser constructs the markdown parser with GFM, frontmatter, and the
// custom block-anchor renderer.
func NewParser() *Parser {
	return &Parser{build: func() (goldmark.Markdown, *blockAnchorRenderer) {
		anch := newBlockAnchorRenderer()
		md := goldmark.New(
			goldmark.WithExtensions(extension.GFM, goldmarkmeta.Meta),
			goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithHeadingAttribute()),
			goldmark.WithRendererOptions(
				html.WithUnsafe(),
				renderer.WithNodeRenderers(util.Prioritized(anch, 100)),
			),
		)
		return md, anch
	}}
}

// Render parses source into the ParseResult. One goldmark walk produces every
// output: rendered HTML, plain text, code text, outline, block hashes.
func (p *Parser) Render(src []byte) (*ParseResult, error) {
	md, anch := p.build()
	pc := parser.NewContext()
	doc := md.Parser().Parse(text.NewReader(src), parser.WithContext(pc))

	// Assign block anchor identities before rendering so the renderer's lookups
	// hit. The counter must reset per document for deterministic ordinals.
	anch.counter = 0
	assignBlockAnchors(doc, src, anch)

	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, src, doc); err != nil {
		return nil, fmt.Errorf("markdown render: %w", err)
	}

	htmlOut := buf.String()
	if sanitizer != nil {
		htmlOut = sanitizer.Sanitize(htmlOut)
	}

	frontmatter := frontmatterFromContext(pc)
	ex := extractor{}
	ex.walk(doc, src, frontmatter)

	return &ParseResult{
		HTML:        htmlOut,
		PlainText:   ex.plainText.String(),
		CodeText:    ex.codeText.String(),
		Outline:     ex.outline,
		HasMermaid:  ex.hasMermaid,
		Blocks:      ex.blocks,
		Frontmatter: frontmatter,
		Title:       ex.title,
		TitleSource: ex.titleSource,
		WordCount:   countWords(ex.plainText.String()),
	}, nil
}

// countWords returns a rough word count: whitespace-delimited tokens.
func countWords(s string) int {
	if strings.TrimSpace(s) == "" {
		return 0
	}
	return len(strings.Fields(s))
}

// hashBlockSource returns a stable hex sha256 over a block's raw source lines.
func hashBlockSource(src []byte, segments *text.Segments) string {
	if segments == nil {
		return ""
	}
	h := sha256.New()
	segs := segments.Sliced(0, segments.Len())
	for _, seg := range segs {
		_, _ = h.Write(seg.Value(src))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// blockSourceOf reconstructs a block's raw source text from its segments.
func blockSourceOf(src []byte, segments *text.Segments) string {
	if segments == nil {
		return ""
	}
	segs := segments.Sliced(0, segments.Len())
	var sb strings.Builder
	for _, seg := range segs {
		sb.Write(seg.Value(src))
	}
	return sb.String()
}

var _ = ast.KindDocument
