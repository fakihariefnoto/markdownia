package markdown

import (
	goldmarkmeta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/parser"
)

// frontmatterFromContext reads the metadata goldmark-meta stored during parse.
func frontmatterFromContext(pc parser.Context) map[string]any {
	if pc == nil {
		return nil
	}
	m, err := goldmarkmeta.TryGet(pc)
	if err != nil {
		return nil
	}
	if len(m) == 0 {
		return nil
	}
	return m
}
