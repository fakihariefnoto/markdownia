package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/anofac/markdownia/internal/domain"
)

// DocumentRepository persists documents and their caches.
type DocumentRepository struct{ db *DB }

// NewDocumentRepository constructs the SQLite document repository.
func NewDocumentRepository(db *DB) *DocumentRepository { return &DocumentRepository{db: db} }

const docCols = `id, source_id, rel_path, title, title_source, file_hash, file_mtime,
	file_size, rendered_html, plain_text, code_text, outline_json, frontmatter_json,
	renderer_version, word_count, has_mermaid, indexed_at`

func scanDocument(row interface{ Scan(...any) error }) (domain.Document, error) {
	var d domain.Document
	var outline string
	var frontmatter sql.NullString
	var hasMermaid int
	err := row.Scan(&d.ID, &d.SourceID, &d.RelPath, &d.Title, &d.TitleSource, &d.FileHash,
		&d.FileMtime, &d.FileSize, &d.RenderedHTML, &d.PlainText, &d.CodeText, &outline,
		&frontmatter, &d.RendererVersion, &d.WordCount, &hasMermaid, &d.IndexedAt)
	if err != nil {
		return d, err
	}
	d.HasMermaid = hasMermaid == 1
	if outline != "" {
		_ = json.Unmarshal([]byte(outline), &d.Outline)
	}
	if frontmatter.Valid && frontmatter.String != "" {
		d.Frontmatter = json.RawMessage(frontmatter.String)
	}
	return d, nil
}

// GetByID returns a full document including the rendered HTML — the hot path.
func (r *DocumentRepository) GetByID(ctx context.Context, id int64) (domain.Document, error) {
	d, err := scanDocument(r.db.Reader.QueryRowContext(ctx,
		"SELECT "+docCols+" FROM documents WHERE id=?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return d, domain.ErrDocumentNotFound
	}
	return d, err
}

// MetaByID returns a document without the heavy columns — for tooltips and rows.
func (r *DocumentRepository) MetaByID(ctx context.Context, id int64) (domain.Document, error) {
	var d domain.Document
	var frontmatter sql.NullString
	var hasMermaid int
	err := r.db.Reader.QueryRowContext(ctx, `
		SELECT id, source_id, rel_path, title, title_source, file_hash, file_mtime,
			file_size, renderer_version, word_count, has_mermaid, indexed_at, frontmatter_json
		FROM documents WHERE id=?`, id).
		Scan(&d.ID, &d.SourceID, &d.RelPath, &d.Title, &d.TitleSource, &d.FileHash,
			&d.FileMtime, &d.FileSize, &d.RendererVersion, &d.WordCount, &hasMermaid,
			&d.IndexedAt, &frontmatter)
	if errors.Is(err, sql.ErrNoRows) {
		return d, domain.ErrDocumentNotFound
	}
	d.HasMermaid = hasMermaid == 1
	if frontmatter.Valid && frontmatter.String != "" {
		d.Frontmatter = json.RawMessage(frontmatter.String)
	}
	return d, err
}

// FileState is the cheap re-index skip check: exists + mtime + size + hash +
// renderer version for one (source, rel_path).
type FileState struct {
	ID              int64
	FileHash        string
	FileMtime       int64
	FileSize        int64
	RendererVersion int
}

// GetFileState returns the stored file metadata for a rel_path, or ok=false.
func (r *DocumentRepository) GetFileState(ctx context.Context, sourceID int64, relPath string) (FileState, bool, error) {
	var f FileState
	err := r.db.Reader.QueryRowContext(ctx, `
		SELECT id, file_hash, file_mtime, file_size, renderer_version
		FROM documents WHERE source_id=? AND rel_path=?`, sourceID, relPath).
		Scan(&f.ID, &f.FileHash, &f.FileMtime, &f.FileSize, &f.RendererVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return f, false, nil
	}
	return f, true, err
}

// Upsert inserts or replaces a document and its FTS row in one transaction
// (batching is the caller's responsibility; each call is one transaction).
func (r *DocumentRepository) Upsert(ctx context.Context, d *domain.Document) error {
	now := time.Now().UTC().Format(time.RFC3339)
	outline, _ := json.Marshal(d.Outline)
	frontmatter := nullableJSON(d.Frontmatter)
	headings := headingsFromOutline(d.Outline)

	return r.db.Tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			INSERT INTO documents(source_id, rel_path, title, title_source, file_hash, file_mtime,
				file_size, rendered_html, plain_text, code_text, outline_json, frontmatter_json,
				renderer_version, word_count, has_mermaid, indexed_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT(source_id, rel_path) DO UPDATE SET
				title=excluded.title, title_source=excluded.title_source, file_hash=excluded.file_hash,
				file_mtime=excluded.file_mtime, file_size=excluded.file_size,
				rendered_html=excluded.rendered_html, plain_text=excluded.plain_text,
				code_text=excluded.code_text, outline_json=excluded.outline_json,
				frontmatter_json=excluded.frontmatter_json, renderer_version=excluded.renderer_version,
				word_count=excluded.word_count, has_mermaid=excluded.has_mermaid, indexed_at=excluded.indexed_at`,
			d.SourceID, d.RelPath, d.Title, d.TitleSource, d.FileHash, d.FileMtime, d.FileSize,
			d.RenderedHTML, d.PlainText, d.CodeText, string(outline), frontmatter,
			d.RendererVersion, d.WordCount, boolInt(d.HasMermaid), now)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		if id == 0 {
			if err := tx.QueryRowContext(ctx,
				`SELECT id FROM documents WHERE source_id=? AND rel_path=?`,
				d.SourceID, d.RelPath).Scan(&id); err != nil {
				return err
			}
		}
		d.ID = id

		// FTS row maintained explicitly, in the same transaction, no triggers.
		// FTS5 virtual tables reject UPSERT; INSERT OR REPLACE overwrites by rowid.
		if _, err := tx.ExecContext(ctx, `
			INSERT OR REPLACE INTO documents_fts(rowid, title, headings, body, code)
			VALUES (?,?,?,?,?)`,
			id, d.Title, headings, d.PlainText, d.CodeText); err != nil {
			return err
		}
		return nil
	})
}

// ReRenderHTML rewrites only the cached HTML and renderer version, keeping the
// source-derived columns intact (used on renderer_version mismatch).
func (r *DocumentRepository) ReRenderHTML(ctx context.Context, id int64, html string, version int) error {
	return r.db.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE documents SET rendered_html=?, renderer_version=? WHERE id=?`,
			html, version, id); err != nil {
			return err
		}
		// FTS rows unchanged: indexed text did not change.
		return nil
	})
}

// TouchMtime updates mtime after a content-hash check found no change.
func (r *DocumentRepository) TouchMtime(ctx context.Context, id int64, mtime int64) error {
	_, err := r.db.Writer.ExecContext(ctx,
		`UPDATE documents SET file_mtime=? WHERE id=?`, mtime, id)
	return err
}

// ListBySource returns metadata for all documents in a source, without HTML.
func (r *DocumentRepository) ListBySource(ctx context.Context, sourceID int64) ([]domain.Document, error) {
	rows, err := r.db.Reader.QueryContext(ctx, `
		SELECT id, source_id, rel_path, title, title_source, file_hash, file_mtime,
			file_size, renderer_version, word_count, has_mermaid, indexed_at
		FROM documents WHERE source_id=? ORDER BY rel_path`, sourceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.Document
	for rows.Next() {
		var d domain.Document
		var hasMermaid int
		if err := rows.Scan(&d.ID, &d.SourceID, &d.RelPath, &d.Title, &d.TitleSource,
			&d.FileHash, &d.FileMtime, &d.FileSize, &d.RendererVersion, &d.WordCount,
			&hasMermaid, &d.IndexedAt); err != nil {
			return nil, err
		}
		d.HasMermaid = hasMermaid == 1
		out = append(out, d)
	}
	return out, rows.Err()
}

// DeleteMissing removes documents whose rel_path is not in keep, cascading
// highlights, bookmarks, tabs, collection entries, and history.
func (r *DocumentRepository) DeleteMissing(ctx context.Context, sourceID int64, keep map[string]bool) (int64, error) {
	var deleted int64
	err := r.db.Tx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT id, rel_path FROM documents WHERE source_id=?`, sourceID)
		if err != nil {
			return err
		}
		var toDelete []int64
		for rows.Next() {
			var id int64
			var rel string
			if err := rows.Scan(&id, &rel); err != nil {
				_ = rows.Close()
				return err
			}
			if !keep[rel] {
				toDelete = append(toDelete, id)
			}
		}
		_ = rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, id := range toDelete {
			if _, err := tx.ExecContext(ctx, `DELETE FROM documents WHERE id=?`, id); err != nil {
				return err
			}
			deleted++
		}
		return nil
	})
	return deleted, err
}

// CountBySource returns the number of documents in a source.
func (r *DocumentRepository) CountBySource(ctx context.Context, sourceID int64) (int64, error) {
	var n int64
	err := r.db.Reader.QueryRowContext(ctx,
		`SELECT count(*) FROM documents WHERE source_id=?`, sourceID).Scan(&n)
	return n, err
}

// ListIDsBySource returns every document id for a source.
func (r *DocumentRepository) ListIDsBySource(ctx context.Context, sourceID int64) ([]int64, error) {
	rows, err := r.db.Reader.QueryContext(ctx,
		`SELECT id FROM documents WHERE source_id=?`, sourceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// GetBlockHashByDocID retrieves the block-hash column of a specific document
// (used by the annotation anchor validator).
func (r *DocumentRepository) GetBySourceAndPath(ctx context.Context, sourceID int64, relPath string) (domain.Document, error) {
	d, err := scanDocument(r.db.Reader.QueryRowContext(ctx,
		"SELECT "+docCols+" FROM documents WHERE source_id=? AND rel_path=?", sourceID, relPath))
	if errors.Is(err, sql.ErrNoRows) {
		return d, domain.ErrDocumentNotFound
	}
	return d, err
}

func nullableJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	return string(raw)
}

func headingsFromOutline(o []domain.OutlineHeading) string {
	var s string
	for _, h := range o {
		if s != "" {
			s += " "
		}
		s += h.Text
	}
	return s
}
