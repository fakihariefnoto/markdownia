package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/anofac/markdownia/internal/domain"
)

// CollectionRepository persists collections and their many-to-many membership.
type CollectionRepository struct{ db *DB }

// NewCollectionRepository constructs the SQLite collection repository.
func NewCollectionRepository(db *DB) *CollectionRepository { return &CollectionRepository{db: db} }

func (r *CollectionRepository) List(ctx context.Context) ([]domain.Collection, error) {
	rows, err := r.db.Reader.QueryContext(ctx, `
		SELECT c.id, c.name, c.description, c.icon, c.sort_order, c.created_at, c.updated_at,
		       (SELECT count(*) FROM collection_documents cd WHERE cd.collection_id=c.id) AS doc_count
		FROM collections c ORDER BY c.sort_order, c.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.Collection
	for rows.Next() {
		var c domain.Collection
		var count int64
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Icon, &c.SortOrder,
			&c.CreatedAt, &c.UpdatedAt, &count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CollectionRepository) GetByID(ctx context.Context, id int64) (domain.Collection, error) {
	var c domain.Collection
	var description, icon sql.NullString
	err := r.db.Reader.QueryRowContext(ctx, `
		SELECT id, name, description, icon, sort_order, created_at, updated_at
		FROM collections WHERE id=?`, id).
		Scan(&c.ID, &c.Name, &description, &icon, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return c, domain.ErrCollectionNotFound
	}
	c.Description = description.String
	c.Icon = icon.String
	return c, err
}

func (r *CollectionRepository) Create(ctx context.Context, c *domain.Collection) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.Writer.ExecContext(ctx, `
		INSERT INTO collections(name, description, icon, sort_order, created_at, updated_at)
		VALUES (?,?,?,?,?,?)`,
		c.Name, c.Description, c.Icon, c.SortOrder, now, now)
	if err != nil {
		return 0, fmt.Errorf("collection create: %w", err)
	}
	return res.LastInsertId()
}

func (r *CollectionRepository) Update(ctx context.Context, c *domain.Collection) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Writer.ExecContext(ctx, `
		UPDATE collections SET name=?, description=?, icon=?, sort_order=?, updated_at=? WHERE id=?`,
		c.Name, c.Description, c.Icon, c.SortOrder, now, c.ID)
	return err
}

func (r *CollectionRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Writer.ExecContext(ctx, `DELETE FROM collections WHERE id=?`, id)
	return err
}

// AddDocuments bulk-adds memberships idempotently; re-adding is a no-op.
func (r *CollectionRepository) AddDocuments(ctx context.Context, collectionID int64, docIDs []int64) error {
	return r.db.Tx(ctx, func(tx *sql.Tx) error {
		now := time.Now().UTC().Format(time.RFC3339)
		for i, id := range docIDs {
			if _, err := tx.ExecContext(ctx, `
				INSERT OR IGNORE INTO collection_documents(collection_id, document_id, sort_order, added_at)
				VALUES (?,?,?,?)`, collectionID, id, i, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *CollectionRepository) RemoveDocuments(ctx context.Context, collectionID int64, docIDs []int64) error {
	for _, id := range docIDs {
		if _, err := r.db.Writer.ExecContext(ctx, `
			DELETE FROM collection_documents WHERE collection_id=? AND document_id=?`,
			collectionID, id); err != nil {
			return err
		}
	}
	return nil
}

// ReorderDocuments rewrites sort_order for the ordered document ids.
func (r *CollectionRepository) ReorderDocuments(ctx context.Context, collectionID int64, orderedIDs []int64) error {
	return r.db.Tx(ctx, func(tx *sql.Tx) error {
		for i, id := range orderedIDs {
			if _, err := tx.ExecContext(ctx, `
				UPDATE collection_documents SET sort_order=? WHERE collection_id=? AND document_id=?`,
				i, collectionID, id); err != nil {
				return err
			}
		}
		return nil
	})
}

// CollectionDocumentRow is a membership row joined with its document and source
// so the collection view shows cross-source membership at a glance.
type CollectionDocumentRow struct {
	domain.CollectionDocument
	Title       string
	RelPath     string
	SourceName  string
}

// CountDocuments returns the number of documents in a collection.
func (r *CollectionRepository) CountDocuments(ctx context.Context, id int64) (int64, error) {
	var n int64
	err := r.db.Reader.QueryRowContext(ctx,
		`SELECT count(*) FROM collection_documents WHERE collection_id=?`, id).Scan(&n)
	return n, err
}

// ListDocuments returns memberships in order, with source breadcrumbs.
func (r *CollectionRepository) ListDocuments(ctx context.Context, collectionID int64) ([]CollectionDocumentRow, error) {
	rows, err := r.db.Reader.QueryContext(ctx, `
		SELECT cd.collection_id, cd.document_id, cd.sort_order, cd.added_at,
		       d.title, d.rel_path, s.name
		FROM collection_documents cd
		JOIN documents d ON d.id = cd.document_id
		JOIN sources s ON s.id = d.source_id
		WHERE cd.collection_id=?
		ORDER BY cd.sort_order, d.title COLLATE NOCASE`, collectionID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []CollectionDocumentRow
	for rows.Next() {
		var row CollectionDocumentRow
		if err := rows.Scan(&row.CollectionID, &row.DocumentID, &row.SortOrder, &row.AddedAt,
			&row.Title, &row.RelPath, &row.SourceName); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// CollectionsForDocument is the reverse lookup for the reader toolbar.
func (r *CollectionRepository) CollectionsForDocument(ctx context.Context, docID int64) ([]domain.Collection, error) {
	rows, err := r.db.Reader.QueryContext(ctx, `
		SELECT c.id, c.name, c.description, c.icon, c.sort_order, c.created_at, c.updated_at
		FROM collections c
		JOIN collection_documents cd ON cd.collection_id = c.id
		WHERE cd.document_id=? ORDER BY c.name COLLATE NOCASE`, docID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []domain.Collection
	for rows.Next() {
		var c domain.Collection
		var description, icon sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &description, &icon, &c.SortOrder,
			&c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.Description = description.String
		c.Icon = icon.String
		out = append(out, c)
	}
	return out, rows.Err()
}
