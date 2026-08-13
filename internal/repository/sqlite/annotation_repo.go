package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/anofac/markdownia/internal/domain"
)

// AnnotationRepository persists bookmarks and highlights, and owns the sweep
// that deletes highlights whose containing block changed.
type AnnotationRepository struct{ db *DB }

// NewAnnotationRepository constructs the SQLite annotation repository.
func NewAnnotationRepository(db *DB) *AnnotationRepository { return &AnnotationRepository{db: db} }

// ---- Bookmarks ----

func (r *AnnotationRepository) AddBookmark(ctx context.Context, b *domain.Bookmark) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.Writer.ExecContext(ctx, `
		INSERT INTO bookmarks(document_id, heading_anchor, note, created_at)
		VALUES (?,?,?,?)`, b.DocumentID, nullableStr(b.HeadingAnchor), nullableStr(b.Note), now)
	if err != nil {
		return 0, fmt.Errorf("bookmark add: %w", err)
	}
	return res.LastInsertId()
}

func (r *AnnotationRepository) RemoveBookmark(ctx context.Context, id int64) error {
	_, err := r.db.Writer.ExecContext(ctx, `DELETE FROM bookmarks WHERE id=?`, id)
	return err
}

// BookmarkRow is a bookmark joined with its document.
type BookmarkRow struct {
	domain.Bookmark
	Title      string
	RelPath    string
	SourceName string
}

func (r *AnnotationRepository) ListBookmarks(ctx context.Context) ([]BookmarkRow, error) {
	rows, err := r.db.Reader.QueryContext(ctx, `
		SELECT b.id, b.document_id, b.heading_anchor, b.note, b.created_at,
		       d.title, d.rel_path, s.name
		FROM bookmarks b
		JOIN documents d ON d.id = b.document_id
		JOIN sources s ON s.id = d.source_id
		ORDER BY b.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []BookmarkRow
	for rows.Next() {
		var row BookmarkRow
		var headingAnchor, note sql.NullString
		if err := rows.Scan(&row.ID, &row.DocumentID, &headingAnchor, &note,
			&row.CreatedAt, &row.Title, &row.RelPath, &row.SourceName); err != nil {
			return nil, err
		}
		row.HeadingAnchor = headingAnchor.String
		row.Note = note.String
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *AnnotationRepository) ListBookmarksBySource(ctx context.Context, sourceID int64) ([]BookmarkRow, error) {
	rows, err := r.db.Reader.QueryContext(ctx, `
		SELECT b.id, b.document_id, b.heading_anchor, b.note, b.created_at,
		       d.title, d.rel_path, s.name
		FROM bookmarks b
		JOIN documents d ON d.id = b.document_id
		JOIN sources s ON s.id = d.source_id
		WHERE d.source_id=? ORDER BY b.created_at DESC`, sourceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []BookmarkRow
	for rows.Next() {
		var row BookmarkRow
		var headingAnchor, note sql.NullString
		if err := rows.Scan(&row.ID, &row.DocumentID, &headingAnchor, &note,
			&row.CreatedAt, &row.Title, &row.RelPath, &row.SourceName); err != nil {
			return nil, err
		}
		row.HeadingAnchor = headingAnchor.String
		row.Note = note.String
		out = append(out, row)
	}
	return out, rows.Err()
}

// ---- Highlights ----

func (r *AnnotationRepository) AddHighlight(ctx context.Context, h *domain.Highlight) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.Writer.ExecContext(ctx, `
		INSERT INTO highlights(document_id, block_hash, block_index, start_offset, end_offset,
			quoted_text, color, note, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		h.DocumentID, h.BlockHash, h.BlockIndex, h.StartOffset, h.EndOffset,
		h.QuotedText, h.Color, nullableStr(h.Note), now, now)
	if err != nil {
		return 0, fmt.Errorf("highlight add: %w", err)
	}
	return res.LastInsertId()
}

func (r *AnnotationRepository) UpdateHighlight(ctx context.Context, id int64, color *string, note *string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	q := `UPDATE highlights SET updated_at=?`
	var args []any
	args = append(args, now)
	if color != nil {
		q += `, color=?`
		args = append(args, *color)
	}
	if note != nil {
		q += `, note=?`
		args = append(args, *note)
	}
	q += ` WHERE id=?`
	args = append(args, id)
	_, err := r.db.Writer.ExecContext(ctx, q, args...)
	return err
}

func (r *AnnotationRepository) RemoveHighlight(ctx context.Context, id int64) error {
	_, err := r.db.Writer.ExecContext(ctx, `DELETE FROM highlights WHERE id=?`, id)
	return err
}

// HighlightRow is a highlight joined with its document.
type HighlightRow struct {
	domain.Highlight
	Title      string
	RelPath    string
	SourceName string
}

func (r *AnnotationRepository) ListHighlights(ctx context.Context, docID int64) ([]domain.Highlight, error) {
	rows, err := r.db.Reader.QueryContext(ctx, `
		SELECT id, document_id, block_hash, block_index, start_offset, end_offset,
			quoted_text, color, note, created_at, updated_at
		FROM highlights WHERE document_id=? ORDER BY created_at`, docID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.Highlight
	for rows.Next() {
		var h domain.Highlight
		var note sql.NullString
		if err := rows.Scan(&h.ID, &h.DocumentID, &h.BlockHash, &h.BlockIndex,
			&h.StartOffset, &h.EndOffset, &h.QuotedText, &h.Color, &note,
			&h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		h.Note = note.String
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *AnnotationRepository) ListAllHighlights(ctx context.Context) ([]HighlightRow, error) {
	rows, err := r.db.Reader.QueryContext(ctx, `
		SELECT h.id, h.document_id, h.block_hash, h.block_index, h.start_offset, h.end_offset,
			h.quoted_text, h.color, h.note, h.created_at, h.updated_at,
			d.title, d.rel_path, s.name
		FROM highlights h
		JOIN documents d ON d.id = h.document_id
		JOIN sources s ON s.id = d.source_id
		ORDER BY h.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []HighlightRow
	for rows.Next() {
 		var row HighlightRow
		var note sql.NullString
		if err := rows.Scan(&row.ID, &row.DocumentID, &row.BlockHash, &row.BlockIndex,
			&row.StartOffset, &row.EndOffset, &row.QuotedText, &row.Color, &note,
			&row.CreatedAt, &row.UpdatedAt, &row.Title, &row.RelPath, &row.SourceName); err != nil {
			return nil, err
		}
		row.Note = note.String
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *AnnotationRepository) ListHighlightsBySource(ctx context.Context, sourceID int64) ([]HighlightRow, error) {
	rows, err := r.db.Reader.QueryContext(ctx, `
		SELECT h.id, h.document_id, h.block_hash, h.block_index, h.start_offset, h.end_offset,
			h.quoted_text, h.color, h.note, h.created_at, h.updated_at,
			d.title, d.rel_path, s.name
		FROM highlights h
		JOIN documents d ON d.id = h.document_id
		JOIN sources s ON s.id = d.source_id
		WHERE d.source_id=? ORDER BY h.created_at DESC`, sourceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []HighlightRow
	for rows.Next() {
 		var row HighlightRow
		var note sql.NullString
		if err := rows.Scan(&row.ID, &row.DocumentID, &row.BlockHash, &row.BlockIndex,
			&row.StartOffset, &row.EndOffset, &row.QuotedText, &row.Color, &note,
			&row.CreatedAt, &row.UpdatedAt, &row.Title, &row.RelPath, &row.SourceName); err != nil {
			return nil, err
		}
		row.Note = note.String
		out = append(out, row)
	}
	return out, rows.Err()
}

// SweepHighlights deletes highlights whose block_hash is absent from keep,
// for one document, inside the caller's transaction. Returns the count removed.
func (r *AnnotationRepository) SweepHighlights(ctx context.Context, tx *sql.Tx, docID int64, keep map[string]bool) (int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, block_hash FROM highlights WHERE document_id=?`, docID)
	if err != nil {
		return 0, err
	}
	var toDelete []int64
	for rows.Next() {
		var id int64
		var h string
		if err := rows.Scan(&id, &h); err != nil {
			_ = rows.Close()
			return 0, err
		}
		if !keep[h] {
			toDelete = append(toDelete, id)
		}
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, id := range toDelete {
		if _, err := tx.ExecContext(ctx, `DELETE FROM highlights WHERE id=?`, id); err != nil {
			return 0, err
		}
	}
	return int64(len(toDelete)), nil
}

// SweepDocumentHighlights deletes highlights whose block_hash is absent from
// keep, for one document, in its own transaction. This is the indexer's hook
// into the highlight sweep.
func (r *AnnotationRepository) SweepDocumentHighlights(ctx context.Context, docID int64, keep map[string]bool) (int64, error) {
	var removed int64
	err := r.db.Tx(ctx, func(tx *sql.Tx) error {
		var err error
		removed, err = r.SweepHighlights(ctx, tx, docID, keep)
		return err
	})
	return removed, err
}

// HighlightByID fetches a single highlight, used by the annotation usecase.
func (r *AnnotationRepository) HighlightByID(ctx context.Context, id int64) (domain.Highlight, error) {
	var h domain.Highlight
	var note sql.NullString
	err := r.db.Reader.QueryRowContext(ctx, `
		SELECT id, document_id, block_hash, block_index, start_offset, end_offset,
			quoted_text, color, note, created_at, updated_at
		FROM highlights WHERE id=?`, id).
		Scan(&h.ID, &h.DocumentID, &h.BlockHash, &h.BlockIndex, &h.StartOffset,
			&h.EndOffset, &h.QuotedText, &h.Color, &note, &h.CreatedAt, &h.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return h, fmt.Errorf("highlight not found")
	}
	h.Note = note.String
	return h, err
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
