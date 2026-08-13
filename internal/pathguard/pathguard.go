// Package pathguard implements path containment — a security control used from
// three places: the indexer walker, zip extraction, and relative asset
// resolution. One implementation, one test suite, one audit surface.
package pathguard

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrEscapesRoot is returned when a path would escape the allowed root.
var ErrEscapesRoot = fmt.Errorf("path escapes root")

// IsContained reports whether candidate resolves to a path inside root, after
// resolving symlinks on both sides. It rejects: traversal components, absolute
// paths outside root, symlinks pointing outside root, and Windows path
// separators on non-Windows.
func IsContained(root, candidate string) bool {
	return Join(root, candidate) != ""
}

// Join resolves candidate against root and returns the contained absolute path,
// or "" when the candidate escapes root. An empty candidate resolves to root
// itself. The returned path is safe to open.
func Join(root, candidate string) string {
	if root == "" {
		return ""
	}

	// Windows separators on non-Windows are a traversal attempt, not a typo.
	if runtime.GOOS != "windows" && strings.Contains(candidate, "\\") {
		return ""
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ""
	}

	// A candidate that is itself absolute must still be inside root.
	if filepath.IsAbs(candidate) {
		return ""
	}

	clean := filepath.Clean(filepath.Join(absRoot, candidate))
	if !pathWithin(clean, absRoot) {
		return ""
	}

	// Resolve symlinks. The root may not exist yet (zip extraction creates it
	// during the operation), so fall back to the lexical check when it does not.
	realRoot, err1 := filepath.EvalSymlinks(absRoot)
	realCand, err2 := filepath.EvalSymlinks(clean)
	if err1 == nil && err2 == nil {
		if !pathWithin(realCand, realRoot) {
			return ""
		}
	} else if err2 == nil {
		// root did not resolve but the candidate did: a symlink escaped.
		if !pathWithin(realCand, absRoot) {
			return ""
		}
	}

	return clean
}

// pathWithin reports whether p is equal to root or inside it.
func pathWithin(p, root string) bool {
	if p == root {
		return true
	}
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// EnsureDirectory creates dir after verifying its parent chain is contained in
// root. Used by zip extraction so a malicious entry cannot create directories
// above the extraction root.
func EnsureDirectory(root, dir string) (string, error) {
	joined := Join(root, dir)
	if joined == "" {
		return "", ErrEscapesRoot
	}
	if err := os.MkdirAll(joined, 0o755); err != nil {
		return "", err
	}
	return joined, nil
}
