// Package reading implements reading state, open tabs, history, and the flat
// settings key/value store.
package reading

import (
	"context"
	"encoding/json"

	"github.com/anofac/markdownia/internal/domain"
)

// Repo is the reading usecase's persistence view, declared here.
type Repo interface {
	GetReadingState(ctx context.Context, ctxType domain.ContextType, ctxID int64) (domain.ReadingState, error)
	SaveScrollPosition(ctx context.Context, ctxType domain.ContextType, ctxID, docID int64, pct float64) error
	GetOpenTabs(ctx context.Context, ctxType domain.ContextType, ctxID int64) ([]domain.OpenTab, error)
	SaveOpenTabs(ctx context.Context, ctxType domain.ContextType, ctxID int64, tabs []domain.OpenTab) error
	GetLastContext(ctx context.Context) (domain.ReadingState, bool, error)
	CleanupDeletedTabs(ctx context.Context) (int64, error)
	RecordFurthest(ctx context.Context, docID int64, pct float64) error
}

// SettingsRepo is the flat KV store.
type SettingsRepo interface {
	GetAll(ctx context.Context) (map[string]json.RawMessage, error)
	Get(ctx context.Context, key string) (json.RawMessage, bool, error)
	Set(ctx context.Context, key string, value json.RawMessage) error
	Reset(ctx context.Context, key string) error
}

// DocumentRepo reports whether a document exists (for tab cleanup).
type DocumentRepo interface {
	GetByID(ctx context.Context, id int64) (domain.Document, error)
}

// Service is the reading + settings usecase.
type Service struct {
	repo    Repo
	settings SettingsRepo
	docs    DocumentRepo
}

// New constructs the reading/settings usecase.
func New(repo Repo, settings SettingsRepo, docs DocumentRepo) *Service {
	return &Service{repo: repo, settings: settings, docs: docs}
}

// GetReadingState returns the resume point for a context.
func (s *Service) GetReadingState(ctx context.Context, ctxType domain.ContextType, ctxID int64) (domain.ReadingState, error) {
	return s.repo.GetReadingState(ctx, ctxType, ctxID)
}

// SaveScrollPosition stores a fraction so restore survives size changes.
func (s *Service) SaveScrollPosition(ctx context.Context, ctxType domain.ContextType, ctxID, docID int64, pct float64) error {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	if err := s.repo.SaveScrollPosition(ctx, ctxType, ctxID, docID, pct); err != nil {
		return err
	}
	_ = s.repo.RecordFurthest(ctx, docID, pct)
	return nil
}

// GetOpenTabs returns a context's tab set.
func (s *Service) GetOpenTabs(ctx context.Context, ctxType domain.ContextType, ctxID int64) ([]domain.OpenTab, error) {
	return s.repo.GetOpenTabs(ctx, ctxType, ctxID)
}

// SaveOpenTabs replaces a context's tab set.
func (s *Service) SaveOpenTabs(ctx context.Context, ctxType domain.ContextType, ctxID int64, tabs []domain.OpenTab) error {
	return s.repo.SaveOpenTabs(ctx, ctxType, ctxID, tabs)
}

// GetLastContext resolves launch restore: last reading state → library home →
// first-run (decided by the caller counting sources).
func (s *Service) GetLastContext(ctx context.Context) (domain.ReadingState, bool, error) {
	return s.repo.GetLastContext(ctx)
}

// CleanupDeletedTabs removes tabs pointing at documents that no longer exist.
func (s *Service) CleanupDeletedTabs(ctx context.Context) (int64, error) {
	return s.repo.CleanupDeletedTabs(ctx)
}

// ---- Settings ----

// GetAll returns every setting with defaults seeded for known keys.
func (s *Service) GetAll(ctx context.Context) (map[string]json.RawMessage, error) {
	all, err := s.settings.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]json.RawMessage{}
	for k, v := range all {
		out[k] = v
	}
	// Seed defaults so the frontend never hardcodes a fallback.
	for k, def := range DefaultSettings() {
		if _, ok := out[k]; !ok {
			b, _ := json.Marshal(def)
			out[k] = b
		}
	}
	return out, nil
}

// DefaultSettings returns the first-run defaults for every known key.
func DefaultSettings() map[string]any {
	return map[string]any{
		"appearance.mode":     "system",
		"appearance.accent":   "teal",
		"reading.theme":       "paper",
		"reading.font":        "sans",
		"reading.size":        1.0,
		"reading.measure":     "72ch",
		"window.state":        nil,
		"panes.sidebar_width": 280.0,
		"panes.outline_width": 240.0,
		"search.include_code": false,
		"updates.last_checked_at": nil,
	}
}

// Get returns one setting.
func (s *Service) Get(ctx context.Context, key string) (json.RawMessage, bool, error) {
	v, ok, err := s.settings.Get(ctx, key)
	if err != nil {
		return nil, false, err
	}
	if ok {
		return v, true, nil
	}
	if def, exists := DefaultSettings()[key]; exists {
		b, _ := json.Marshal(def)
		return b, true, nil
	}
	return nil, false, nil
}

// Set writes a validated setting.
func (s *Service) Set(ctx context.Context, key, rawValue string) error {
	var value json.RawMessage
	if err := json.Unmarshal([]byte(rawValue), &value); err != nil {
		return err
	}
	if err := validate(key, value); err != nil {
		return err
	}
	return s.settings.Set(ctx, key, value)
}

// Reset removes a setting.
func (s *Service) Reset(ctx context.Context, key string) error {
	return s.settings.Reset(ctx, key)
}

// validate rejects unknown theme/accent/measure names.
func validate(key string, v json.RawMessage) error {
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return nil // non-string values (numbers) are fine
	}
	switch key {
	case "reading.theme":
		switch s {
		case "paper", "sepia", "solarized", "nord", "dracula", "gruvbox":
			return nil
		}
		return domain.ErrInvalidArgument
	case "appearance.accent":
		switch s {
		case "teal", "indigo", "forest", "copper", "plum", "slate":
			return nil
		}
		return domain.ErrInvalidArgument
	case "reading.measure":
		switch s {
		case "60ch", "72ch", "88ch", "full":
			return nil
		}
		return domain.ErrInvalidArgument
	case "reading.font":
		switch s {
		case "sans", "serif":
			return nil
		}
		return domain.ErrInvalidArgument
	case "appearance.mode":
		switch s {
		case "light", "dark", "system":
			return nil
		}
		return domain.ErrInvalidArgument
	}
	return nil
}
