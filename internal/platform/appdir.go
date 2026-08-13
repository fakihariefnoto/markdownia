// Package platform provides per-OS paths and logging for the app data
// directory, derived from the bundle identifier com.markdownia.app.
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// AppID is the bundle identifier used for the macOS bundle ID, the Windows
// application ID, and the OS app-data directory name. It must be identical
// across all three platforms.
const AppID = "com.markdownia.app"

// AppDir resolves the OS app-data directory for the application, creating it
// if missing.
//
// macOS:   ~/Library/Application Support/com.markdownia.app
// Windows: %APPDATA%\com.markdownia.app
// Linux:   $XDG_DATA_HOME/com.markdownia.app (fallback ~/.local/share)
func AppDir() (string, error) {
	var base string
	switch runtime.GOOS {
	case "darwin":
		base = os.Getenv("HOME")
		if base == "" {
			return "", fmt.Errorf("appdir: HOME is not set")
		}
		base = filepath.Join(base, "Library", "Application Support")
	case "windows":
		base = os.Getenv("APPDATA")
		if base == "" {
			return "", fmt.Errorf("appdir: APPDATA is not set")
		}
	case "linux":
		base = os.Getenv("XDG_DATA_HOME")
		if base == "" {
			home := os.Getenv("HOME")
			if home == "" {
				return "", fmt.Errorf("appdir: HOME is not set")
			}
			base = filepath.Join(home, ".local", "share")
		}
	default:
		return "", fmt.Errorf("appdir: unsupported OS %q", runtime.GOOS)
	}

	dir := filepath.Join(base, AppID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("appdir: create %s: %w", dir, err)
	}
	return dir, nil
}
