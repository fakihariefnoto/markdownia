// Package indexer orchestrates source indexing with the changeset ordering
// that protects highlights (domains.md §5.1).
package indexer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/markdown"
	"github.com/anofac/markdownia/internal/pathguard"
	"github.com/anofac/markdownia/internal/repository/sqlite"
)

// FileState mirrors the repository's cheap re-index skip check.
type FileState = sqlite.FileState

// Result reports what an index pass did, for the post-index summary.
type Result struct {
	Indexed          int64
	RemovedHighlights int64
	DeletedDocs      int64
	Unchanged        int64
}

// SourceRepo is the indexer's view of source persistence.
type SourceRepo interface {
	GetByID(ctx context.Context, id int64) (domain.Source, error)
	SetStatus(ctx context.Context, id int64, status domain.SourceStatus, errMsg string) error
	MarkIndexed(ctx context.Context, id int64, count int64, commit string) error
}

// DocRepo is the indexer's view of document persistence.
type DocRepo interface {
	GetFileState(ctx context.Context, sourceID int64, relPath string) (FileState, bool, error)
	Upsert(ctx context.Context, d *domain.Document) error
	ReRenderHTML(ctx context.Context, id int64, html string, version int) error
	TouchMtime(ctx context.Context, id int64, mtime int64) error
	DeleteMissing(ctx context.Context, sourceID int64, keep map[string]bool) (int64, error)
	CountBySource(ctx context.Context, sourceID int64) (int64, error)
}

// Sweeper deletes highlights whose containing block changed. Implemented by
// the annotation repository.
type Sweeper interface {
	SweepDocumentHighlights(ctx context.Context, docID int64, keep map[string]bool) (int64, error)
}

// Progress reports index progress and results.
type Progress interface {
	SourceProgress(sourceID int64, phase string, current, total int)
	SourceIndexed(sourceID int64, indexed, removedHighlights, deletedDocs int64)
	SearchInvalidated()
}

// Indexer is the heavy component.
type Indexer struct {
	srcRepo  SourceRepo
	docRepo  DocRepo
	parser   *markdown.Parser
	sweeper  Sweeper
	progress Progress
	workers  int
}

// New constructs the indexer.
func New(srcRepo SourceRepo, docRepo DocRepo, parser *markdown.Parser, sweeper Sweeper, progress Progress, workers int) *Indexer {
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
		if workers > 8 {
			workers = 8
		}
	}
	return &Indexer{srcRepo: srcRepo, docRepo: docRepo, parser: parser, sweeper: sweeper, progress: progress, workers: workers}
}

// Index runs a full index pass for a source. It is the single entry point used
// by both import and refresh.
func (ix *Indexer) Index(ctx context.Context, sourceID int64) error {
	src, err := ix.srcRepo.GetByID(ctx, sourceID)
	if err != nil {
		return err
	}

	// Phase 1: count (indeterminate) then walk (determinate total).
	_ = ix.srcRepo.SetStatus(ctx, sourceID, domain.StatusIndexing, "")

	files, err := walk(src.RootPath, src.IgnoreGlobs)
	if err != nil {
		return err
	}
	total := len(files)
	ix.progress.SourceProgress(sourceID, "indexing", 0, total)

	// Phase 2: bounded worker pool over the changeset.
	type job struct {
		rel string
	}
	jobs := make(chan job, total)
	results := make(chan docResult, total)
	var wg sync.WaitGroup

	for i := 0; i < ix.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				res := ix.process(ctx, src, j.rel)
				results <- res
			}
		}()
	}

	go func() {
		for _, f := range files {
			select {
			case jobs <- job{rel: f}:
			case <-ctx.Done():
				close(jobs)
				return
			}
		}
		close(jobs)
	}()

	var indexed, unchanged, removed, deleted int64
	keep := map[string]bool{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for r := range results {
			keep[r.rel] = true
			if r.indexed {
				indexed++
			} else {
				unchanged++
			}
			removed += r.removed
			ix.progress.SourceProgress(sourceID, "indexing", int(indexed+unchanged), total)
		}
	}()

	wg.Wait()
	close(results)
	<-done

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrOperationCancelled, err)
	}

	// Phase 3: delete documents whose files no longer exist.
	deleted, err = ix.docRepo.DeleteMissing(ctx, sourceID, keep)
	if err != nil {
		return err
	}

	count, err := ix.docRepo.CountBySource(ctx, sourceID)
	if err != nil {
		return err
	}

	commit := src.GitCommit
	if err := ix.srcRepo.MarkIndexed(ctx, sourceID, count, commit); err != nil {
		return err
	}

	ix.progress.SourceIndexed(sourceID, indexed, removed, deleted)
	ix.progress.SearchInvalidated()
	return nil
}

// docResult is one file's processing outcome.
type docResult struct {
	rel     string
	indexed bool
	removed int64
}

// process handles one file with the changeset ordering that protects
// highlights: row exists? → mtime+size unchanged? → renderer version current?
// → skip. Otherwise re-render or re-parse.
func (ix *Indexer) process(ctx context.Context, src domain.Source, rel string) docResult {
	abs := pathguard.Join(src.RootPath, rel)
	if abs == "" {
		return docResult{rel: rel}
	}

	info, err := os.Stat(abs)
	if err != nil {
		// Missing file; DeleteMissing handles removal.
		return docResult{rel: rel}
	}
	if info.IsDir() {
		return docResult{rel: rel}
	}
	mtime := info.ModTime().Unix()
	size := info.Size()

	state, known, err := ix.docRepo.GetFileState(ctx, src.ID, rel)
	if err != nil {
		return docResult{rel: rel}
	}

	// Ordering: mtime check happens before any parse.
	if known && state.FileMtime == mtime && state.FileSize == size {
		if state.RendererVersion == markdown.RendererVersion {
			// SKIP ENTIRELY: no parse, no write, blocks untouched, highlights safe.
			return docResult{rel: rel}
		}
		// Re-render only from identical source. Block hashes recomputed from
		// identical text match — highlights safe.
// #nosec G304 -- abs was produced by pathguard.Join against the source root.
		content, err := os.ReadFile(abs)
		if err != nil {
			return docResult{rel: rel}
		}
		res, err := ix.parser.Render(content)
		if err != nil {
			return docResult{rel: rel}
		}
		if err := ix.docRepo.ReRenderHTML(ctx, state.ID, res.HTML, markdown.RendererVersion); err != nil {
			return docResult{rel: rel}
		}
		return docResult{rel: rel}
	}

	// mtime differs. Hash check: git rewrites mtimes wholesale, so a matching
	// content hash means only the mtime should be updated.
	if known {
		hash := hashFile(abs)
		if hash == state.FileHash {
			_ = ix.docRepo.TouchMtime(ctx, state.ID, mtime)
			return docResult{rel: rel}
		}
	}

	// Full re-parse and upsert.
// #nosec G304 -- abs was produced by pathguard.Join against the source root.
	content, err := os.ReadFile(abs)
	if err != nil {
		return docResult{rel: rel}
	}
	res, err := ix.parser.Render(content)
	if err != nil {
		return docResult{rel: rel}
	}

	title := res.Title
	if res.TitleSource == domain.TitleFilename {
		title = fileBaseTitle(rel)
	}

	doc := &domain.Document{
		SourceID:        src.ID,
		RelPath:         rel,
		Title:           title,
		TitleSource:     res.TitleSource,
		FileHash:        hashFile(abs),
		FileMtime:       mtime,
		FileSize:        size,
		RenderedHTML:    res.HTML,
		PlainText:       res.PlainText,
		CodeText:        res.CodeText,
		Outline:         res.Outline,
		RendererVersion: markdown.RendererVersion,
		WordCount:       int64(res.WordCount),
		HasMermaid:      res.HasMermaid,
	}
	if err := ix.docRepo.Upsert(ctx, doc); err != nil {
		return docResult{rel: rel}
	}

	// Sweep highlights for this document inside the index transaction: remove
	// any whose block hash is absent from the current blocks.
	var removedCount int64
	if len(res.Blocks) > 0 {
		keep := map[string]bool{}
		for _, b := range res.Blocks {
			keep[b.Hash] = true
		}
		if removedCount, err = ix.sweepForDoc(ctx, doc.ID, keep); err != nil {
			removedCount = 0
		}
	}

	return docResult{rel: rel, indexed: true, removed: removedCount}
}

// sweepForDoc delegates to the sweeper when available.
func (ix *Indexer) sweepForDoc(ctx context.Context, docID int64, keep map[string]bool) (int64, error) {
	if ix.sweeper == nil {
		return 0, nil
	}
	return ix.sweeper.SweepDocumentHighlights(ctx, docID, keep)
}

// hashFile returns a stable sha256 hex of a file's contents.
func hashFile(path string) string {
// #nosec G304 -- hashFile is only called with a pathguard-verified path.
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// fileBaseTitle is the filename-derived title (no extension, trimmed).
func fileBaseTitle(rel string) string {
	base := filepath.Base(rel)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		return base
	}
	return name
}
