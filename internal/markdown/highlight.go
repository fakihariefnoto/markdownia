package markdown

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/util"
)

// highlightFormatter is the shared chroma formatter producing class-based
// spans (the reading-theme CSS skins the tokens, not the HTML).
var highlightFormatter = html.New(html.WithClasses(true))

// writeHighlightedCode writes a fenced or indented code block to w with chroma
// highlighting applied at index time. Unknown languages degrade to plain text.
func writeHighlightedCode(w util.BufWriter, source []byte, n gast.Node) {
	var lang []byte
	var lines = n.Lines()
	if fn, ok := n.(*gast.FencedCodeBlock); ok {
		lang = fn.Language(source)
	}

	content := blockSourceOf(source, lines)
	if lang == nil {
		_, _ = w.WriteString(escapeHTML(content))
		return
	}

	lexer := lexers.Get(string(lang))
	if lexer == nil {
		lexer = lexers.Fallback
	}
	style := styles.Fallback
	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		_, _ = w.WriteString(escapeHTML(content))
		return
	}
	var buf bytes.Buffer
	if err := highlightFormatter.Format(&buf, style, iterator); err != nil {
		_, _ = w.WriteString(escapeHTML(content))
		return
	}
	_, _ = w.Write(buf.Bytes())
}

// escapeHTML escapes HTML-sensitive characters in code content.
func escapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return r.Replace(s)
}

// ChromaStyleFor returns a chroma style name for a reading-theme family; used
// only by the frontend CSS generation (web task 02) and tests.
func ChromaStyleFor(theme string) string {
	switch theme {
	case "sepia", "paper":
		return "github"
	case "solarized":
		return "solarized-light"
	case "nord":
		return "nord"
	case "dracula":
		return "dracula"
	case "gruvbox":
		return "gruvbox"
	default:
		return "github"
	}
}
