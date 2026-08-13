package binding

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/anofac/markdownia/internal/domain"
)

// ReadingUsecase is what ReadingService delegates to.
type ReadingUsecase interface {
	GetReadingState(ctx context.Context, ctxType domain.ContextType, ctxID int64) (domain.ReadingState, error)
	SaveScrollPosition(ctx context.Context, ctxType domain.ContextType, ctxID, docID int64, pct float64) error
	GetOpenTabs(ctx context.Context, ctxType domain.ContextType, ctxID int64) ([]domain.OpenTab, error)
	SaveOpenTabs(ctx context.Context, ctxType domain.ContextType, ctxID int64, tabs []domain.OpenTab) error
	GetLastContext(ctx context.Context) (domain.ReadingState, bool, error)
	CleanupDeletedTabs(ctx context.Context) (int64, error)
}

// SettingsUsecase is what SettingsService delegates to.
type SettingsUsecase interface {
	GetAll(ctx context.Context) (map[string]json.RawMessage, error)
	Get(ctx context.Context, key string) (json.RawMessage, bool, error)
	Set(ctx context.Context, key, value string) error
	Reset(ctx context.Context, key string) error
}

// ReadingService is the Wails-bound reading-surface.
type ReadingService struct {
	usecase ReadingUsecase
	logger  *slog.Logger
}

// NewReadingService constructs the bound reading service.
func NewReadingService(usecase ReadingUsecase, logger *slog.Logger) *ReadingService {
	return &ReadingService{usecase: usecase, logger: logger}
}

func (s *ReadingService) GetReadingState(ctx context.Context, contextType string, contextID int64) (ReadingStateDTO, error) {
	ct := domain.ContextType(contextType)
	st, err := s.usecase.GetReadingState(ctxFromArgs(ctx), ct, contextID)
	if err != nil {
		// No row yet is a normal first visit, not an error.
		if err.Error() == "sql: no rows in result set" {
			return ReadingStateDTO{ContextType: contextType, ContextID: contextID}, nil
		}
		return ReadingStateDTO{}, err
	}
	return ReadingStateDTO{
		ContextType: string(st.ContextType), ContextID: st.ContextID,
		DocumentID: st.DocumentID, ScrollPct: st.ScrollPct,
	}, nil
}

func (s *ReadingService) SaveScrollPosition(ctx context.Context, contextType string, contextID, docID int64, pct float64) error {
	return s.usecase.SaveScrollPosition(ctxFromArgs(ctx), domain.ContextType(contextType), contextID, docID, pct)
}

func (s *ReadingService) GetOpenTabs(ctx context.Context, contextType string, contextID int64) ([]OpenTabDTO, error) {
	tabs, err := s.usecase.GetOpenTabs(ctxFromArgs(ctx), domain.ContextType(contextType), contextID)
	if err != nil {
		return nil, err
	}
	out := make([]OpenTabDTO, 0, len(tabs))
	for _, t := range tabs {
		out = append(out, OpenTabDTO{DocumentID: t.DocumentID, Pane: t.Pane, IsActive: t.IsActive, Title: t.Title, RelPath: t.RelPath})
	}
	return out, nil
}

func (s *ReadingService) SaveOpenTabs(ctx context.Context, contextType string, contextID int64, tabs []OpenTabDTO) error {
	converted := make([]domain.OpenTab, 0, len(tabs))
	for _, t := range tabs {
		converted = append(converted, domain.OpenTab{
			ContextType: domain.ContextType(contextType), ContextID: contextID,
			DocumentID: t.DocumentID, Pane: t.Pane, IsActive: t.IsActive,
		})
	}
	return s.usecase.SaveOpenTabs(ctxFromArgs(ctx), domain.ContextType(contextType), contextID, converted)
}

// GetLastContext resolves launch restore.
func (s *ReadingService) GetLastContext(ctx context.Context) (ReadingStateDTO, error) {
	st, ok, err := s.usecase.GetLastContext(ctxFromArgs(ctx))
	if err != nil || !ok {
		return ReadingStateDTO{}, err
	}
	return ReadingStateDTO{
		ContextType: string(st.ContextType), ContextID: st.ContextID,
		DocumentID: st.DocumentID, ScrollPct: st.ScrollPct,
	}, nil
}

// CleanupDeletedTabs is called after source deletion and re-indexes.
func (s *ReadingService) CleanupDeletedTabs(ctx context.Context) (int64, error) {
	return s.usecase.CleanupDeletedTabs(ctxFromArgs(ctx))
}

// SettingsService is the Wails-bound settings surface.
type SettingsService struct {
	usecase SettingsUsecase
	logger  *slog.Logger
}

// NewSettingsService constructs the bound settings service.
func NewSettingsService(usecase SettingsUsecase, logger *slog.Logger) *SettingsService {
	return &SettingsService{usecase: usecase, logger: logger}
}

func (s *SettingsService) GetAll(ctx context.Context) (map[string]json.RawMessage, error) {
	return s.usecase.GetAll(ctxFromArgs(ctx))
}

func (s *SettingsService) Get(ctx context.Context, key string) (json.RawMessage, error) {
	v, ok, err := s.usecase.Get(ctxFromArgs(ctx), key)
	if err != nil || !ok {
		return nil, err
	}
	return v, nil
}

func (s *SettingsService) Set(ctx context.Context, key, value string) error {
	return s.usecase.Set(ctxFromArgs(ctx), key, value)
}

func (s *SettingsService) Reset(ctx context.Context, key string) error {
	return s.usecase.Reset(ctxFromArgs(ctx), key)
}
