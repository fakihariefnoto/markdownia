package domain

// ContextType names a reading context: the whole library, one source, or one
// collection.
type ContextType string

const (
	ContextLibrary    ContextType = "library"
	ContextSource     ContextType = "source"
	ContextCollection ContextType = "collection"
)

// ReadingState is the resume point for one reading context.
type ReadingState struct {
	ContextType ContextType
	ContextID   int64
	DocumentID  int64
	ScrollPct   float64
	UpdatedAt   string
}

// OpenTab is one open reading tab in a pane of a context's tab set.
type OpenTab struct {
	ID          int64
	ContextType ContextType
	ContextID   int64
	DocumentID  int64
	Pane        int // 0 = primary, 1 = split
	Position    int
	IsActive    bool
	Title       string // denormalized from documents for tab labels
	RelPath     string
}

// ReadingHistory is one row per document: the app needs "recently read", not
// an append-only analytics log.
type ReadingHistory struct {
	DocumentID        int64
	LastOpenedAt      string
	OpenCount         int64
	FurthestScrollPct float64
}
