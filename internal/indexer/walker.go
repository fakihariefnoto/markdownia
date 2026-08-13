// Package indexer walks a source, decides what changed (in the exact order
// that protects highlights), parses changed documents, and sweeps orphaned
// highlights inside the index transaction.
package indexer

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/anofac/markdownia/internal/domain"
	"github.com/anofac/markdownia/internal/pathguard"
)

// walker discovers markdown files under a source root, honoring ignore globs
// and skipping obvious noise, path-guarding every path before it is opened.
type walker struct {
	root     string
	ignores  []string
	markdown []string
}

// walk collects every markdown file under root, relative to root.
func walk(root string, ignores []string) ([]string, error) {
	w := &walker{root: root, ignores: ignores}
	if err := w.recurse(""); err != nil {
		return nil, err
	}
	return w.markdown, nil
}

// WalkForTest exposes the walker to integration tests.
func WalkForTest(root string, ignores []string) ([]string, error) {
	return walk(root, ignores)
}

func (w *walker) recurse(rel string) error {
	dir := pathguard.Join(w.root, rel)
	if dir == "" {
		return domain.ErrPathEscapesRoot
	}
	entries, err := readDir(dir)
	if err != nil {
		return fmt.Errorf("indexer walk %s: %w", dir, err)
	}
	for _, e := range entries {
		name := e.Name()
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		if e.IsDir() {
			if isNoiseDir(name) || matchesAny(childRel, w.ignores) {
				continue
			}
			if err := w.recurse(childRel); err != nil {
				return err
			}
			continue
		}
		if !e.IsDir() {
			ext := strings.ToLower(filepath.Ext(name))
			if domain.IsMarkdownExtension(ext) && !matchesAny(childRel, w.ignores) {
				w.markdown = append(w.markdown, childRel)
			}
		}
	}
	return nil
}

// readDir is an indirection for testability.
var readDir = func(dir string) ([]DirEntry, error) {
	return osReadDir(dir)
}

// isNoiseDir reports directory names never worth indexing.
func isNoiseDir(name string) bool {
	switch name {
	case ".git", "node_modules", ".hg", ".svn", "vendor", ".next", "dist":
		return true
	}
	return false
}

// matchesAny applies gitignore-style suffix globs to a rel path.
func matchesAny(rel string, globs []string) bool {
	for _, g := range globs {
		if g == "" {
			continue
		}
		if matchGlob(rel, g) {
			return true
		}
	}
	return false
}

// matchGlob supports simple gitignore-style patterns: `**/node_modules`,
// `*.log`, `CHANGELOG.md`, directory names matching anywhere.
func matchGlob(rel, pattern string) bool {
	p := strings.TrimSpace(pattern)
	p = strings.Trim(p, "/")
	if p == "" {
		return false
	}
	// A bare name matches the last path element (gitignore behavior for
	// patterns without a slash).
	if !strings.Contains(p, "/") && !strings.Contains(p, "*") {
		last := rel
		if i := strings.LastIndex(rel, "/"); i >= 0 {
			last = rel[i+1:]
		}
		return last == p
	}
	// Glob match over the full path.
	ok, _ := filepath.Match(p, rel)
	if ok {
		return true
	}
	// Directory-wide patterns like `**/node_modules/**`.
	if strings.HasPrefix(p, "**/") {
		ok, _ = filepath.Match(strings.TrimPrefix(p, "**/"), rel)
		if ok {
			return true
		}
		if strings.HasSuffix(rel, "/"+strings.TrimPrefix(p, "**/")) {
			return true
		}
	}
	if strings.HasSuffix(p, "/**") {
		return strings.HasPrefix(rel, strings.TrimSuffix(p, "/**"))
	}
	return false
}
