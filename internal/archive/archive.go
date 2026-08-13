// Package archive extracts zip archives into app-managed storage with
// zip-slip containment: every entry is pathguard-checked before it is written.
package archive

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/pathguard"
)

const maxZipSize = 1 << 30 // 1 GB hard cap on extracted content

// Extract unzips src into dest, guarding every entry against escaping the
// destination root. Progress (entries extracted, total) is reported via the
// callback when non-nil. A malicious archive fails the import rather than
// writing outside the root.
func Extract(ctx context.Context, src, dest string, onProgress func(current, total int)) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrZipCorrupt, err)
	}
	defer func() { _ = r.Close() }()

	// Two passes: count total entries for honest progress, then extract.
	total := len(r.File)

	var written int64
	for i, f := range r.File {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%w: %v", domain.ErrOperationCancelled, err)
		}

		// Containment: never allow a path to escape dest.
		target := pathguard.Join(dest, f.Name)
		if target == "" {
			return fmt.Errorf("%w: entry %q escapes archive root", domain.ErrPathEscapesRoot, f.Name)
		}

		info := f.FileInfo()
		if info.IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			if onProgress != nil {
				onProgress(i+1, total)
			}
			continue
		}

		// Size cap before writing. Cap the uint64 at the int64 bound before
		// converting so an oversized header cannot overflow the comparison.
		entrySize := f.UncompressedSize64
		if entrySize > uint64(maxZipSize) {
			return fmt.Errorf("%w: extracted size exceeds %d bytes", domain.ErrZipTooLarge, maxZipSize)
		}
		if written+int64(entrySize) > maxZipSize {
			return fmt.Errorf("%w: extracted size exceeds %d bytes", domain.ErrZipTooLarge, maxZipSize)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		// Symlink entries are refused outright — a zip can otherwise smuggle a
		// link that points outside the extraction root.
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%w: symlink entry %q", domain.ErrPathEscapesRoot, f.Name)
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		// #nosec G304 -- target was produced by pathguard.Join against the extraction root.
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			_ = rc.Close()
			return err
		}
		n, err := io.Copy(out, io.LimitReader(rc, maxZipSize-written))
		written += n
		cerr := out.Close()
		rcErr := rc.Close()
		if err != nil || cerr != nil {
			return fmt.Errorf("extract %s: %w", f.Name, errOr(cerr, err))
		}
		if rcErr != nil {
			return fmt.Errorf("extract %s: %w", f.Name, rcErr)
		}

		if onProgress != nil {
			onProgress(i+1, total)
		}
	}
	return nil
}

func errOr(a, b error) error {
	if a != nil {
		return a
	}
	return b
}
