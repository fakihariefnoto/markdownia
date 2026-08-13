package domain

// HighlightColor names the five semantic highlight colors. Hex values live in
// the design system and vary by reading-theme family; only the name persists.
type HighlightColor string

const (
	ColorYellow HighlightColor = "yellow"
	ColorGreen  HighlightColor = "green"
	ColorBlue   HighlightColor = "blue"
	ColorPink   HighlightColor = "pink"
	ColorOrange HighlightColor = "orange"
)

// AllHighlightColors lists the valid highlight colors.
var AllHighlightColors = []HighlightColor{
	ColorYellow, ColorGreen, ColorBlue, ColorPink, ColorOrange,
}

// IsValidHighlightColor reports whether c is a known semantic color name.
func IsValidHighlightColor(c string) bool {
	for _, valid := range AllHighlightColors {
		if string(valid) == c {
			return true
		}
	}
	return false
}

// HighlightAnchor is the frontend-computed anchor for a highlight: the hash of
// the containing block, its index within the document, and character offsets
// within that block's plain text.
type HighlightAnchor struct {
	BlockHash   string `json:"blockHash"`
	BlockIndex  int    `json:"blockIndex"`
	StartOffset int    `json:"startOffset"`
	EndOffset   int    `json:"endOffset"`
}

// Highlight is a text highlight anchored to a containing block.
type Highlight struct {
	ID          int64
	DocumentID  int64
	BlockHash   string
	BlockIndex  int
	StartOffset int
	EndOffset   int
	QuotedText  string
	Color       HighlightColor
	Note        string
	CreatedAt   string
	UpdatedAt   string
}

// Bookmark anchors to a document, not to text within it, and therefore
// survives re-index unconditionally.
type Bookmark struct {
	ID            int64
	DocumentID    int64
	HeadingAnchor string
	Note          string
	CreatedAt     string
}
