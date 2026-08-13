package markdown

import (
	"bytes"
	"fmt"
	"strconv"

	gast "github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast" // table cell kinds
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// blockAnchorRenderer is a goldmark NodeRenderer that emits data-block-hash /
// data-block-index attributes on block-level elements, matching the outline
// anchors on headings. It delegates content rendering to html.Renderer for
// everything it does not override.
type blockAnchorRenderer struct {
	// hashFor holds the block hash for a node, computed in a pre-pass.
	hashFor map[gast.Node]blockAnchor
	// indexFor holds the per-document ordinal assigned in the pre-pass.
	indexFor map[gast.Node]int
	// textLen holds the plain-text length of each block.
	textLen map[gast.Node]int
	// counter assigns block ordinals deterministically.
	counter int
}

type blockAnchor struct {
	hash   string
	index  int
	length int
}

func newBlockAnchorRenderer() *blockAnchorRenderer {
	return &blockAnchorRenderer{
		hashFor:  map[gast.Node]blockAnchor{},
		indexFor: map[gast.Node]int{},
		textLen:  map[gast.Node]int{},
	}
}

// assignBlockAnchors walks the document, assigning a stable hash and ordinal to
// every block-level element that can carry highlights.
func assignBlockAnchors(doc gast.Node, source []byte, r *blockAnchorRenderer) {
	seen := map[string]int{}
	_ = gast.Walk(doc, func(n gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering {
			return assignBlockAnchor(n, source, r, seen), nil
		}
		return gast.WalkContinue, nil
	})
}

func assignBlockAnchor(n gast.Node, source []byte, r *blockAnchorRenderer, seen map[string]int) gast.WalkStatus {
	switch n.(type) {
	case *gast.Paragraph, *gast.Heading, *gast.ListItem, *gast.Blockquote,
		*gast.FencedCodeBlock, *gast.CodeBlock, *extast.TableCell:
		lines := n.Lines()
		if lines != nil && lines.Len() > 0 {
			hash := hashBlockSource(source, lines)
			if hash != "" {
				idx := seen[hash]
				seen[hash] = idx + 1
				r.counter++
				an := blockAnchor{hash: hash, index: idx, length: len(blockPlainText(n, source))}
				r.hashFor[n] = an
				r.indexFor[n] = r.counter
				r.textLen[n] = an.length
			}
		}
	}
	return gast.WalkContinue
}

// RegisterFuncs satisfies renderer.NodeRenderer. We override only the block
// elements that need anchor attributes; everything else falls through to the
// html renderer registered at a lower priority.
func (r *blockAnchorRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(gast.KindParagraph, r.renderAnchored("p"))
	reg.Register(gast.KindHeading, r.renderHeading)
	reg.Register(gast.KindListItem, r.renderAnchored("li"))
	reg.Register(gast.KindBlockquote, r.renderAnchored("blockquote"))
	reg.Register(gast.KindFencedCodeBlock, r.renderCode)
	reg.Register(gast.KindCodeBlock, r.renderCode)
	reg.Register(extast.KindTableCell, r.renderAnchoredCell)
}

// anchorAttrs returns the data-block-hash and data-block-index attribute text
// for a node, or "" when the node has no anchor assigned.
func (r *blockAnchorRenderer) anchorAttrs(n gast.Node) string {
	an, ok := r.hashFor[n]
	if !ok {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteString(` data-block-hash="`)
	buf.WriteString(an.hash)
	buf.WriteString(`" data-block-index="`)
	buf.WriteString(strconv.Itoa(an.index))
	buf.WriteByte('"')
	return buf.String()
}

// renderAnchored renders an opening/closing tag pair with anchor attributes.
func (r *blockAnchorRenderer) renderAnchored(tag string) renderer.NodeRendererFunc {
	return func(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
		if entering {
			_ = w.WriteByte('<')
			_, _ = w.WriteString(tag)
			_, _ = w.WriteString(r.anchorAttrs(n))
			_ = w.WriteByte('>')
			if tag != "p" {
				_ = w.WriteByte('\n')
			}
		} else {
			_, _ = w.WriteString("</")
			_, _ = w.WriteString(tag)
			_, _ = w.WriteString(">\n")
		}
		return gast.WalkContinue, nil
	}
}

// renderHeading writes h1-h6 with the auto id (from outline anchors) and the
// block anchor attributes.
func (r *blockAnchorRenderer) renderHeading(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	hn := n.(*gast.Heading)
	if entering {
		_, _ = w.WriteString("<h")
		_ = w.WriteByte("0123456"[hn.Level])
		if id, ok := hn.AttributeString("id"); ok {
			_, _ = w.WriteString(` id="`)
			if b, ok := id.([]byte); ok {
				_, _ = w.Write(b)
			} else {
				_, _ = fmt.Fprintf(w, "%v", id)
			}
			_ = w.WriteByte('"')
		}
		_, _ = w.WriteString(r.anchorAttrs(n))
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</h")
		_ = w.WriteByte("0123456"[hn.Level])
		_, _ = w.WriteString(">\n")
	}
	return gast.WalkContinue, nil
}

// renderCode writes a fenced code block with anchor attributes and chroma
// highlighting applied at index time.
func (r *blockAnchorRenderer) renderCode(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<pre")
		_, _ = w.WriteString(r.anchorAttrs(n))
		_, _ = w.WriteString("><code")
		if fn, ok := n.(*gast.FencedCodeBlock); ok {
			lang := fn.Language(source)
			if lang != nil {
				_, _ = w.WriteString(` class="language-`)
				_, _ = w.Write(lang)
				_ = w.WriteByte('"')
			}
		}
		_ = w.WriteByte('>')
		writeHighlightedCode(w, source, n)
		_, _ = w.WriteString("</code></pre>\n")
		return gast.WalkSkipChildren, nil
	}
	return gast.WalkContinue, nil
}

// renderAnchoredCell renders th/td with anchor attributes.
func (r *blockAnchorRenderer) renderAnchoredCell(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	tag := "td"
	if n.Parent() != nil && n.Parent().Kind() == extast.KindTableHeader {
		tag = "th"
	}
	if entering {
		_ = w.WriteByte('<')
		_, _ = w.WriteString(tag)
		_, _ = w.WriteString(r.anchorAttrs(n))
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</")
		_, _ = w.WriteString(tag)
		_, _ = w.WriteString(">\n")
	}
	return gast.WalkContinue, nil
}
