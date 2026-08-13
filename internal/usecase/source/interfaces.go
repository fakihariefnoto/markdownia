package source

import (
	"context"

	"github.com/anofac/markdownia/internal/domain"
)

// Repository is the source usecase's narrow view of persistence. Declared here
// (consuming package) per Go convention.
type Repository interface {
	List(ctx context.Context) ([]domain.Source, error)
	GetByID(ctx context.Context, id int64) (domain.Source, error)
	Create(ctx context.Context, s *domain.Source) (int64, error)
	Update(ctx context.Context, s *domain.Source) error
	SetStatus(ctx context.Context, id int64, status domain.SourceStatus, errMsg string) error
	MarkIndexed(ctx context.Context, id int64, count int64, commit string) error
	Delete(ctx context.Context, id int64) error
	Counts(ctx context.Context, id int64) (documents, highlights, bookmarks, collectionEntries int64, err error)
}

// DocumentRepository is used to count documents after indexing and to clean up
// stale tabs.
type DocumentRepository interface {
	CountBySource(ctx context.Context, sourceID int64) (int64, error)
}

// GitClient clones, pulls, and reads git metadata.
type GitClient interface {
	Clone(ctx context.Context, url, branch, dest string, progress func(string)) error
	Pull(ctx context.Context, dest string, progress func(string)) error
	HeadInfo(dest string) (branch, commit string, err error)
	IsRepository(path string) bool
}

// Indexer indexes (or re-indexes) a source. The source usecase invokes it
// asynchronously; progress and completion arrive as events.
type Indexer interface {
	Index(ctx context.Context, sourceID int64) error
}

// ProgressSink reports source operation progress and status to the UI.
type ProgressSink interface {
	SourceProgress(sourceID int64, phase string, current, total int)
	SourceStatus(sourceID int64, status domain.SourceStatus, errMsg string)
	SourceIndexed(sourceID int64, indexed, removedHighlights, deletedDocs int64)
	SearchInvalidated()
}

// DeletionPreview carries the numbers the confirm dialog must state before a
// user commits to deleting a source.
type DeletionPreview struct {
	Documents        int64
	Highlights       int64
	Bookmarks        int64
	CollectionEntries int64
	DeletesFilesOnDisk bool
}
