package domain

import "encoding/json"

// TitleSource records which rule resolved a document's title.
type TitleSource string

const (
	TitleFrontmatter TitleSource = "frontmatter"
	TitleH1          TitleSource = "h1"
	TitleFilename    TitleSource = "filename"
)

// OutlineHeading is one node in the document's heading tree.
type OutlineHeading struct {
	Level      int    `json:"level"`
	Text       string `json:"text"`
	Anchor     string `json:"anchor"`
	BlockIndex int    `json:"blockIndex"`
}

// Document is one markdown file, carrying the caches that make the hot path
// fast: rendered HTML, plain text, code text, outline, and frontmatter.
type Document struct {
	ID             int64
	SourceID       int64
	RelPath        string
	Title          string
	TitleSource    TitleSource
	FileHash       string
	FileMtime      int64
	FileSize       int64
	RenderedHTML   string
	PlainText      string
	CodeText       string
	Outline        []OutlineHeading
	Frontmatter    json.RawMessage
	RendererVersion int
	WordCount      int64
	HasMermaid     bool
	IndexedAt      string
}

// IsMarkdownExtension reports whether the given filename is treated as
// markdown (`.md`, `.markdown`, `.mdx`).
func IsMarkdownExtension(name string) bool {
	switch name {
	case ".md", ".markdown", ".mdx":
		return true
	}
	return false
}
