package domain

// Collection is a curated, document-level reading list (the logical axis).
type Collection struct {
	ID          int64
	Name        string
	Description string
	Icon        string
	SortOrder   int64
	CreatedAt   string
	UpdatedAt   string
}

// CollectionDocument is a membership row: a document at a manual position
// within one collection.
type CollectionDocument struct {
	CollectionID int64
	DocumentID   int64
	SortOrder    int64
	AddedAt      string
}
