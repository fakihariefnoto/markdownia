package binding

import (
	"context"
	"log/slog"

	"github.com/anofac/markdownia/internal/native"
)

// NativeLayer is what NativeService delegates to.
type NativeLayer interface {
	PickFolder(ctx context.Context) (string, bool, error)
	PickZipFile(ctx context.Context) (string, bool, error)
	PickSaveLocation(ctx context.Context, defaultName string) (string, bool, error)
	OpenExternal(ctx context.Context, url string) error
	RevealInFileManager(ctx context.Context, path string)
	GetWindowState(ctx context.Context) native.WindowState
	SaveWindowState(ctx context.Context, state native.WindowState) error
	CheckForUpdates(ctx context.Context, repo string) (native.UpdateResult, error)
}

// NativeService is the Wails-bound native surface.
type NativeService struct {
	native NativeLayer
	logger *slog.Logger
}

// NewNativeService constructs the bound native service.
func NewNativeService(n NativeLayer, logger *slog.Logger) *NativeService {
	return &NativeService{native: n, logger: logger}
}

func (s *NativeService) PickFolder(ctx context.Context) (string, bool, error) {
	path, ok, err := s.native.PickFolder(ctxFromArgs(ctx))
	s.logger.Info("PickFolder called", "path", path, "ok", ok, "error", err)
	return path, ok, err
}

func (s *NativeService) PickZipFile(ctx context.Context) (string, bool, error) {
	return s.native.PickZipFile(ctxFromArgs(ctx))
}

func (s *NativeService) PickSaveLocation(ctx context.Context, defaultName string) (string, bool, error) {
	return s.native.PickSaveLocation(ctxFromArgs(ctx), defaultName)
}

func (s *NativeService) OpenExternal(ctx context.Context, url string) error {
	return s.native.OpenExternal(ctxFromArgs(ctx), url)
}

func (s *NativeService) RevealInFileManager(ctx context.Context, path string) {
	s.native.RevealInFileManager(ctxFromArgs(ctx), path)
}

func (s *NativeService) GetWindowState(ctx context.Context) (WindowStateDTO, error) {
	st := s.native.GetWindowState(ctxFromArgs(ctx))
	return WindowStateDTO{Width: st.Width, Height: st.Height, X: st.X, Y: st.Y, Maximized: st.Maximized}, nil
}

func (s *NativeService) SaveWindowState(ctx context.Context, state WindowStateDTO) error {
	return s.native.SaveWindowState(ctxFromArgs(ctx), native.WindowState{
		Width: state.Width, Height: state.Height, X: state.X, Y: state.Y, Maximized: state.Maximized,
	})
}

func (s *NativeService) CheckForUpdates(ctx context.Context, repo string) (native.UpdateResult, error) {
	return s.native.CheckForUpdates(ctxFromArgs(ctx), repo)
}
