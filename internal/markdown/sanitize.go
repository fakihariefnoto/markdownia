package markdown

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// sanitizer is the single bluemonday policy applied at index time, before
// caching. The invariant is "nothing unsafe is stored" (ADR A6).
var sanitizer *bluemonday.Policy

func init() {
	p := bluemonday.UGCPolicy()

	// UGCPolicy already strips script/iframe/object and javascript:/data: URLs,
	// but it allows a broad tag set. Keep the tags goldmark actually emits and
	// drop the rest, so the stored HTML is exactly what the renderer produced.
	p.AllowElements(
		"p", "br", "hr",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"ul", "ol", "li",
		"blockquote", "pre", "code",
		"table", "thead", "tbody", "tr", "th", "td",
		"img", "a", "strong", "em", "del", "ins", "sup", "sub", "mark", "span",
	)
	p.AllowAttrs("class").Globally()
	p.AllowAttrs("href", "rel", "target").OnElements("a")
	p.AllowAttrs("src", "alt", "title").OnElements("img")
	p.AllowAttrs("colspan", "rowspan").OnElements("th", "td")
	p.AllowAttrs("id").OnElements("h1", "h2", "h3", "h4", "h5", "h6")
	// The block anchor contract: the reading pane resolves any text node to a
	// block via these attributes, and the sweep matches on them.
	p.AllowAttrs("data-block-hash", "data-block-index").OnElements(
		"p", "li", "blockquote", "pre", "td", "th",
	)
	// Chroma's class-based highlighting output.
	p.AllowAttrs("class").OnElements("span")

	sanitizer = p
}

// Sanitize strips any markup not permitted by the index-time policy.
func Sanitize(html string) string {
	return sanitizer.Sanitize(html)
}

// StripExportAttrs removes the data-block-* attributes from a fragment. Used
// by the standalone HTML exporter so exported files do not leak anchor
// metadata, and by PDF export so the print stylesheet stays clean.
func StripExportAttrs(html string) string {
	return stripDataAttrs(html)
}

// stripDataAttrs removes the data-block-* attributes from a fragment.
func stripDataAttrs(html string) string {
	var b strings.Builder
	idx := 0
	for {
		start := strings.Index(html[idx:], "data-block-")
		if start < 0 {
			b.WriteString(html[idx:])
			break
		}
		start += idx
		// find end of attribute value
		end := strings.IndexByte(html[start:], '"')
		if end < 0 {
			b.WriteString(html[idx:])
			break
		}
		endQuote := start + end + 1
		attrEnd := endQuote + 1
		// swallow a trailing space if present
		if attrEnd < len(html) && html[attrEnd] == ' ' {
			attrEnd++
		}
		b.WriteString(html[idx:start])
		idx = attrEnd
	}
	return b.String()
}
