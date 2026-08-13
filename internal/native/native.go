// Package native is the ONLY package permitted to import Wails APIs (ADR A10).
// It handles dialogs, menus, window state, opening the OS browser, and event
// emission — keeping every other Go package testable headlessly.
package native

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"time"
)

// AppVersion is stamped at build time via ldflags from the git tag.
var AppVersion = "dev"

// Events is the interface the event emitter uses. The concrete implementation
// (Wails event emission) is injected at wiring time so this package stays
// testable.
type Events interface {
	Emit(name string, data any)
}

// WindowState is the persisted window geometry.
type WindowState struct {
	Width    int  `json:"width"`
	Height   int  `json:"height"`
	X        int  `json:"x"`
	Y        int  `json:"y"`
	Maximized bool `json:"maximized"`
}

// SettingsRepo is the narrow view of settings the native layer needs.
type SettingsRepo interface {
	Get(ctx context.Context, key string) (json.RawMessage, bool, error)
	Set(ctx context.Context, key string, value json.RawMessage) error
}

// Dialog shows native dialogs. Implemented per-OS at wiring time.
type Dialog interface {
	PickFolder(ctx context.Context) (string, bool, error)
	PickZipFile(ctx context.Context) (string, bool, error)
	PickSaveLocation(ctx context.Context, defaultName string) (string, bool, error)
}

// Window controls the OS window.
type Window interface {
	GetState() WindowState
	SetState(WindowState)
	ClampToDisplays(state WindowState) WindowState
	RevealInFileManager(path string)
	Notify(title, body string)
}

// Browser opens URLs and reveals paths in the OS.
type Browser interface {
	OpenExternal(url string) error
}

// Native bundles the native services for the binding layer.
type Native struct {
	events   Events
	settings SettingsRepo
	dialog   Dialog
	window   Window
	browser  Browser
	logger   *slog.Logger
	client   *http.Client
}

// New constructs the native layer.
func New(events Events, settings SettingsRepo, dialog Dialog, window Window, browser Browser, logger *slog.Logger) *Native {
	return &Native{
		events: events, settings: settings, dialog: dialog, window: window,
		browser: browser, logger: logger,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// PickFolder opens a native folder picker. Cancellation is a typed result.
func (n *Native) PickFolder(ctx context.Context) (string, bool, error) {
	return n.dialog.PickFolder(ctx)
}

// PickZipFile opens a native file picker filtered to .zip.
func (n *Native) PickZipFile(ctx context.Context) (string, bool, error) {
	return n.dialog.PickZipFile(ctx)
}

// PickSaveLocation opens a native save dialog.
func (n *Native) PickSaveLocation(ctx context.Context, defaultName string) (string, bool, error) {
	return n.dialog.PickSaveLocation(ctx, defaultName)
}

// OpenExternal hands a URL to the OS browser. The webview never navigates
// off-origin — a security control, since it holds the bindings bridge.
func (n *Native) OpenExternal(ctx context.Context, url string) error {
	if n.browser == nil {
		return fmt.Errorf("browser not wired")
	}
	return n.browser.OpenExternal(url)
}

// RevealInFileManager reveals a path in the OS file manager.
func (n *Native) RevealInFileManager(ctx context.Context, path string) {
	if n.window != nil {
		n.window.RevealInFileManager(path)
	}
}

// GetWindowState reads the current window geometry.
func (n *Native) GetWindowState(ctx context.Context) WindowState {
	if n.window == nil {
		return WindowState{}
	}
	return n.window.GetState()
}

// SaveWindowState persists geometry, clamped to the current display set so a
// window saved on a now-disconnected monitor still opens visibly.
func (n *Native) SaveWindowState(ctx context.Context, state WindowState) error {
	if n.settings == nil {
		return nil
	}
	if n.window != nil {
		state = n.window.ClampToDisplays(state)
	}
	b, _ := json.Marshal(state)
	return n.settings.Set(ctx, "window.state", b)
}

// Emit forwards an event to the frontend.
func (n *Native) Emit(name string, data any) {
	if n.events != nil {
		n.events.Emit(name, data)
	}
}

// CheckForUpdates queries GitHub Releases only when invoked and opens the
// release page in the OS browser on a newer version. The app never downloads
// or installs anything itself (PRD decision D7).
func (n *Native) CheckForUpdates(ctx context.Context, repo string) (UpdateResult, error) {
	if n.client == nil {
		n.client = &http.Client{Timeout: 10 * time.Second}
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return UpdateResult{State: "failed", Message: "Couldn't reach GitHub."}, nil
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := n.client.Do(req)
	if err != nil {
		return UpdateResult{State: "failed", Message: "Couldn't reach GitHub. You appear to be offline."}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
		Name    string `json:"name"`
	}
	if resp.StatusCode != http.StatusOK {
		return UpdateResult{State: "failed", Message: fmt.Sprintf("Update check failed (HTTP %d).", resp.StatusCode)}, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return UpdateResult{State: "failed", Message: "Update check returned an unreadable response."}, nil
	}

	if release.TagName != "" && release.TagName != AppVersion {
		// Newer or different version available: open its release page.
		page := release.HTMLURL
		if page == "" {
			page = fmt.Sprintf("https://github.com/%s/releases", repo)
		}
		if n.browser != nil {
			_ = n.browser.OpenExternal(page)
		}
		return UpdateResult{
			State:      "available",
			Message:    release.Name,
			Current:    AppVersion,
			Available:  release.TagName,
			ReleaseURL: page,
		}, nil
	}
	return UpdateResult{State: "up_to_date", Message: "You're up to date.", Current: AppVersion}, nil
}

// UpdateResult is the update-check dialog state.
type UpdateResult struct {
	State      string `json:"state"` // checking | up_to_date | available | failed
	Message    string `json:"message,omitempty"`
	Current    string `json:"current,omitempty"`
	Available  string `json:"available,omitempty"`
	ReleaseURL string `json:"releaseUrl,omitempty"`
}

// Platform returns the runtime platform for the frontend's KeyHint rendering.
func Platform() string { return runtime.GOOS }
