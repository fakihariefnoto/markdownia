package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// migrate applies every embedded migration in filename order, exactly once,
// tracking applied versions in a schema_migrations table. Forward-only and
// safe to run unattended: running twice is a no-op.
func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`); err != nil {
		return fmt.Errorf("migrate: create ledger: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("migrate: read embedded migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	applied := map[string]bool{}
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("migrate: read ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return fmt.Errorf("migrate: scan ledger: %w", err)
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("migrate: ledger rows: %w", err)
	}

	for _, e := range entries {
		if applied[e.Name()] {
			continue
		}
		raw, err := migrationFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", e.Name(), err)
		}
		// modernc.org/sqlite accepts multiple statements per Exec.
		if _, err := db.ExecContext(ctx, string(raw)); err != nil {
			return fmt.Errorf("migrate: apply %s: %w", e.Name(), err)
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO schema_migrations(version, applied_at) VALUES (?, datetime('now'))`,
			e.Name()); err != nil {
			return fmt.Errorf("migrate: record %s: %w", e.Name(), err)
		}
	}
	return nil
}
