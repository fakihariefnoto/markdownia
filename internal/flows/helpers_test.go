package flows

import (
	"context"
	"time"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/indexer"
	"github.com/anofac/markdownia/internal/markdown"
	"github.com/anofac/markdownia/internal/repository/sqlite"
)

// nopGit satisfies the source usecase's GitClient for folder-source tests.
type nopGit struct{}

func (n *nopGit) Clone(ctx context.Context, url, branch, dest string, progress func(string)) error {
	return nil
}
func (n *nopGit) Pull(ctx context.Context, dest string, progress func(string)) error { return nil }
func (n *nopGit) HeadInfo(dest string) (string, string, error)                       { return "", "", nil }
func (n *nopGit) IsRepository(path string) bool                                      { return false }

// newIndexer wires the real indexer for flow tests.
func newIndexer(docRepo *sqlite.DocumentRepository, srcRepo *sqlite.SourceRepository, parser *markdown.Parser, sweeper *sqlite.AnnotationRepository, progress indexer.Progress, extracted string) *indexer.Indexer {
	return indexer.New(srcRepo, docRepo, parser, sweeper, progress, 2)
}

func sleepNow() <-chan time.Time {
	return time.After(20 * time.Millisecond)
}

// newSearchRepo exposes the SQLite search repository for flow tests.
func newSearchRepo(db *sqlite.DB) *sqlite.SearchRepository {
	return sqlite.NewSearchRepository(db)
}

var _ domain.SourceKind
