package binding

import (
	"context"
	"log/slog"
	"time"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/usecase/search"
)

// SearchUsecase is what SearchService delegates to.
type SearchUsecase interface {
	Search(ctx context.Context, text string, scope domain.ContextType, scopeID int64, includeCode bool, limit, offset int) ([]search.Result, error)
}

// SearchService is the Wails-bound search surface.
type SearchService struct {
	usecase SearchUsecase
	logger  *slog.Logger
}

// NewSearchService constructs the bound search service.
func NewSearchService(usecase SearchUsecase, logger *slog.Logger) *SearchService {
	return &SearchService{usecase: usecase, logger: logger}
}

// Search runs a scoped query and returns results with the elapsed time.
func (s *SearchService) Search(ctx context.Context, q string, scope ScopeDTO, includeCode bool, offset int) (SearchResultsDTO, error) {
	start := time.Now()
	results, err := s.usecase.Search(ctxFromArgs(ctx), q, domain.ContextType(scope.Type), scope.ID, includeCode, 50, offset)
	if err != nil {
		return SearchResultsDTO{}, err
	}
	out := make([]SearchResultDTO, 0, len(results))
	for _, r := range results {
		out = append(out, SearchResultDTO{
			DocumentID: r.DocumentID, Title: r.Title, RelPath: r.RelPath,
			SourceName: r.SourceName, Snippet: r.Snippet, Rank: r.Rank,
		})
	}
	return SearchResultsDTO{Results: out, ElapsedMS: time.Since(start).Milliseconds()}, nil
}
