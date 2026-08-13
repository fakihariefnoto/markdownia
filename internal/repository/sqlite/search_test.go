package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/anofac/markdownia/internal/domain"
)

func searchTestDB(t *testing.T) (*DB, func() int64) {
	t.Helper()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "t.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	docRepo := NewDocumentRepository(db)
	ctx := context.Background()
	if _, err := db.Writer.ExecContext(ctx,
		`INSERT INTO sources(kind,name,root_path,status,created_at,updated_at) VALUES('folder','src','/x','ready','now','now')`); err != nil {
		t.Fatal(err)
	}
	docs := []*domain.Document{
		{SourceID: 1, RelPath: "api.md", Title: "API Reference", TitleSource: domain.TitleFilename,
			FileHash: "h1", FileMtime: 1, FileSize: 1, RenderedHTML: "<p>x</p>",
			PlainText: "The database connection string lives here.", RendererVersion: 1, WordCount: 7},
		{SourceID: 1, RelPath: "cook.md", Title: "Cooking", TitleSource: domain.TitleFilename,
			FileHash: "h2", FileMtime: 1, FileSize: 1, RenderedHTML: "<p>x</p>",
			PlainText: "Recipes for pasta and sauce.", RendererVersion: 1, WordCount: 5},
	}
	for _, d := range docs {
		if err := docRepo.Upsert(ctx, d); err != nil {
			t.Fatal(err)
		}
	}
	last := int64(len(docs))
	return db, func() int64 { return last }
}

// TestSearchInjectionAttempts asserts malicious MATCH input returns results or
// no results, never an error, and never injects into the query.
func TestSearchInjectionAttempts(t *testing.T) {
	db, _ := searchTestDB(t)
	repo := NewSearchRepository(db)
	attacks := []string{
		`'; DROP TABLE documents; --`,
		`database" OR 1=1 --`,
		`( ) OR ( )`,
		`*`,
		`"`,
		`a:b:c`,
		`--`,
	}
	for _, q := range attacks {
		results, err := repo.Search(context.Background(), Query{Text: q, Scope: domain.ContextLibrary, Limit: 10})
		if err != nil {
			t.Errorf("query %q returned error: %v", q, err)
			continue
		}
		// The table must still exist — nothing was injected.
		var n int
		if err := db.Reader.QueryRow(`SELECT count(*) FROM documents`).Scan(&n); err != nil {
			t.Errorf("documents table unusable after query %q: %v", q, err)
		}
		_ = results
	}
}

// TestSearchCodeToggle asserts flipping IncludeCode changes results without a
// re-index.
func TestSearchCodeToggle(t *testing.T) {
	db, _ := searchTestDB(t)
	docRepo := NewDocumentRepository(db)
	if err := docRepo.Upsert(context.Background(), &domain.Document{
		SourceID: 1, RelPath: "code.md", Title: "Snippet", TitleSource: domain.TitleFilename,
		FileHash: "h3", FileMtime: 1, FileSize: 1, RenderedHTML: "<p>x</p>",
		PlainText: "Some prose.", CodeText: "DATABASE_URL=postgres://localhost",
		RendererVersion: 1, WordCount: 2,
	}); err != nil {
		t.Fatal(err)
	}

	repo := NewSearchRepository(db)
	off, err := repo.Search(context.Background(), Query{Text: "DATABASE_URL", Scope: domain.ContextLibrary, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	on, err := repo.Search(context.Background(), Query{Text: "DATABASE_URL", Scope: domain.ContextLibrary, IncludeCode: true, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(off) != 0 {
		t.Errorf("code search returned results with toggle off: %d", len(off))
	}
	if len(on) == 0 {
		t.Error("code search returned nothing with toggle on")
	}
}

// TestSearchScopeFiltering asserts source scoping excludes other sources.
func TestSearchScopeFiltering(t *testing.T) {
	db, _ := searchTestDB(t)
	if _, err := db.Writer.ExecContext(context.Background(),
		`INSERT INTO sources(kind,name,root_path,status,created_at,updated_at) VALUES('folder','other','/y','ready','now','now')`); err != nil {
		t.Fatal(err)
	}
	if err := NewDocumentRepository(db).Upsert(context.Background(), &domain.Document{
		SourceID: 2, RelPath: "other.md", Title: "Other API", TitleSource: domain.TitleFilename,
		FileHash: "h4", FileMtime: 1, FileSize: 1, RenderedHTML: "<p>x</p>",
		PlainText: "The database is elsewhere.", RendererVersion: 1, WordCount: 4,
	}); err != nil {
		t.Fatal(err)
	}

	repo := NewSearchRepository(db)
	scoped, err := repo.Search(context.Background(), Query{Text: "database", Scope: domain.ContextSource, ScopeID: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range scoped {
		if r.Title == "Other API" {
			t.Errorf("scope leak: result %q from source 2 returned for source 1 scope", r.Title)
		}
	}
}

// TestSearchRanking asserts a title match outranks a body mention.
func TestSearchRanking(t *testing.T) {
	db, _ := searchTestDB(t)
	if err := NewDocumentRepository(db).Upsert(context.Background(), &domain.Document{
		SourceID: 1, RelPath: "database.md", Title: "Database", TitleSource: domain.TitleFilename,
		FileHash: "h5", FileMtime: 1, FileSize: 1, RenderedHTML: "<p>x</p>",
		PlainText: "Nothing relevant here.", RendererVersion: 1, WordCount: 3,
	}); err != nil {
		t.Fatal(err)
	}

	repo := NewSearchRepository(db)
	results, err := repo.Search(context.Background(), Query{Text: "database", Scope: domain.ContextLibrary, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("no results for 'database'")
	}
	if results[0].Title != "Database" {
		t.Errorf("top result %q, want 'Database' (title match should outrank body)", results[0].Title)
	}
}
