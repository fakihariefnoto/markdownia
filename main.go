// Markdownia — an offline-first desktop reader for markdown libraries.
//
// This is the Wails 3 entrypoint. The Go layer is wired in internal/app; the
// Wails-coupled collaborators (dialogs, window, menu, browser, event
// emission) live in internal/native/wails — the only places that import
// Wails APIs (ADR A10).
//
// Wails version pinned: v3.0.0-beta.7. Frontend build output is embedded
// from frontend/dist (produced by the web/ Bun build).

package main

import (
	"context"
	"embed"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/anofac/markdownia/internal/app"
	"github.com/anofac/markdownia/internal/native"
	"github.com/anofac/markdownia/internal/native/wails"
	"github.com/anofac/markdownia/internal/platform"
	"github.com/anofac/markdownia/internal/repository/sqlite"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIconData []byte

// appicon returns the embedded app icon bytes.
func appicon() ([]byte, error) {
	return appIconData, nil
}

// version is stamped at build time via ldflags from the git tag.
var version = "dev"

func main() {
	native.AppVersion = version

	cfg, err := app.Resolve()
	if err != nil {
		slog.Error("app config resolution failed", "error", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		slog.Error("app config invalid", "error", err)
		os.Exit(1)
	}

	logger, logFile, err := platform.Logger(cfg.AppDir, slog.LevelInfo)
	if err != nil {
		slog.Error("logging setup failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = logFile.Close() }()

	db, err := sqlite.Open(context.Background(), cfg.DBPath, logger)
	if err != nil {
		logger.Error("database open failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	appInstance := application.New(application.Options{
		Name:        "Markdownia",
		Description: "Turn any pile of markdown into a beautifully rendered reading library.",
		Services:    []application.Service{},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// App + window icon from the embedded master.
	if icon, err := appicon(); err == nil {
		appInstance.SetIcon(icon)
	}

	// Window: 1280×800 first launch, centered; 900×600 minimum enforced.
	mainWindow := appInstance.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "Markdownia",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		// Frameless: no native chrome on any platform. The frontend draws its
		// own titlebar + window controls (close/minimise/maximise) themed to the
		// app, with --wails-draggable regions for dragging. On macOS the native
		// frame is kept (rounded corners/shadow) and the traffic lights hidden.
		Frameless: true,
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(249, 250, 251), // color.background light
		URL:              "/",
	})

	// Wails-coupled collaborators (the only Wails imports in the tree).
	events := wails.NewEvents(appInstance)
	dialog := wails.NewDialog(appInstance, mainWindow)
	window := wails.NewWindow(appInstance, mainWindow)
	browser := wails.NewBrowser(appInstance)

	// Wire the whole Go layer.
	svcs, err := app.Build(db, events, &nativeLayer{
		native: native.New(events, &settingsAdapter{db: db}, dialog, window, browser, logger),
	}, cfg.ExtractedZip, &settingsAdapter{db: db}, logger)
	if err != nil {
		logger.Error("service wiring failed", "error", err)
		os.Exit(1)
	}

	// Register the bound services with Wails.
	appInstance.RegisterService(application.NewService(svcs.Source))
	appInstance.RegisterService(application.NewService(svcs.Library))
	appInstance.RegisterService(application.NewService(svcs.Search))
	appInstance.RegisterService(application.NewService(svcs.Collection))
	appInstance.RegisterService(application.NewService(svcs.Annotation))
	appInstance.RegisterService(application.NewService(svcs.Reading))
	appInstance.RegisterService(application.NewService(svcs.Settings))
	appInstance.RegisterService(application.NewService(svcs.Export))
	appInstance.RegisterService(application.NewService(svcs.Native))

	// Native menu (see desktop/tasks/02-window-shell.md).
	buildMenu(appInstance, svcs.Native)

	// Restore window state from settings, clamped to the current display set.
	restoreWindowState(mainWindow, window, db, logger)

	if err := appInstance.Run(); err != nil {
		logger.Error("application run failed", "error", err)
		os.Exit(1)
	}
}

// nativeLayer adapts the Wails-built native.Native to binding.NativeLayer.
type nativeLayer struct {
	native *native.Native
}

func (n *nativeLayer) PickFolder(ctx context.Context) (string, bool, error) {
	return n.native.PickFolder(ctx)
}
func (n *nativeLayer) PickZipFile(ctx context.Context) (string, bool, error) {
	return n.native.PickZipFile(ctx)
}
func (n *nativeLayer) PickSaveLocation(ctx context.Context, defaultName string) (string, bool, error) {
	return n.native.PickSaveLocation(ctx, defaultName)
}
func (n *nativeLayer) OpenExternal(ctx context.Context, url string) error {
	return n.native.OpenExternal(ctx, url)
}
func (n *nativeLayer) RevealInFileManager(ctx context.Context, path string) {
	n.native.RevealInFileManager(ctx, path)
}
func (n *nativeLayer) GetWindowState(ctx context.Context) native.WindowState {
	return n.native.GetWindowState(ctx)
}
func (n *nativeLayer) SaveWindowState(ctx context.Context, state native.WindowState) error {
	return n.native.SaveWindowState(ctx, state)
}
func (n *nativeLayer) CheckForUpdates(ctx context.Context, repo string) (native.UpdateResult, error) {
	return n.native.CheckForUpdates(ctx, repo)
}

// settingsAdapter bridges the SQLite settings repo to native.SettingsRepo.
type settingsAdapter struct{ db *sqlite.DB }

func (s *settingsAdapter) Get(ctx context.Context, key string) (json.RawMessage, bool, error) {
	return sqlite.NewSettingsRepository(s.db).Get(ctx, key)
}
func (s *settingsAdapter) Set(ctx context.Context, key string, value json.RawMessage) error {
	return sqlite.NewSettingsRepository(s.db).Set(ctx, key, value)
}

func restoreWindowState(w *application.WebviewWindow, win *wails.Window, db *sqlite.DB, logger *slog.Logger) {
	ctx := context.Background()
	repo := sqlite.NewSettingsRepository(db)
	raw, ok, err := repo.Get(ctx, "window.state")
	if err != nil || !ok {
		return
	}
	var state native.WindowState
	if err := json.Unmarshal(raw, &state); err != nil {
		return
	}
	state = win.ClampToDisplays(state)
	w.SetPosition(state.X, state.Y)
	w.SetSize(state.Width, state.Height)
	if state.Maximized {
		w.Maximise()
	}
	logger.Info("window state restored", "state", state)
}
