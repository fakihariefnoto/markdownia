package flows

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Regression: replacing a folder's contents in place (delete old, add new
// files) then refreshing the source yields the new document set.
func TestRefreshReplacedFolder(t *testing.T) {
	e := newEnv(t)
	root := t.TempDir()

	write := func(rel, content string) {
		t.Helper()
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("old.md", "# Old\n\ncontent\n")
	id, err := e.sourceSvc.ImportFolder(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	e.waitIndexed(t, id)

	// Replace: remove old file, add two new files.
	if err := os.Remove(filepath.Join(root, "old.md")); err != nil {
		t.Fatal(err)
	}
	write("new1.md", "# New1\n\nhello\n")
	write("new2.md", "# New2\n\nworld\n")

	if err := e.sourceSvc.RefreshSource(context.Background(), id); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	e.waitIndexed(t, id)

	count, err := e.docRepo.CountBySource(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("count after replace = %d, want 2", count)
	}
}
