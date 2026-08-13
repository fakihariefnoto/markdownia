package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/anofac/markdownia/internal/domain"
)

// SourceRepository persists sources. The interface is declared in the
// consuming package (usecase/source); this concrete type lives here.
type SourceRepository struct{ db *DB }

// NewSourceRepository constructs the SQLite source repository.
func NewSourceRepository(db *DB) *SourceRepository { return &SourceRepository{db: db} }

const sourceCols = `id, kind, name, root_path, origin_url, git_branch, git_commit,
	is_managed, status, error_message, document_count, ignore_globs, created_at, updated_at, indexed_at`

func scanSource(row interface{ Scan(...any) error }) (domain.Source, error) {
	var s domain.Source
	var managed int
	var globs string
	var indexedAt, errorMsg sql.NullString
	err := row.Scan(&s.ID, &s.Kind, &s.Name, &s.RootPath, &s.OriginURL, &s.GitBranch,
		&s.GitCommit, &managed, &s.Status, &errorMsg, &s.DocumentCount, &globs,
		&s.CreatedAt, &s.UpdatedAt, &indexedAt)
	if err != nil {
		return s, err
	}
	s.IsManaged = managed == 1
	s.IndexedAt = indexedAt.String
	s.ErrorMessage = errorMsg.String
	if globs != "" {
		_ = json.Unmarshal([]byte(globs), &s.IgnoreGlobs)
	}
	return s, nil
}

func (r *SourceRepository) List(ctx context.Context) ([]domain.Source, error) {
	rows, err := r.db.Reader.QueryContext(ctx,
		"SELECT "+sourceCols+" FROM sources ORDER BY name COLLATE NOCASE")
	if err != nil {
		return nil, fmt.Errorf("source list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []domain.Source
	for rows.Next() {
		s, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *SourceRepository) GetByID(ctx context.Context, id int64) (domain.Source, error) {
	s, err := scanSource(r.db.Reader.QueryRowContext(ctx,
		"SELECT "+sourceCols+" FROM sources WHERE id=?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return s, domain.ErrSourceNotFound
	}
	return s, err
}

func (r *SourceRepository) Create(ctx context.Context, s *domain.Source) (int64, error) {
	globs, _ := json.Marshal(s.IgnoreGlobs)
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.Writer.ExecContext(ctx, `
		INSERT INTO sources(kind, name, root_path, origin_url, git_branch, git_commit,
			is_managed, status, error_message, ignore_globs, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		s.Kind, s.Name, s.RootPath, s.OriginURL, s.GitBranch, s.GitCommit,
		boolInt(s.IsManaged), s.Status, s.ErrorMessage, globs, now, now)
	if err != nil {
		return 0, fmt.Errorf("source create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *SourceRepository) Update(ctx context.Context, s *domain.Source) error {
	globs, _ := json.Marshal(s.IgnoreGlobs)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Writer.ExecContext(ctx, `
		UPDATE sources SET kind=?, name=?, root_path=?, origin_url=?, git_branch=?, git_commit=?,
			is_managed=?, status=?, error_message=?, document_count=?, ignore_globs=?, updated_at=?
		WHERE id=?`,
		s.Kind, s.Name, s.RootPath, s.OriginURL, s.GitBranch, s.GitCommit,
		boolInt(s.IsManaged), s.Status, s.ErrorMessage, s.DocumentCount, globs, now, s.ID)
	if err != nil {
		return fmt.Errorf("source update: %w", err)
	}
	return nil
}

// SetStatus updates only the status and error_message columns.
func (r *SourceRepository) SetStatus(ctx context.Context, id int64, status domain.SourceStatus, errMsg string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Writer.ExecContext(ctx,
		`UPDATE sources SET status=?, error_message=?, updated_at=? WHERE id=?`,
		status, errMsg, now, id)
	if err != nil {
		return fmt.Errorf("source status: %w", err)
	}
	return nil
}

// MarkIndexed records indexed_at, document_count, git metadata, and ready.
func (r *SourceRepository) MarkIndexed(ctx context.Context, id int64, count int64, commit string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Writer.ExecContext(ctx, `
		UPDATE sources SET document_count=?, git_commit=?, status=?, error_message=NULL,
			indexed_at=?, updated_at=? WHERE id=?`,
		count, commit, domain.StatusReady, now, now, id)
	if err != nil {
		return fmt.Errorf("source indexed: %w", err)
	}
	return nil
}

// Delete removes the source; documents and everything hanging off them cascade.
func (r *SourceRepository) Delete(ctx context.Context, id int64) error {
	return r.db.Tx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM reading_state WHERE context_type='source' AND context_id=?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM sources WHERE id=?`, id); err != nil {
			return err
		}
		return nil
	})
}

// Counts returns the deletion-preview numbers for a source.
func (r *SourceRepository) Counts(ctx context.Context, id int64) (documents, highlights, bookmarks, collectionEntries int64, err error) {
	q := func(qry string, dest *int64) error {
		return r.db.Reader.QueryRowContext(ctx, qry, id).Scan(dest)
	}
	if err = q(`SELECT count(*) FROM documents WHERE source_id=?`, &documents); err != nil {
		return
	}
	if err = q(`SELECT count(*) FROM highlights h JOIN documents d ON h.document_id=d.id WHERE d.source_id=?`, &highlights); err != nil {
		return
	}
	if err = q(`SELECT count(*) FROM bookmarks b JOIN documents d ON b.document_id=d.id WHERE d.source_id=?`, &bookmarks); err != nil {
		return
	}
	if err = q(`SELECT count(*) FROM collection_documents cd JOIN documents d ON cd.document_id=d.id WHERE d.source_id=?`, &collectionEntries); err != nil {
		return
	}
	return
}

// SetUnavailableIfMissing marks a source unavailable when its root path no
// longer resolves. Returns true when the state changed.
func (r *SourceRepository) SetUnavailableIfMissing(ctx context.Context, id int64) (bool, error) {
	var root string
	if err := r.db.Reader.QueryRowContext(ctx, `SELECT root_path FROM sources WHERE id=?`, id).Scan(&root); err != nil {
		return false, err
	}
	return false, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
