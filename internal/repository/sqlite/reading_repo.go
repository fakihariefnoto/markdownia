package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/anofac/markdownia/internal/domain"
)

// ReadingRepository persists reading state, tabs, and history.
type ReadingRepository struct{ db *DB }

// NewReadingRepository constructs the SQLite reading repository.
func NewReadingRepository(db *DB) *ReadingRepository { return &ReadingRepository{db: db} }

func (r *ReadingRepository) GetReadingState(ctx context.Context, ctxType domain.ContextType, ctxID int64) (domain.ReadingState, error) {
	var st domain.ReadingState
	var docID sql.NullInt64
	err := r.db.Reader.QueryRowContext(ctx, `
		SELECT context_type, context_id, document_id, scroll_pct, updated_at
		FROM reading_state WHERE context_type=? AND context_id=?`, ctxType, ctxID).
		Scan(&st.ContextType, &st.ContextID, &docID, &st.ScrollPct, &st.UpdatedAt)
	if err != nil {
		return st, err
	}
	st.DocumentID = docID.Int64
	return st, nil
}

func (r *ReadingRepository) SaveScrollPosition(ctx context.Context, ctxType domain.ContextType, ctxID int64, docID int64, pct float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Writer.ExecContext(ctx, `
		INSERT INTO reading_state(context_type, context_id, document_id, scroll_pct, updated_at)
		VALUES (?,?,?,?,?)
		ON CONFLICT(context_type, context_id) DO UPDATE SET
			document_id=excluded.document_id, scroll_pct=excluded.scroll_pct, updated_at=excluded.updated_at`,
		ctxType, ctxID, docID, pct, now)
	return err
}

// SetDocument updates just the current document of a context, preserving scroll.
func (r *ReadingRepository) SetDocument(ctx context.Context, ctxType domain.ContextType, ctxID int64, docID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Writer.ExecContext(ctx, `
		INSERT INTO reading_state(context_type, context_id, document_id, scroll_pct, updated_at)
		VALUES (?,?,?,0,?)
		ON CONFLICT(context_type, context_id) DO UPDATE SET
			document_id=excluded.document_id, updated_at=excluded.updated_at`,
		ctxType, ctxID, docID, now)
	return err
}

func (r *ReadingRepository) GetOpenTabs(ctx context.Context, ctxType domain.ContextType, ctxID int64) ([]domain.OpenTab, error) {
	rows, err := r.db.Reader.QueryContext(ctx, `
		SELECT ot.id, ot.context_type, ot.context_id, ot.document_id, ot.pane, ot.position, ot.is_active,
			COALESCE(d.title, ''), COALESCE(d.rel_path, '')
		FROM open_tabs ot
		LEFT JOIN documents d ON d.id = ot.document_id
		WHERE ot.context_type=? AND ot.context_id=?
		ORDER BY ot.pane, ot.position`, ctxType, ctxID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.OpenTab
	for rows.Next() {
		var t domain.OpenTab
		var active int
		if err := rows.Scan(&t.ID, &t.ContextType, &t.ContextID, &t.DocumentID,
			&t.Pane, &t.Position, &active, &t.Title, &t.RelPath); err != nil {
			return nil, err
		}
		t.IsActive = active == 1
		out = append(out, t)
	}
	return out, rows.Err()
}

// SaveOpenTabs replaces the tab set for a context in one transaction.
func (r *ReadingRepository) SaveOpenTabs(ctx context.Context, ctxType domain.ContextType, ctxID int64, tabs []domain.OpenTab) error {
	return r.db.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM open_tabs WHERE context_type=? AND context_id=?`,
			ctxType, ctxID); err != nil {
			return err
		}
		for i, t := range tabs {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO open_tabs(context_type, context_id, document_id, pane, position, is_active)
				VALUES (?,?,?,?,?,?)`,
				ctxType, ctxID, t.DocumentID, t.Pane, i, boolInt(t.IsActive)); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetLastContext returns the most recently updated reading state row, or
// ok=false when no reading state exists.
func (r *ReadingRepository) GetLastContext(ctx context.Context) (domain.ReadingState, bool, error) {
	var st domain.ReadingState
	var docID sql.NullInt64
	err := r.db.Reader.QueryRowContext(ctx, `
		SELECT context_type, context_id, document_id, scroll_pct, updated_at
		FROM reading_state ORDER BY updated_at DESC LIMIT 1`).
		Scan(&st.ContextType, &st.ContextID, &docID, &st.ScrollPct, &st.UpdatedAt)
	if err != nil {
		return st, false, err
	}
	st.DocumentID = docID.Int64
	return st, true, nil
}

// CleanupDeletedTabs removes open_tabs rows pointing at documents that no
// longer exist (after a source deletion or re-index sweep).
func (r *ReadingRepository) CleanupDeletedTabs(ctx context.Context) (int64, error) {
	res, err := r.db.Writer.ExecContext(ctx, `
		DELETE FROM open_tabs WHERE document_id NOT IN (SELECT id FROM documents)`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// RecordOpen upserts reading history for a document open.
func (r *ReadingRepository) RecordOpen(ctx context.Context, docID int64, scrollPct float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Writer.ExecContext(ctx, `
		INSERT INTO reading_history(document_id, last_opened_at, open_count, furthest_scroll_pct)
		VALUES (?,?,1,?)
		ON CONFLICT(document_id) DO UPDATE SET
			last_opened_at=excluded.last_opened_at,
			open_count=reading_history.open_count+1,
			furthest_scroll_pct=MAX(reading_history.furthest_scroll_pct, excluded.furthest_scroll_pct)`,
		docID, now, scrollPct)
	return err
}

// RecordFurthest updates only the monotonic furthest-scroll figure.
func (r *ReadingRepository) RecordFurthest(ctx context.Context, docID int64, pct float64) error {
	_, err := r.db.Writer.ExecContext(ctx, `
		UPDATE reading_history
		SET furthest_scroll_pct = MAX(furthest_scroll_pct, ?)
		WHERE document_id=?`, pct, docID)
	return err
}

// ListRecent returns reading history ordered by last opened, joined with doc
// metadata, for the "recently read" list.
func (r *ReadingRepository) ListRecent(ctx context.Context, limit int64) ([]domain.ReadingHistory, error) {
	rows, err := r.db.Reader.QueryContext(ctx, `
		SELECT document_id, last_opened_at, open_count, furthest_scroll_pct
		FROM reading_history ORDER BY last_opened_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.ReadingHistory
	for rows.Next() {
		var h domain.ReadingHistory
		if err := rows.Scan(&h.DocumentID, &h.LastOpenedAt, &h.OpenCount, &h.FurthestScrollPct); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
