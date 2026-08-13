// Package app holds the application bootstrap: configuration derivation,
// dependency construction, and the Wails entrypoint ordering.
package app

import (
	"fmt"
	"path/filepath"

	"github.com/anofac/markdownia/internal/platform"
)

// Config carries every filesystem path the app needs. It is derived entirely
// from the platform app-data directory; no other package hardcodes a path.
type Config struct {
	AppDir       string
	DBPath       string
	LogPath      string
	ExtractedZip string
}

// Resolve computes Config from the OS app-data directory.
func Resolve() (Config, error) {
	dir, err := platform.AppDir()
	if err != nil {
		return Config{}, err
	}
	return Config{
		AppDir:       dir,
		DBPath:       filepath.Join(dir, "markdownia.db"),
		LogPath:      platform.LogPath(dir),
		ExtractedZip: filepath.Join(dir, "extracted"),
	}, nil
}

// Validate reports any configuration problem that would prevent startup.
func (c Config) Validate() error {
	if c.DBPath == "" {
		return fmt.Errorf("config: empty database path")
	}
	return nil
}
