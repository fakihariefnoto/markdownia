package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func openTest(t *testing.T) *DB {
	t.Helper()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"), nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestMigrationsFullSchema asserts that applying migrations to an empty file
// produces the complete schema, and that running twice is a no-op.
func TestMigrationsFullSchema(t *testing.T) {
	db := openTest(t)

	var count int
	if err := db.Reader.QueryRow(`
		SELECT count(*) FROM sqlite_master WHERE type='table' AND name IN (
			'sources','documents','documents_fts','collections','collection_documents',
			'highlights','bookmarks','reading_state','open_tabs','reading_history','settings'
		)`).Scan(&count); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if count != 11 {
		t.Fatalf("expected 11 tables, got %d", count)
	}

	// Running migrations again is a no-op.
	if err := migrate(context.Background(), db.Writer); err != nil {
		t.Fatalf("re-migrate: %v", err)
	}
}

// TestPragmasSet asserts each PRAGMA reads back as configured. foreign_keys=ON
// is the critical one: SQLite defaults it off and every ON DELETE CASCADE in
// the ERD is inert without it.
func TestPragmasSet(t *testing.T) {
	db := openTest(t)

	checks := []struct{ pragma, want string }{
		{"journal_mode", "wal"},
		{"synchronous", "1"}, // NORMAL
		{"foreign_keys", "1"},
	}
	for _, c := range checks {
		var got string
		if err := db.Reader.QueryRow("PRAGMA " + c.pragma).Scan(&got); err != nil {
			t.Fatalf("read %s: %v", c.pragma, err)
		}
		if got != c.want {
			t.Errorf("PRAGMA %s = %q, want %q", c.pragma, got, c.want)
		}
	}
}

// TestSingleActiveTabPerPane asserts the partial unique index rejects a second
// active tab in the same (context, pane) at the database, not in app code.
func seedDoc(t *testing.T, db *DB, relPath string) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.Writer.Exec(`
		INSERT INTO sources(kind, name, root_path, status, created_at, updated_at)
		VALUES ('folder', 's', '/tmp/s', 'ready', 'now', 'now')`); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	if _, err := db.Writer.ExecContext(ctx, `
		INSERT INTO documents(source_id, rel_path, title, title_source, file_hash, file_mtime, file_size, rendered_html, indexed_at, renderer_version)
		VALUES (1, ?, 'A', 'filename', 'h1', 1, 1, '<p>a</p>', 'now', 0)`, relPath); err != nil {
		t.Fatalf("seed doc: %v", err)
	}
}

func TestSingleActiveTabPerPane(t *testing.T) {
	db := openTest(t)
	seedDoc(t, db, "a.md")
	if _, err := db.Writer.Exec(`
		INSERT INTO open_tabs(context_type, context_id, document_id, pane, position, is_active)
		VALUES ('library', 0, 1, 0, 0, 1)`); err != nil {
		t.Fatalf("insert active: %v", err)
	}
	if _, err := db.Writer.Exec(`
		INSERT INTO open_tabs(context_type, context_id, document_id, pane, position, is_active)
		VALUES ('library', 0, 1, 0, 1, 1)`); err == nil {
		t.Fatal("expected second active tab in same pane to fail at the database")
	}
}

// TestActiveTabAllowedPerPane asserts a split pane may have its own active tab.
func TestActiveTabAllowedPerPane(t *testing.T) {
	db := openTest(t)
	seedDoc(t, db, "a.md")
	for _, pane := range []int{0, 1} {
		if _, err := db.Writer.Exec(`
			INSERT INTO open_tabs(context_type, context_id, document_id, pane, position, is_active)
			VALUES ('library', 0, 1, ?, ?, 1)`, pane, pane); err != nil {
			t.Fatalf("insert active pane %d: %v", pane, err)
		}
	}
}
