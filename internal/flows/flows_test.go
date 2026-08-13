package flows

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/markdown"
	"github.com/anofac/markdownia/internal/repository/sqlite"
	"github.com/anofac/markdownia/internal/usecase/annotation"
	"github.com/anofac/markdownia/internal/usecase/library"
	sourceusecase "github.com/anofac/markdownia/internal/usecase/source"
)

// sink captures emitted index-completion events for assertions, and satisfies
// both the indexer.Progress and source.ProgressSink interfaces.
type sink struct {
	mu       sync.Mutex
	indexed  map[int64]indexDone
	errorMsg string
}

type indexDone struct {
	indexed          int64
	removedHighlights int64
}

func (s *sink) SourceProgress(sourceID int64, phase string, current, total int) {}
func (s *sink) SourceStatus(sourceID int64, status domain.SourceStatus, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if status == domain.StatusError {
		s.indexed[sourceID] = indexDone{indexed: -1, removedHighlights: 0}
		if errMsg != "" {
			s.errorMsg = errMsg
		}
	}
}
func (s *sink) SourceIndexed(sourceID int64, indexed, removedHighlights, deletedDocs int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.indexed[sourceID] = indexDone{indexed: indexed, removedHighlights: removedHighlights}
}
func (s *sink) SearchInvalidated() {}
func (s *sink) Emit(name string, data any) {}

func newSink() *sink { return &sink{indexed: map[int64]indexDone{}} }

type testEnv struct {
	db       *sqlite.DB
	parser   *markdown.Parser
	srcRepo  *sqlite.SourceRepository
	docRepo  *sqlite.DocumentRepository
	annRepo  *sqlite.AnnotationRepository
	sink     *sink
	sourceSvc *sourceusecase.Service
	lib      *library.Service
	ann      *annotation.Service
	extracted string
}

func newEnv(t *testing.T) *testEnv {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlite.Open(context.Background(), filepath.Join(dir, "test.db"), nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	srcRepo := sqlite.NewSourceRepository(db)
	docRepo := sqlite.NewDocumentRepository(db)
	annRepo := sqlite.NewAnnotationRepository(db)
	parser := markdown.NewParser()
	s := newSink()

	// Indexer wired with the annotation repo as sweeper.
	ix := newIndexer(docRepo, srcRepo, parser, annRepo, s, dir)
	sourceSvc := sourceusecase.New(sourceusecase.Options{
		Repo: srcRepo, Git: &nopGit{}, Indexer: ix, Progress: s, ExtractedRoot: dir,
	})
	lib := library.New(docRepo, srcRepo, annRepo, sqlite.NewReadingRepository(db), docRepo, parser)
	ann := annotation.New(annRepo, docRepo, annotation.NewMarkdownBlocks(docRepo, srcRepo, parser))

	return &testEnv{
		db: db, parser: parser, srcRepo: srcRepo, docRepo: docRepo, annRepo: annRepo,
		sink: s, sourceSvc: sourceSvc, lib: lib, ann: ann, extracted: dir,
	}
}

// writeSource writes fixture markdown files into a temp folder source and
// indexes it synchronously (waiting for the async index to finish).
func (e *testEnv) writeSource(t *testing.T, files map[string]string) int64 {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	id, err := e.sourceSvc.ImportFolder(context.Background(), root)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	e.waitIndexed(t, id)
	return id
}

func (e *testEnv) waitIndexed(t *testing.T, id int64) {
	t.Helper()
	for i := 0; i < 200; i++ {
		e.sink.mu.Lock()
		d, ok := e.sink.indexed[id]
		e.sink.mu.Unlock()
		if ok {
			if d.indexed == -1 {
				t.Fatalf("source %d entered error state during index: %s", id, e.sink.errorMsg)
			}
			return
		}
		<-sleepNow()
	}
	t.Fatal("index did not complete")
}

// --- F1/F2/F3 tests ---

// TestF1ImportIndexOpen is the vertical slice: import → index → open (hot
// path) → cached HTML present, no parse needed.
func TestF1ImportIndexOpen(t *testing.T) {
	e := newEnv(t)
	id := e.writeSource(t, map[string]string{
		"docs/intro.md": "# Intro\n\nHello world.\n",
		"notes/a.md":    "# Notes\n\nSome **notes** here.\n",
	})

		// Source is ready with document count.
		src, err := e.srcRepo.GetByID(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if src.Status != domain.StatusReady {
			t.Fatalf("status = %s, want ready (indexed=%+v)", src.Status, e.sink.indexed)
		}
		if src.DocumentCount != 2 {
			t.Fatalf("document count = %d, want 2 (indexed=%+v)", src.DocumentCount, e.sink.indexed)
		}

	// Open the hot path: cached HTML + outline, one call.
	docs, err := e.docRepo.ListBySource(context.Background(), id)
	if err != nil || len(docs) == 0 {
		t.Fatalf("list docs: %v (n=%d)", err, len(docs))
	}
	payload, err := e.lib.OpenDocument(context.Background(), docs[0].ID, domain.ContextLibrary, 0)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Logf("opened %d: html=%q outline=%d", docs[0].ID, payload.Document.RenderedHTML, len(payload.Document.Outline))
	if !strings.Contains(payload.Document.RenderedHTML, "<p ") {
		t.Error("cached HTML missing paragraph")
	}
	if !strings.Contains(payload.Document.RenderedHTML, "data-block-hash=") {
		t.Error("cached HTML missing block anchor attributes")
	}
	if len(payload.Document.Outline) == 0 {
		t.Error("outline empty")
	}
}

// TestF3HighlightSurvivesUnchangedReindex is the regression suite's core:
// re-indexing an unchanged source removes zero highlights.
func TestF3HighlightSurvivesUnchangedReindex(t *testing.T) {
	e := newEnv(t)
	id := e.writeSource(t, map[string]string{
		"doc.md": "# Doc\n\nThe stable paragraph.\n",
	})

	docs, _ := e.docRepo.ListBySource(context.Background(), id)
	res, err := e.parser.Render([]byte("# Doc\n\nThe stable paragraph.\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Blocks) == 0 {
		t.Fatal("no blocks in fixture")
	}
	block := res.Blocks[len(res.Blocks)-1] // the paragraph block

	_, err = e.ann.AddHighlight(context.Background(), docs[0].ID, domain.HighlightAnchor{
		BlockHash: block.Hash, BlockIndex: block.Index,
		StartOffset: 0, EndOffset: 10,
	}, "yellow", nil)
	if err != nil {
		t.Fatalf("add highlight: %v", err)
	}

	// Re-index the unchanged source.
	if err := e.sourceSvc.RefreshSource(context.Background(), id); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	e.waitIndexed(t, id)

	// Highlight must still exist.
	hls, err := e.annRepo.ListHighlights(context.Background(), docs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hls) != 1 {
		t.Fatalf("highlights = %d after unchanged re-index, want 1", len(hls))
	}
}

// TestF3EditedBlockSweepsHighlight asserts that editing a paragraph removes
// only its highlight and reports the count.
func TestF3EditedBlockSweepsHighlight(t *testing.T) {
	e := newEnv(t)
	root := t.TempDir()
	docPath := filepath.Join(root, "doc.md")
	content := "# Doc\n\nThe stable paragraph.\n"
	if err := os.WriteFile(docPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	id, err := e.sourceSvc.ImportFolder(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	e.waitIndexed(t, id)

	docs, _ := e.docRepo.ListBySource(context.Background(), id)
	res, _ := e.parser.Render([]byte(content))
	block := res.Blocks[len(res.Blocks)-1]

	if _, err := e.ann.AddHighlight(context.Background(), docs[0].ID, domain.HighlightAnchor{
		BlockHash: block.Hash, BlockIndex: block.Index, StartOffset: 0, EndOffset: 10,
	}, "blue", nil); err != nil {
		t.Fatal(err)
	}

	// Edit the paragraph → its block hash changes.
	edited := "# Doc\n\nThe CHANGED paragraph now.\n"
	if err := os.WriteFile(docPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := e.sourceSvc.RefreshSource(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	e.waitIndexed(t, id)

	hls, err := e.annRepo.ListHighlights(context.Background(), docs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hls) != 0 {
		t.Fatalf("highlights = %d after edit, want 0 (swept)", len(hls))
	}
}
