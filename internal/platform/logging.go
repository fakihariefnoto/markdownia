package platform

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	logMaxSize  = 5 << 20 // 5 MB per file
	logMaxFiles = 3
)

// Logger returns a structured logger writing to a rotating local file in the
// app-data directory. Nothing is ever transmitted; the path is retrievable
// for the Help → Reveal Log File action.
func Logger(appDir string, level slog.Level) (*slog.Logger, *os.File, error) {
	path := filepath.Join(appDir, "markdownia.log")
	// #nosec G304 -- path is the fixed log file under the app-data dir.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("logging: open %s: %w", path, err)
	}

	opts := &slog.HandlerOptions{Level: level}
	logger := slog.New(slog.NewTextHandler(&rotatingWriter{file: f}, opts))
	return logger, f, nil
}

// LogPath returns the path of the rotating log file in the app-data dir.
func LogPath(appDir string) string {
	return filepath.Join(appDir, "markdownia.log")
}

// rotatingWriter truncates the file when it exceeds logMaxSize, keeping the
// newest logMaxFiles generations via simple rename rotation.
type rotatingWriter struct {
	file *os.File
	size int64
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	if w.size+int64(len(p)) > logMaxSize {
		_ = w.rotate()
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingWriter) rotate() error {
	path := w.file.Name()
	_ = w.file.Close()
	for i := logMaxFiles - 1; i > 0; i-- {
		older := fmt.Sprintf("%s.%d", path, i-1)
		newer := fmt.Sprintf("%s.%d", path, i)
		if _, err := os.Stat(older); err == nil {
			_ = os.Rename(older, newer)
		}
	}
	// #nosec G304 -- path is the fixed log file under the app-data dir.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	w.file = f
	w.size = 0
	return nil
}

var _ io.Writer = (*rotatingWriter)(nil)
