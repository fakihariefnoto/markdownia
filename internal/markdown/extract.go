package markdown

import (
	"strings"

	"github.com/anofac/markdownia/internal/domain"
	gast "github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// extractor walks the goldmark AST once to produce every non-HTML output of a
// parse: heading outline, plain text (code excluded), code text, block hashes,
// mermaid detection, and title resolution.
type extractor struct {
	plainText   strings.Builder
	codeText    strings.Builder
	outline     []domain.OutlineHeading
	hasMermaid  bool
	blocks      []Block
	blockSeen   map[string]int // hash -> count of identical blocks
	title       string
	titleSource domain.TitleSource
	titleFound  bool
	inParagraph bool
}

func (e *extractor) walk(doc gast.Node, source []byte, frontmatter map[string]any) {
	e.blockSeen = map[string]int{}

	if t, ok := frontmatter["title"]; ok {
		if s, ok := t.(string); ok && strings.TrimSpace(s) != "" {
			e.title = s
			e.titleSource = domain.TitleFrontmatter
			e.titleFound = true
		}
	}

	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		switch node := n.(type) {
		case *gast.Paragraph:
			if entering {
				e.recordBlock(node, source)
				e.inParagraph = true
			} else {
				e.inParagraph = false
				e.plainText.WriteByte('\n')
			}
			return gast.WalkContinue, nil

		case *gast.Heading:
			if !entering {
				return gast.WalkContinue, nil
			}
			e.recordBlock(node, source)
			if !e.titleFound {
				t := headingText(node, source)
				if t != "" {
					e.title = t
					e.titleSource = domain.TitleH1
					e.titleFound = true
				}
			}
			e.outline = append(e.outline, domain.OutlineHeading{
				Level:      node.Level,
				Text:       headingText(node, source),
				Anchor:     headingAnchor(node),
				BlockIndex: len(e.blocks),
			})
			e.plainText.WriteString(headingText(node, source))
			e.plainText.WriteByte('\n')
			return gast.WalkContinue, nil

		case *gast.Blockquote:
			if entering {
				e.recordBlock(node, source)
			}
			return gast.WalkContinue, nil

		case *gast.ListItem:
			if entering {
				e.recordBlock(node, source)
			}
			return gast.WalkContinue, nil

		case *gast.FencedCodeBlock:
			if !entering {
				return gast.WalkContinue, nil
			}
			e.recordBlock(node, source)
			if isMermaidFence(node, source) {
				e.hasMermaid = true
			}
			e.writeCodeLines(node.Lines(), source)
			e.plainText.WriteByte('\n')
			return gast.WalkContinue, nil

		case *gast.CodeBlock:
			if !entering {
				return gast.WalkContinue, nil
			}
			e.recordBlock(node, source)
			e.writeCodeLines(node.Lines(), source)
			e.plainText.WriteByte('\n')
			return gast.WalkContinue, nil

		case *gast.Text:
			if entering {
				e.plainText.Write(node.Segment.Value(source))
				if node.SoftLineBreak() {
					e.plainText.WriteByte(' ')
				}
			}
			return gast.WalkContinue, nil

		case *gast.String:
			if entering {
				e.plainText.Write(node.Value)
			}
			return gast.WalkContinue, nil

		case *gast.CodeSpan:
			// CodeSpan's text lives in child Text/String nodes, walked below.
			return gast.WalkContinue, nil

		case *extast.TableCell:
			if entering {
				e.recordBlock(node, source)
			}
			return gast.WalkContinue, nil

		case *gast.Image, *gast.Link:
			// Child text contributes to plain text via Text nodes.
			return gast.WalkContinue, nil
		}
		return gast.WalkContinue, nil
	})

	if e.title == "" {
		e.title = "Untitled"
		e.titleSource = domain.TitleFilename
	}
}

// recordBlock assigns a stable block identity for any block-level node that
// can contain highlightable text.
func (e *extractor) recordBlock(n gast.Node, source []byte) {
	lines := n.Lines()
	if lines == nil || lines.Len() == 0 {
		return
	}
	hash := hashBlockSource(source, lines)
	if hash == "" {
		return
	}
	idx := e.blockSeen[hash]
	e.blockSeen[hash] = idx + 1

	plain := blockPlainText(n, source)
	e.blocks = append(e.blocks, Block{
		Hash:       hash,
		Index:      idx,
		TextLength: len(plain),
		Text:       plain,
	})
}

// blockPlainText extracts the plain text of a block node from its child text
// segments, matching what the reading pane sees in the DOM.
func blockPlainText(n gast.Node, source []byte) string {
	var sb strings.Builder
	_ = gast.Walk(n, func(c gast.Node, entering bool) (gast.WalkStatus, error) {
		if !entering {
			return gast.WalkContinue, nil
		}
		switch node := c.(type) {
		case *gast.Text:
			sb.Write(node.Segment.Value(source))
			if node.SoftLineBreak() {
				sb.WriteByte(' ')
			}
		case *gast.String:
			sb.Write(node.Value)
		case *gast.CodeSpan:
			// handled via child Text nodes
		}
		return gast.WalkContinue, nil
	})
	return sb.String()
}

// writeCodeLines appends code-fence contents to the code_text builder.
func (e *extractor) writeCodeLines(lines *text.Segments, source []byte) {
	if lines == nil {
		return
	}
	segs := lines.Sliced(0, lines.Len())
	for _, seg := range segs {
		e.codeText.Write(seg.Value(source))
	}
}

func headingText(n *gast.Heading, source []byte) string {
	return strings.TrimSpace(blockPlainText(n, source))
}

// headingAnchor returns the anchor value for a heading. When auto-heading-id
// is enabled goldmark stores an id attribute; fall back to a slug.
func headingAnchor(n *gast.Heading) string {
	if id, ok := n.AttributeString("id"); ok {
		if s, ok := id.([]byte); ok && len(s) > 0 {
			return string(s)
		}
	}
	return ""
}

func isMermaidFence(n *gast.FencedCodeBlock, source []byte) bool {
	lang := n.Language(source)
	if lang == nil {
		return false
	}
	return strings.EqualFold(string(lang), "mermaid")
}
