package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

// TestFTS5Available is the go/no-go check for the search design: the pure-Go
// driver must ship FTS5 for the whole FTS5 external-content plan to work.
func TestFTS5Available(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(context.Background(), filepath.Join(dir, "test.db"), nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Writer.Exec(`
		CREATE VIRTUAL TABLE t_fts USING fts5(body);
		INSERT INTO t_fts(body) VALUES ('database url connection string');`); err != nil {
		t.Fatalf("FTS5 unavailable: %v", err)
	}
	var n int
	if err := db.Writer.QueryRow(`SELECT count(*) FROM t_fts WHERE t_fts MATCH 'database'`).Scan(&n); err != nil {
		t.Fatalf("fts5 query: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 match, got %d", n)
	}
}
