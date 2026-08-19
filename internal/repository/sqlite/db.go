// Package sqlite owns all SQL access. It is the only package that imports the
// SQLite driver. All schema lives in embedded forward-only migrations.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the dual-handle SQLite setup: a single-connection writer and a read
// pool. SQLite permits one writer at a time; serializing in Go beats surfacing
// SQLITE_BUSY to the user, while the read pool keeps document opens instant
// during a large index.
type DB struct {
	Writer *sql.DB
	Reader *sql.DB
}

// Open opens the database at path, applies PRAGMAs, and runs forward-only
// migrations. Both handles share the same file.
//
// PRAGMAs are passed via DSN _pragma params so every pooled connection (not
// just the ones created at Open) carries busy_timeout and foreign_keys. A
// fresh connection without busy_timeout can fail writes with an immediate
// SQLITE_BUSY, which previously let the indexer silently drop documents under
// concurrent worker-pool writes.
func Open(ctx context.Context, path string, logger *slog.Logger) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("sqlite: mkdir %s: %w", filepath.Dir(path), err)
	}

	dsn := "file:" + path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(5000)"
	writer, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open writer: %w", err)
	}
	reader, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open reader: %w", err)
	}

	// SQLite permits one writer at a time; pinning the writer handle to a
	// single connection serializes transactions in Go instead of surfacing
	// SQLITE_BUSY to callers.
	writer.SetMaxOpenConns(1)

	db := &DB{Writer: writer, Reader: reader}
	if err := migrate(ctx, writer); err != nil {
		return nil, err
	}
	if logger != nil {
		logger.Info("database open", "path", path)
	}
	return db, nil
}

// Close closes both handles.
func (d *DB) Close() error {
	if d.Writer != nil {
		_ = d.Writer.Close()
	}
	if d.Reader != nil {
		_ = d.Reader.Close()
	}
	return nil
}

// Tx runs fn inside a writer transaction, rolling back on panic or error.
func (d *DB) Tx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := d.Writer.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
