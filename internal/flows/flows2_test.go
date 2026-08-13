package flows

import (
	"context"
	"testing"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/markdown"
	"github.com/anofac/markdownia/internal/repository/sqlite"
)

// TestF4CreateHighlight validates the full anchor round-trip: render → compute
// anchor from block hashes → Go validation → stored → visible.
func TestF4CreateHighlight(t *testing.T) {
	e := newEnv(t)
	id := e.writeSource(t, map[string]string{
		"doc.md": "# Doc\n\nThe first paragraph to highlight.\n\nSecond para.\n",
	})

	docs, _ := e.docRepo.ListBySource(context.Background(), id)

	// Derive the paragraph block from the same parse the indexer used.
	res, err := e.parser.Render([]byte("# Doc\n\nThe first paragraph to highlight.\n\nSecond para.\n"))
	if err != nil {
		t.Fatal(err)
	}
	var para *markdown.Block
	for i := range res.Blocks {
		if res.Blocks[i].TextLength > 20 {
			para = &res.Blocks[i]
			break
		}
	}
	if para == nil {
		t.Fatal("no paragraph block found")
	}

	// Valid anchor accepted.
	idHl, err := e.ann.AddHighlight(context.Background(), docs[0].ID, domain.HighlightAnchor{
		BlockHash: para.Hash, BlockIndex: para.Index, StartOffset: 0, EndOffset: 10,
	}, "yellow", nil)
	if err != nil {
		t.Fatalf("add valid highlight: %v", err)
	}
	if idHl == 0 {
		t.Fatal("highlight id zero")
	}

	// Invalid anchor rejected at creation.
	_, err = e.ann.AddHighlight(context.Background(), docs[0].ID, domain.HighlightAnchor{
		BlockHash: "deadbeef", BlockIndex: 0, StartOffset: 0, EndOffset: 5,
	}, "blue", nil)
	if err == nil {
		t.Error("invalid block hash accepted")
	}

	// Offsets past block length rejected.
	_, err = e.ann.AddHighlight(context.Background(), docs[0].ID, domain.HighlightAnchor{
		BlockHash: para.Hash, BlockIndex: para.Index, StartOffset: 0, EndOffset: 99999,
	}, "blue", nil)
	if err == nil {
		t.Error("out-of-range offsets accepted")
	}

	// Invalid color rejected.
	_, err = e.ann.AddHighlight(context.Background(), docs[0].ID, domain.HighlightAnchor{
		BlockHash: para.Hash, BlockIndex: para.Index, StartOffset: 0, EndOffset: 5,
	}, "chartreuse", nil)
	if err == nil {
		t.Error("invalid color accepted")
	}

	// Cross-block selection is a frontend concern; here we assert the stored
	// highlight is listed.
	hls, err := e.annRepo.ListHighlights(context.Background(), docs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hls) != 1 {
		t.Fatalf("highlights = %d, want 1", len(hls))
	}
}

// TestF5DeleteSourceCascades asserts source deletion removes documents and
// their annotations, closing the loop on the delete preview.
func TestF5DeleteSourceCascades(t *testing.T) {
	e := newEnv(t)
	id := e.writeSource(t, map[string]string{"doc.md": "# Doc\n\nBody.\n"})

	docs, _ := e.docRepo.ListBySource(context.Background(), id)
	res, _ := e.parser.Render([]byte("# Doc\n\nBody.\n"))
	para := res.Blocks[len(res.Blocks)-1]
	if _, err := e.ann.AddHighlight(context.Background(), docs[0].ID, domain.HighlightAnchor{
		BlockHash: para.Hash, BlockIndex: para.Index, StartOffset: 0, EndOffset: 4,
	}, "pink", nil); err != nil {
		t.Fatal(err)
	}

	if err := e.sourceSvc.DeleteSource(context.Background(), id); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Document gone, highlight gone (cascade).
	docsAfter, _ := e.docRepo.ListBySource(context.Background(), id)
	if len(docsAfter) != 0 {
		t.Fatalf("documents after delete = %d, want 0", len(docsAfter))
	}
	all, err := e.annRepo.ListAllHighlights(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 0 {
		t.Fatalf("highlights after delete = %d, want 0", len(all))
	}
}

// TestF6SearchAndOpen runs a scoped FTS search against a real index.
func TestF6SearchAndOpen(t *testing.T) {
	e := newEnv(t)
	id := e.writeSource(t, map[string]string{
		"api.md": "# API\n\nThe database connection lives here.\n",
		"cook.md": "# Cooking\n\nRecipes for pasta.\n",
	})

	docs, _ := e.docRepo.ListBySource(context.Background(), id)
	_ = docs

	// The SQLite search repo is reachable through the full wiring; drive it
	// through the usecase adapter via the app Build path is heavy, so query
	// the repo directly — FTS5 correctness is what matters here.
	searchRepo := newSearchRepo(e.db)
	results, err := searchRepo.Search(context.Background(), sqlite.Query{
		Text: "database", Scope: domain.ContextSource, ScopeID: id,
		IncludeCode: false, Limit: 10, Offset: 0,
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("search found nothing for 'database'")
	}
	if results[0].Title != "API" {
		t.Errorf("top result = %q, want API", results[0].Title)
	}
}
