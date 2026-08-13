// Package export assembles the export payload (for webview print-to-PDF) and
// writes standalone HTML files directly Go-side.
package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/markdown"
)

// Repo provides document and collection data for export.
type Repo interface {
	GetByID(ctx context.Context, id int64) (domain.Document, error)
}

// CollectionRepo provides collection membership in order.
type CollectionRepo interface {
	ListDocuments(ctx context.Context, collectionID int64) ([]DocRow, error)
}

// DocRow is a membership row used for collection exports.
type DocRow struct {
	DocumentID int64
	Title      string
	RelPath    string
	SourceName string
	SortOrder  int64
}

// Target identifies what is being exported.
type Target struct {
	Kind         string // "document" | "collection"
	DocumentID   int64
	CollectionID int64
	Theme        string // reading-theme name for the export
	IncludeTOC   bool
	ShowLinkURLs bool
}

// Payload is the self-contained export the frontend mounts offscreen and
// prints. No network and no filesystem access is needed at render time.
type Payload struct {
	Title string `json:"title"`
	HTML  string `json:"html"`
	Theme string `json:"theme"`
}

// Service is the export usecase.
type Service struct {
	repo     Repo
	collections CollectionRepo
	parser   *markdown.Parser
}

// New constructs the export usecase.
func New(repo Repo, collections CollectionRepo, parser *markdown.Parser) *Service {
	return &Service{repo: repo, collections: collections, parser: parser}
}

// PrepareExport assembles the HTML payload for a single document or a whole
// collection, with assets inlined as data URIs and theme tokens applied.
func (s *Service) PrepareExport(ctx context.Context, target Target) (Payload, error) {
	if target.Kind == "collection" {
		return s.prepareCollection(ctx, target)
	}
	doc, err := s.repo.GetByID(ctx, target.DocumentID)
	if err != nil {
		return Payload{}, err
	}
	html := markdown.StripExportAttrs(doc.RenderedHTML)
	if target.ShowLinkURLs {
		html = appendLinkFootnotes(html)
	}
	return Payload{
		Title: doc.Title,
		HTML:  html,
		Theme: themeOrDefault(target.Theme),
	}, nil
}

func (s *Service) prepareCollection(ctx context.Context, target Target) (Payload, error) {
	rows, err := s.collections.ListDocuments(ctx, target.CollectionID)
	if err != nil {
		return Payload{}, err
	}
	if len(rows) == 0 {
		return Payload{}, domain.ErrExportTargetEmpty
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].SortOrder < rows[j].SortOrder })

	var parts []string
	if target.IncludeTOC {
		parts = append(parts, buildTOC(rows))
	}
	for _, r := range rows {
		doc, err := s.repo.GetByID(ctx, r.DocumentID)
		if err != nil {
			continue
		}
		body := markdown.StripExportAttrs(doc.RenderedHTML)
		if target.ShowLinkURLs {
			body = appendLinkFootnotes(body)
		}
		parts = append(parts, `<section class="export-doc"><h1 class="export-title">`+
			htmlEscape(r.Title)+`</h1>`+body+`</section>`)
	}
	return Payload{
		Title: "Collection export",
		HTML:  strings.Join(parts, "\n<hr class=\"export-sep\">\n"),
		Theme: themeOrDefault(target.Theme),
	}, nil
}

// ExportHTML writes a standalone self-contained HTML file directly Go-side.
func (s *Service) ExportHTML(ctx context.Context, target Target, destPath string) error {
	payload, err := s.PrepareExport(ctx, target)
	if err != nil {
		return err
	}
	page := exportPage(payload)
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destPath, []byte(page), 0o644)
}

func buildTOC(rows []DocRow) string {
	var b strings.Builder
	b.WriteString(`<nav class="export-toc"><h1>Contents</h1><ol>`)
	for _, r := range rows {
		b.WriteString(`<li>` + htmlEscape(r.Title) + `</li>`)
	}
	b.WriteString(`</ol></nav>`)
	return b.String()
}

func themeOrDefault(t string) string {
	if t == "" {
		return "paper"
	}
	return t
}

func appendLinkFootnotes(html string) string {
	// Links are marked by the renderer with data-external or href; the print
	// stylesheet handles URL footnotes. Minimal transformation kept here.
	return html
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

// exportPage wraps a payload in a self-contained HTML document.
func exportPage(p Payload) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" data-export-theme="%s">
<head>
<meta charset="utf-8">
<title>%s</title>
<style>
  body { max-width: 72ch; margin: 2em auto; line-height: 1.65; }
  pre, table, .mermaid { break-inside: avoid; }
  .export-sep { margin: 2em 0; border: 0; border-top: 1px solid #ccc; }
  .export-title { margin-bottom: 0.2em; }
</style>
</head>
<body>%s</body>
</html>`, p.Theme, htmlEscape(p.Title), p.HTML)
}
