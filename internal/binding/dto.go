package binding

import (
	"github.com/anofac/markdownia/internal/domain"
)

// SourceDTO crosses the bridge for sources.
type SourceDTO struct {
	ID            int64              `json:"id"`
	Kind          string             `json:"kind"`
	Name          string             `json:"name"`
	RootPath      string             `json:"rootPath"`
	OriginURL     string             `json:"originUrl,omitempty"`
	GitBranch     string             `json:"gitBranch,omitempty"`
	GitCommit     string             `json:"gitCommit,omitempty"`
	IsManaged     bool               `json:"isManaged"`
	Status        string             `json:"status"`
	ErrorMessage  string             `json:"errorMessage,omitempty"`
	DocumentCount int64              `json:"documentCount"`
	IgnoreGlobs   []string           `json:"ignoreGlobs,omitempty"`
	IndexedAt     string             `json:"indexedAt,omitempty"`
}

// DeletionPreviewDTO powers the source-delete confirm dialog.
type DeletionPreviewDTO struct {
	Documents         int64 `json:"documents"`
	Highlights        int64 `json:"highlights"`
	Bookmarks         int64 `json:"bookmarks"`
	CollectionEntries int64 `json:"collectionEntries"`
	DeletesFilesOnDisk bool  `json:"deletesFilesOnDisk"`
}

// DocumentDTO is the reader's hot-path payload.
type DocumentDTO struct {
	ID          int64    `json:"id"`
	SourceID    int64    `json:"sourceId"`
	RelPath     string   `json:"relPath"`
	Title       string   `json:"title"`
	RenderedHTML string  `json:"renderedHtml"`
	Outline     []OutlineEntryDTO `json:"outline"`
	Highlights  []HighlightDTO    `json:"highlights"`
	WordCount   int64    `json:"wordCount"`
	HasMermaid  bool     `json:"hasMermaid"`
	MissingFile bool     `json:"missingFile,omitempty"`
}

// OutlineEntryDTO is one heading in a document's outline.
type OutlineEntryDTO struct {
	Level      int    `json:"level"`
	Text       string `json:"text"`
	Anchor     string `json:"anchor"`
	BlockIndex int    `json:"blockIndex"`
}

// DocumentMetaDTO is the lightweight document metadata.
type DocumentMetaDTO struct {
	ID       int64  `json:"id"`
	SourceID int64  `json:"sourceId"`
	RelPath  string `json:"relPath"`
	Title    string `json:"title"`
	WordCount int64 `json:"wordCount"`
}

// HighlightDTO is one highlight crossing the bridge.
type HighlightDTO struct {
	ID          int64  `json:"id"`
	DocumentID  int64  `json:"documentId"`
	BlockHash   string `json:"blockHash"`
	BlockIndex  int    `json:"blockIndex"`
	StartOffset int    `json:"startOffset"`
	EndOffset   int    `json:"endOffset"`
	QuotedText  string `json:"quotedText"`
	Color       string `json:"color"`
	Note        string `json:"note,omitempty"`
}

// BookmarkDTO is one bookmark crossing the bridge.
type BookmarkDTO struct {
	ID            int64  `json:"id"`
	DocumentID    int64  `json:"documentId"`
	HeadingAnchor string `json:"headingAnchor,omitempty"`
	Note          string `json:"note,omitempty"`
	Title         string `json:"title"`
	RelPath       string `json:"relPath"`
	SourceName    string `json:"sourceName"`
}

// AnchorDTO is the frontend-computed highlight anchor.
type AnchorDTO struct {
	BlockHash   string `json:"blockHash"`
	BlockIndex  int    `json:"blockIndex"`
	StartOffset int    `json:"startOffset"`
	EndOffset   int    `json:"endOffset"`
}

// LinkTargetDTO is the resolution of a link click.
type LinkTargetDTO struct {
	Internal   bool   `json:"internal"`
	External   bool   `json:"external"`
	DocumentID int64  `json:"documentId,omitempty"`
	Anchor     string `json:"anchor,omitempty"`
	URL        string `json:"url,omitempty"`
	NotInLibrary bool `json:"notInLibrary,omitempty"`
	Folder     bool   `json:"folder,omitempty"`
	FolderRel  string `json:"folderRel,omitempty"`
}

// CollectionDTO is a collection with its document count.
type CollectionDTO struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Icon          string `json:"icon,omitempty"`
	DocumentCount int64  `json:"documentCount"`
}

// CollectionDocumentDTO is a membership row with source breadcrumbs.
type CollectionDocumentDTO struct {
	DocumentID int64  `json:"documentId"`
	Title      string `json:"title"`
	RelPath    string `json:"relPath"`
	SourceName string `json:"sourceName"`
	SortOrder  int64  `json:"sortOrder"`
}

// SearchResultDTO is one search hit.
type SearchResultDTO struct {
	DocumentID int64   `json:"documentId"`
	Title      string  `json:"title"`
	RelPath    string  `json:"relPath"`
	SourceName string  `json:"sourceName"`
	Snippet    string  `json:"snippet"`
	Rank       float64 `json:"rank"`
}

// SearchResultsDTO is a page of results.
type SearchResultsDTO struct {
	Results []SearchResultDTO `json:"results"`
	ElapsedMS int64           `json:"elapsedMs"`
}

// ScopeDTO is the search scope.
type ScopeDTO struct {
	Type string `json:"type"` // library | source | collection
	ID   int64  `json:"id,omitempty"`
}

// ReadingStateDTO is a context's resume point.
type ReadingStateDTO struct {
	ContextType string  `json:"contextType"`
	ContextID   int64   `json:"contextId"`
	DocumentID  int64   `json:"documentId,omitempty"`
	ScrollPct   float64 `json:"scrollPct"`
}

// OpenTabDTO is one reading tab.
type OpenTabDTO struct {
	DocumentID int64  `json:"documentId"`
	Pane       int    `json:"pane"`
	IsActive   bool   `json:"isActive"`
	Title      string `json:"title,omitempty"`
	RelPath    string `json:"relPath,omitempty"`
}

// ExportTargetDTO selects what to export.
type ExportTargetDTO struct {
	Kind         string `json:"kind"` // document | collection
	DocumentID   int64  `json:"documentId,omitempty"`
	CollectionID int64  `json:"collectionId,omitempty"`
	Theme        string `json:"theme,omitempty"`
	IncludeTOC   bool   `json:"includeToc,omitempty"`
	ShowLinkURLs bool   `json:"showLinkUrls,omitempty"`
}

// ExportPayloadDTO is the self-contained export payload.
type ExportPayloadDTO struct {
	Title string `json:"title"`
	HTML  string `json:"html"`
	Theme string `json:"theme"`
}

// WindowStateDTO is the persisted window geometry.
type WindowStateDTO struct {
	Width    int  `json:"width"`
	Height   int  `json:"height"`
	X        int  `json:"x"`
	Y        int  `json:"y"`
	Maximized bool `json:"maximized"`
}

// toDocumentDTO maps a domain document + highlights to the DTO.
func toDocumentDTO(d domain.Document, missing bool) DocumentDTO {
	outline := make([]OutlineEntryDTO, 0, len(d.Outline))
	for _, h := range d.Outline {
		outline = append(outline, OutlineEntryDTO{
			Level: h.Level, Text: h.Text, Anchor: h.Anchor, BlockIndex: h.BlockIndex,
		})
	}
	return DocumentDTO{
		ID: d.ID, SourceID: d.SourceID, RelPath: d.RelPath, Title: d.Title,
		RenderedHTML: d.RenderedHTML, Outline: outline, WordCount: d.WordCount,
		HasMermaid: d.HasMermaid, MissingFile: missing,
	}
}

func toSourceDTO(s domain.Source) SourceDTO {
	return SourceDTO{
		ID: s.ID, Kind: string(s.Kind), Name: s.Name, RootPath: s.RootPath,
		OriginURL: s.OriginURL, GitBranch: s.GitBranch, GitCommit: s.GitCommit,
		IsManaged: s.IsManaged, Status: string(s.Status), ErrorMessage: s.ErrorMessage,
		DocumentCount: s.DocumentCount, IgnoreGlobs: s.IgnoreGlobs, IndexedAt: s.IndexedAt,
	}
}

func toHighlightDTO(h domain.Highlight) HighlightDTO {
	return HighlightDTO{
		ID: h.ID, DocumentID: h.DocumentID, BlockHash: h.BlockHash, BlockIndex: h.BlockIndex,
		StartOffset: h.StartOffset, EndOffset: h.EndOffset, QuotedText: h.QuotedText,
		Color: string(h.Color), Note: h.Note,
	}
}
