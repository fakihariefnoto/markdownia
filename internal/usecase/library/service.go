// Package library implements the reading hot path: navigation tree, document
// open (cached HTML + outline + highlights in one call), link resolution, and
// relative asset serving.
package library

import (
	"context"
	"fmt"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/markdown"
	"github.com/anofac/markdownia/internal/pathguard"
)

// Repo is the library usecase's persistence view.
type Repo interface {
	GetByID(ctx context.Context, id int64) (domain.Document, error)
	MetaByID(ctx context.Context, id int64) (domain.Document, error)
	GetBySourceAndPath(ctx context.Context, sourceID int64, relPath string) (domain.Document, error)
	ListBySource(ctx context.Context, sourceID int64) ([]domain.Document, error)
	ReRenderHTML(ctx context.Context, id int64, html string, version int) error
}

// SourceRepo provides source roots for asset serving and link resolution.
type SourceRepo interface {
	GetByID(ctx context.Context, id int64) (domain.Source, error)
}

// AnnotationRepo supplies highlights for the open-document payload.
type AnnotationRepo interface {
	ListHighlights(ctx context.Context, docID int64) ([]domain.Highlight, error)
}

// ReadingRepo records reading state and history on open.
type ReadingRepo interface {
	SetDocument(ctx context.Context, ctxType domain.ContextType, ctxID int64, docID int64) error
	RecordOpen(ctx context.Context, docID int64, scrollPct float64) error
	ListRecent(ctx context.Context, limit int64) ([]domain.ReadingHistory, error)
}

// MetaRepo provides recent-document titles for the home screen.
type MetaRepo interface {
	MetaByID(ctx context.Context, id int64) (domain.Document, error)
}

// OpenPayload is the hot-path response: everything the reader needs in one call.
type OpenPayload struct {
	Document   domain.Document
	Highlights []domain.Highlight
}

// Service is the library usecase.
type Service struct {
	repo        Repo
	sources     SourceRepo
	annotations AnnotationRepo
	reading     ReadingRepo
	meta        MetaRepo
	parser      *markdown.Parser
}

// New constructs the library usecase.
func New(repo Repo, sources SourceRepo, annotations AnnotationRepo, reading ReadingRepo, meta MetaRepo, parser *markdown.Parser) *Service {
	return &Service{repo: repo, sources: sources, annotations: annotations, reading: reading, meta: meta, parser: parser}
}

// GetTree returns the folder hierarchy for a source, shaped for virtualized
// rendering (a flat list of tree nodes; the frontend derives nesting).
func (s *Service) GetTree(ctx context.Context, sourceID int64) ([]TreeNode, error) {
	docs, err := s.repo.ListBySource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	nodes := make([]TreeNode, 0, len(docs))
	for _, d := range docs {
		nodes = append(nodes, TreeNode{
			ID:       d.ID,
			RelPath:  d.RelPath,
			Title:    d.Title,
			Depth:    depthOf(d.RelPath),
			IsFolder: false,
		})
	}
	// Inject folder nodes.
	return buildTree(nodes), nil
}

// TreeNode is one entry in the source tree.
type TreeNode struct {
	ID       int64  `json:"id"`
	RelPath  string `json:"relPath"`
	Title    string `json:"title"`
	Depth    int    `json:"depth"`
	IsFolder bool   `json:"isFolder"`
}

func depthOf(rel string) int {
	return strings.Count(rel, "/")
}

func buildTree(docs []TreeNode) []TreeNode {
	// Depth-first pre-order: each folder is immediately followed by its
	// descendants, so the frontend can render a collapsible hierarchy from the
	// flat list by tracking open folder paths.
	// Collect folder paths.
	folderSet := map[string]bool{}
	for _, d := range docs {
		dir := filepath.Dir(d.RelPath)
		for dir != "." {
			folderSet[dir] = true
			dir = filepath.Dir(dir)
		}
	}

	// docsByDir: docs grouped by their parent directory.
	docsByDir := map[string][]TreeNode{}
	for _, d := range docs {
		p := filepath.Dir(d.RelPath)
		docsByDir[p] = append(docsByDir[p], d)
	}
	for p := range docsByDir {
		sort.Slice(docsByDir[p], func(i, j int) bool {
			return docsByDir[p][i].RelPath < docsByDir[p][j].RelPath
		})
	}

	// childrenFolders: direct child folders of each directory.
	childrenFolders := map[string][]string{}
	for f := range folderSet {
		p := filepath.Dir(f)
		childrenFolders[p] = append(childrenFolders[p], f)
	}
	for p := range childrenFolders {
		sort.Strings(childrenFolders[p])
	}

	var out []TreeNode
	var walk func(dir string)
	walk = func(dir string) {
		for _, f := range childrenFolders[dir] {
			out = append(out, TreeNode{RelPath: f, Title: filepath.Base(f), Depth: depthOf(f), IsFolder: true})
			walk(f)
		}
		out = append(out, docsByDir[dir]...)
	}
	walk(".")
	return out
}

// OpenDocument is the hot path: cached HTML, outline, and highlights in one
// call, recording reading state and history. ≤100ms at the reference size.
func (s *Service) OpenDocument(ctx context.Context, docID int64, ctxType domain.ContextType, ctxID int64) (OpenPayload, error) {
	doc, err := s.repo.GetByID(ctx, docID)
	if err != nil {
		return OpenPayload{}, err
	}

	// Renderer-version mismatch: re-render on demand (once per doc per upgrade).
	if doc.RendererVersion != markdown.RendererVersion {
		if reRendered, err := s.reRender(ctx, doc); err == nil {
			doc = reRendered
		}
	}

	highlights, err := s.annotations.ListHighlights(ctx, docID)
	if err != nil {
		return OpenPayload{}, err
	}

	_ = s.reading.SetDocument(ctx, ctxType, ctxID, docID)
	_ = s.reading.RecordOpen(ctx, docID, 0)

	return OpenPayload{Document: doc, Highlights: highlights}, nil
}

// reRender re-parses the document's source file and rewrites its cached HTML
// and renderer version. Recomputing from identical source means block hashes
// match — highlights are safe (domains.md §5.1).
func (s *Service) reRender(ctx context.Context, doc domain.Document) (domain.Document, error) {
	src, err := s.sources.GetByID(ctx, doc.SourceID)
	if err != nil {
		return doc, err
	}
	abs := pathguard.Join(src.RootPath, doc.RelPath)
	if abs == "" {
		return doc, domain.ErrPathEscapesRoot
	}
// #nosec G304 -- abs was produced by pathguard.Join against the source root.
	// #nosec G304 -- abs was produced by pathguard.Join against the source root.
	content, err := os.ReadFile(abs)
	if err != nil {
		return doc, fmt.Errorf("re-render source: %w", err)
	}
	res, err := s.parser.Render(content)
	if err != nil {
		return doc, err
	}
	if err := s.repo.ReRenderHTML(ctx, doc.ID, res.HTML, markdown.RendererVersion); err != nil {
		return doc, err
	}
	doc.RenderedHTML = res.HTML
	doc.RendererVersion = markdown.RendererVersion
	return doc, nil
}

// GetDocumentMeta returns title/path/frontmatter without the HTML payload.
func (s *Service) GetDocumentMeta(ctx context.Context, docID int64) (domain.Document, error) {
	return s.repo.MetaByID(ctx, docID)
}

// ResolveLink decides internal-navigation vs external-browser for a link.
func (s *Service) ResolveLink(ctx context.Context, fromDocID int64, href string) (LinkTarget, error) {
	if href == "" {
		return LinkTarget{}, domain.ErrInvalidArgument
	}
	from, err := s.repo.MetaByID(ctx, fromDocID)
	if err != nil {
		return LinkTarget{}, err
	}
	src, err := s.sources.GetByID(ctx, from.SourceID)
	if err != nil {
		return LinkTarget{}, err
	}

	// Fragment-only: internal, same document.
	if strings.HasPrefix(href, "#") {
		return LinkTarget{Internal: true, DocumentID: fromDocID, Anchor: strings.TrimPrefix(href, "#")}, nil
	}

	// Absolute web URL: external.
	if u, err := url.Parse(href); err == nil && (u.Scheme == "http" || u.Scheme == "https" || u.Scheme == "mailto") {
		return LinkTarget{External: true, URL: href}, nil
	}

	// Relative path: resolve against the source root.
	target := pathguard.Join(src.RootPath, filepath.Dir(from.RelPath)+"/"+href)
	if target == "" {
		return LinkTarget{}, domain.ErrPathEscapesRoot
	}
	rel, err := filepath.Rel(src.RootPath, target)
	if err != nil {
		return LinkTarget{}, err
	}
	rel = filepath.ToSlash(rel)

	// Strip anchor from the file path portion.
	anchor := ""
	if i := strings.IndexByte(rel, '#'); i >= 0 {
		anchor = rel[i+1:]
		rel = rel[:i]
	}

	// A relative link with no extension pointing at an existing directory is a
	// folder target: the frontend expands and highlights it in the source tree.
	if rel != "" && filepath.Ext(rel) == "" {
		dir := pathguard.Join(src.RootPath, rel)
		if dir != "" {
			if info, err := os.Stat(dir); err == nil && info.IsDir() {
				return LinkTarget{Internal: true, Folder: true, FolderRel: rel, SourceID: from.SourceID}, nil
			}
		}
	}

	if !domain.IsMarkdownExtension(filepath.Ext(rel)) {
		// Not a markdown document — treat as external (image/asset, opened by
		// the reading pane via GetAsset).
		return LinkTarget{Internal: false, External: false, AssetPath: rel, SourceID: from.SourceID}, nil
	}

	targetDoc, err := s.repo.GetBySourceAndPath(ctx, from.SourceID, rel)
	if err != nil {
		if err == domain.ErrDocumentNotFound {
			return LinkTarget{}, nil // not in library — typed result, not an error
		}
		return LinkTarget{}, err
	}
	return LinkTarget{Internal: true, DocumentID: targetDoc.ID, Anchor: anchor}, nil
}

// LinkTarget is the resolution of a link.
type LinkTarget struct {
	Internal   bool   `json:"internal"`
	External   bool   `json:"external"`
	DocumentID int64  `json:"documentId,omitempty"`
	Anchor     string `json:"anchor,omitempty"`
	URL        string `json:"url,omitempty"`
	AssetPath  string `json:"assetPath,omitempty"`
	SourceID   int64  `json:"sourceId,omitempty"`
	Folder     bool   `json:"folder,omitempty"`
	FolderRel  string `json:"folderRel,omitempty"`
}

// GetAsset returns a relative image's bytes and content type, path-guarded to
// the source root. Only images are served.
func (s *Service) GetAsset(ctx context.Context, docID int64, relPath string) ([]byte, string, error) {
	doc, err := s.repo.MetaByID(ctx, docID)
	if err != nil {
		return nil, "", err
	}
	src, err := s.sources.GetByID(ctx, doc.SourceID)
	if err != nil {
		return nil, "", err
	}

	ctype := mime.TypeByExtension(filepath.Ext(relPath))
	if !strings.HasPrefix(ctype, "image/") {
		return nil, "", domain.ErrAssetNotFound
	}

	abs := pathguard.Join(src.RootPath, filepath.Dir(doc.RelPath)+"/"+relPath)
	if abs == "" {
		return nil, "", domain.ErrAssetNotFound
	}
	// #nosec G304 -- abs was produced by pathguard.Join against the source root.
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, "", domain.ErrAssetNotFound
	}
	return data, ctype, nil
}

// ListRecent returns recently-read documents with their titles.
func (s *Service) ListRecent(ctx context.Context, limit int64) ([]RecentDoc, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	history, err := s.reading.ListRecent(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]RecentDoc, 0, len(history))
	for _, h := range history {
		doc, err := s.meta.MetaByID(ctx, h.DocumentID)
		if err != nil {
			continue
		}
		out = append(out, RecentDoc{
			DocumentID:        h.DocumentID,
			Title:             doc.Title,
			RelPath:           doc.RelPath,
			SourceID:          doc.SourceID,
			LastOpenedAt:      h.LastOpenedAt,
			FurthestScrollPct: h.FurthestScrollPct,
		})
	}
	return out, nil
}

// RecentDoc is one entry in the recently-read list.
type RecentDoc struct {
	DocumentID        int64   `json:"documentId"`
	Title             string  `json:"title"`
	RelPath           string  `json:"relPath"`
	SourceID          int64   `json:"sourceId"`
	LastOpenedAt      string  `json:"lastOpenedAt"`
	FurthestScrollPct float64 `json:"furthestScrollPct"`
}
