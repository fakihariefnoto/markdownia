// Package wails implements the native package's collaborator interfaces on
// top of Wails 3. This is the ONLY place, alongside native, that imports
// Wails APIs (ADR A10). Everything here is a thin adapter.
package wails

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anofac/markdownia/internal/native"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Events adapts Wails event emission to native.Events.
type Events struct{ app *application.App }

// NewEvents constructs the Wails event emitter.
func NewEvents(app *application.App) *Events { return &Events{app: app} }

// Emit sends an event to the frontend.
func (e *Events) Emit(name string, data any) {
	e.app.Event.Emit(name, data)
}

// Dialog adapts Wails dialogs to native.Dialog. It needs a window to attach
// to; nil falls back to app-default dialogs.
type Dialog struct {
	app    *application.App
	window application.Window
}

// NewDialog constructs the Wails dialog adapter.
func NewDialog(app *application.App, window application.Window) *Dialog {
	return &Dialog{app: app, window: window}
}

// PickFolder opens a native folder picker. Cancellation is typed: ok=false.
func (d *Dialog) PickFolder(ctx context.Context) (string, bool, error) {
	builder := d.app.Dialog.OpenFile()
	builder.CanChooseDirectories(true)
	builder.CanChooseFiles(false)
	if d.window != nil {
		builder.AttachToWindow(d.window)
	}
	path, err := builder.PromptForSingleSelection()
	if err != nil {
		return "", false, err
	}
	if path == "" {
		return "", false, nil // cancelled
	}
	return path, true, nil
}

// PickZipFile opens a file picker filtered to .zip.
func (d *Dialog) PickZipFile(ctx context.Context) (string, bool, error) {
	builder := d.app.Dialog.OpenFile()
	builder.CanChooseFiles(true)
	builder.CanChooseDirectories(false)
	builder.AddFilter("ZIP archives", "*.zip")
	if d.window != nil {
		builder.AttachToWindow(d.window)
	}
	path, err := builder.PromptForSingleSelection()
	if err != nil {
		return "", false, err
	}
	if path == "" {
		return "", false, nil
	}
	return path, true, nil
}

// PickSaveLocation opens a native save dialog.
func (d *Dialog) PickSaveLocation(ctx context.Context, defaultName string) (string, bool, error) {
	builder := d.app.Dialog.SaveFile()
	builder.SetFilename(defaultName)
	if d.window != nil {
		builder.AttachToWindow(d.window)
	}
	path, err := builder.PromptForSingleSelection()
	if err != nil {
		return "", false, err
	}
	if path == "" {
		return "", false, nil
	}
	return path, true, nil
}

// Window adapts Wails window state to native.Window.
type Window struct {
	app    *application.App
	window *application.WebviewWindow
}

// NewWindow constructs the Wails window adapter.
func NewWindow(app *application.App, window *application.WebviewWindow) *Window {
	return &Window{app: app, window: window}
}

// GetState returns the current window geometry.
func (w *Window) GetState() native.WindowState {
	if w.window == nil {
		return native.WindowState{Width: 1280, Height: 800}
	}
	x, y := w.window.Position()
	b := w.window.Bounds()
	return native.WindowState{
		Width:    b.Width,
		Height:   b.Height,
		X:        x,
		Y:        y,
		Maximized: w.window.IsMaximised(),
	}
}

// SetState restores window geometry.
func (w *Window) SetState(s native.WindowState) {
	if w.window == nil {
		return
	}
	w.window.SetPosition(s.X, s.Y)
	w.window.SetSize(s.Width, s.Height)
	if s.Maximized {
		w.window.Maximise()
	}
}

// ClampToDisplays ensures saved geometry lands on a currently-connected
// display. A window saved on a now-disconnected monitor would otherwise open
// off-screen and read as a crash.
func (w *Window) ClampToDisplays(s native.WindowState) native.WindowState {
	if s.Width < 900 || s.Height < 600 {
		s.Width, s.Height = 1280, 800
	}
	// If no screen contains the window's center point, re-center it.
	screen := w.app.Screen.ScreenNearestDipRect(application.Rect{
		X: s.X, Y: s.Y, Width: s.Width, Height: s.Height,
	})
	b := screen.Bounds
	cx, cy := s.X+s.Width/2, s.Y+s.Height/2
	ok := cx >= b.X && cx <= b.X+b.Width && cy >= b.Y && cy <= b.Y+b.Height
	if !ok {
		s.X, s.Y = 0, 0 // OS places the window on a visible display
	}
	return s
}

// RevealInFileManager reveals a path in the OS file manager.
func (w *Window) RevealInFileManager(path string) {
	_ = w.app.Browser.OpenFile(path)
}

// Notify is a no-op in this Wails version (no notification API yet). The plan
// gates notifications to long index/clone completion only; wired here so the
// interface contract holds and behavior can be added upstream.
func (w *Window) Notify(title, body string) {}

// Browser adapts OS browser opening to native.Browser.
type Browser struct{ app *application.App }

// NewBrowser constructs the Wails browser adapter.
func NewBrowser(app *application.App) *Browser { return &Browser{app: app} }

// OpenExternal hands a URL to the OS browser.
func (b *Browser) OpenExternal(url string) error {
	return b.app.Browser.OpenURL(url)
}

// Settings adapts the SQLite settings repo to native.SettingsRepo. It is
// constructed in the app layer; here we provide the Wails-free wiring type.
type Settings struct {
	GetFn func(ctx context.Context, key string) (json.RawMessage, bool, error)
	SetFn func(ctx context.Context, key string, value json.RawMessage) error
}

// Get reads a setting.
func (s *Settings) Get(ctx context.Context, key string) (json.RawMessage, bool, error) {
	if s.GetFn == nil {
		return nil, false, fmt.Errorf("settings get not wired")
	}
	return s.GetFn(ctx, key)
}

// Set writes a setting.
func (s *Settings) Set(ctx context.Context, key string, value json.RawMessage) error {
	if s.SetFn == nil {
		return fmt.Errorf("settings set not wired")
	}
	return s.SetFn(ctx, key, value)
}
