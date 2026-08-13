package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

// SettingsRepository persists the flat JSON key/value store.
type SettingsRepository struct{ db *DB }

// NewSettingsRepository constructs the SQLite settings repository.
func NewSettingsRepository(db *DB) *SettingsRepository { return &SettingsRepository{db: db} }

func (r *SettingsRepository) GetAll(ctx context.Context) (map[string]json.RawMessage, error) {
	rows, err := r.db.Reader.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = json.RawMessage(v)
	}
	return out, rows.Err()
}

func (r *SettingsRepository) Get(ctx context.Context, key string) (json.RawMessage, bool, error) {
	var v string
	err := r.db.Reader.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return json.RawMessage(v), true, nil
}

func (r *SettingsRepository) Set(ctx context.Context, key string, value json.RawMessage) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Writer.ExecContext(ctx, `
		INSERT INTO settings(key, value, updated_at) VALUES (?,?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
		key, string(value), now)
	return err
}

func (r *SettingsRepository) Reset(ctx context.Context, key string) error {
	_, err := r.db.Writer.ExecContext(ctx, `DELETE FROM settings WHERE key=?`, key)
	return err
}
