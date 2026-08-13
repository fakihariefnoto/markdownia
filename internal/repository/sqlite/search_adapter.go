package sqlite

import (
	"context"

	"github.com/anofac/markdownia/internal/usecase/search"
)

// SearchAdapter adapts the SQLite search repository to the search usecase's
// interface. It lives here so the repository layer owns the mapping.
type SearchAdapter struct{ repo *SearchRepository }

// NewSearchAdapter wraps a search repository.
func NewSearchAdapter(repo *SearchRepository) *SearchAdapter { return &SearchAdapter{repo: repo} }

// Search implements search.Repo.
func (a *SearchAdapter) Search(ctx context.Context, q search.Query) ([]search.Result, error) {
	results, err := a.repo.Search(ctx, Query{
		Text:        q.Text,
		Scope:       q.Scope,
		ScopeID:     q.ScopeID,
		IncludeCode: q.IncludeCode,
		Limit:       q.Limit,
		Offset:      q.Offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]search.Result, 0, len(results))
	for _, r := range results {
		out = append(out, search.Result{
			DocumentID: r.DocumentID,
			Title:      r.Title,
			RelPath:    r.RelPath,
			SourceName: r.SourceName,
			Snippet:    r.Snippet,
			Rank:       r.Rank,
		})
	}
	return out, nil
}
