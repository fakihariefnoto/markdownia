// Package search implements FTS5 query building, scoping, and snippets.
package search

import (
	"context"

	"github.com/anofac/markdownia/internal/domain"
)

// Query is a sanitized search request.
type Query struct {
	Text        string
	Scope       domain.ContextType
	ScopeID     int64
	IncludeCode bool
	Limit       int
	Offset      int
}

// Result is one ranked search hit.
type Result struct {
	DocumentID int64   `json:"documentId"`
	Title      string  `json:"title"`
	RelPath    string  `json:"relPath"`
	SourceName string  `json:"sourceName"`
	Snippet    string  `json:"snippet"`
	Rank       float64 `json:"rank"`
}

// Repo is the search usecase's persistence view, declared here.
type Repo interface {
	Search(ctx context.Context, q Query) ([]Result, error)
}

// Service is the search usecase.
type Service struct {
	repo Repo
}

// New constructs the search usecase.
func New(repo Repo) *Service { return &Service{repo: repo} }

// Search runs a scoped full-text query.
func (s *Service) Search(ctx context.Context, text string, scope domain.ContextType, scopeID int64, includeCode bool, limit, offset int) ([]Result, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	return s.repo.Search(ctx, Query{
		Text:        text,
		Scope:       scope,
		ScopeID:     scopeID,
		IncludeCode: includeCode,
		Limit:       limit,
		Offset:      offset,
	})
}
