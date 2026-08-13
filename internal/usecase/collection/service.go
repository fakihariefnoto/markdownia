// Package collection implements the logical browsing axis: curated,
// document-level, many-to-many reading lists (PRD decision D5).
package collection

import (
	"context"
	"strings"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/repository/sqlite"
)

// Repo is the collection usecase's persistence view, declared here.
type Repo interface {
	List(ctx context.Context) ([]domain.Collection, error)
	GetByID(ctx context.Context, id int64) (domain.Collection, error)
	Create(ctx context.Context, c *domain.Collection) (int64, error)
	Update(ctx context.Context, c *domain.Collection) error
	Delete(ctx context.Context, id int64) error
	AddDocuments(ctx context.Context, collectionID int64, docIDs []int64) error
	RemoveDocuments(ctx context.Context, collectionID int64, docIDs []int64) error
	ReorderDocuments(ctx context.Context, collectionID int64, orderedIDs []int64) error
	ListDocuments(ctx context.Context, collectionID int64) ([]sqlite.CollectionDocumentRow, error)
	CollectionsForDocument(ctx context.Context, docID int64) ([]domain.Collection, error)
	CountDocuments(ctx context.Context, id int64) (int64, error)
}

// Collection is the DTO with its document count.
type Collection struct {
	domain.Collection
	DocumentCount int64 `json:"documentCount"`
}

// Service is the collection usecase.
type Service struct {
	repo Repo
}

// New constructs the collection usecase.
func New(repo Repo) *Service { return &Service{repo: repo} }

// ListCollections returns all collections with document counts.
func (s *Service) ListCollections(ctx context.Context) ([]Collection, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Collection, 0, len(list))
	for _, c := range list {
		count, err := s.repo.CountDocuments(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, Collection{Collection: c, DocumentCount: count})
	}
	return out, nil
}

// CreateCollection creates a named collection, rejecting duplicates.
func (s *Service) CreateCollection(ctx context.Context, name, description string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, domain.ErrInvalidArgument
	}
	id, err := s.repo.Create(ctx, &domain.Collection{Name: name, Description: description})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return 0, domain.ErrDuplicateName
		}
		return 0, err
	}
	return id, nil
}

// RenameCollection updates the display name.
func (s *Service) RenameCollection(ctx context.Context, id int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.ErrInvalidArgument
	}
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	c.Name = name
	if err := s.repo.Update(ctx, &c); err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return domain.ErrDuplicateName
	}
	return err
}

// UpdateDescription updates only the description.
func (s *Service) UpdateDescription(ctx context.Context, id int64, description string) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	c.Description = description
	return s.repo.Update(ctx, &c)
}

// DeleteCollection removes the list and its membership rows only. No document
// and no file is affected.
func (s *Service) DeleteCollection(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// AddDocuments bulk-adds memberships idempotently.
func (s *Service) AddDocuments(ctx context.Context, collectionID int64, docIDs []int64) error {
	return s.repo.AddDocuments(ctx, collectionID, docIDs)
}

// RemoveDocuments removes memberships only.
func (s *Service) RemoveDocuments(ctx context.Context, collectionID int64, docIDs []int64) error {
	return s.repo.RemoveDocuments(ctx, collectionID, docIDs)
}

// ReorderDocuments writes sort_order on the join.
func (s *Service) ReorderDocuments(ctx context.Context, collectionID int64, orderedIDs []int64) error {
	return s.repo.ReorderDocuments(ctx, collectionID, orderedIDs)
}

// ListCollectionDocuments returns memberships with source breadcrumbs.
func (s *Service) ListCollectionDocuments(ctx context.Context, id int64) ([]sqlite.CollectionDocumentRow, error) {
	return s.repo.ListDocuments(ctx, id)
}

// CollectionsForDocument is the reverse lookup for the reader toolbar.
func (s *Service) CollectionsForDocument(ctx context.Context, docID int64) ([]domain.Collection, error) {
	return s.repo.CollectionsForDocument(ctx, docID)
}
