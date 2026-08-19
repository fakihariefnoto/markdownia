package flows

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Regression: RelocateSource re-indexes the new location immediately, so the
// library reflects the new folder's documents without a manual refresh.
func TestRelocateReindexes(t *testing.T) {
	e := newEnv(t)
	oldRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(oldRoot, "old.md"), []byte("# Old\n\ncontent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	id, err := e.sourceSvc.ImportFolder(context.Background(), oldRoot)
	if err != nil {
		t.Fatal(err)
	}
	e.waitIndexed(t, id)

	newRoot := t.TempDir()
	for rel, content := range map[string]string{
		"new1.md": "# New1\n\nhello\n",
		"new2.md": "# New2\n\nworld\n",
		"new3.md": "# New3\n\nfoo\n",
	} {
		if err := os.WriteFile(filepath.Join(newRoot, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := e.sourceSvc.RelocateSource(context.Background(), id, newRoot); err != nil {
		t.Fatalf("relocate: %v", err)
	}
	e.waitIndexed(t, id)

	// The import already populated the sink, so waitIndexed can return before
	// the relocate-triggered index finishes. Poll for the expected count.
	var count int64
	for i := 0; i < 200; i++ {
		c, err := e.docRepo.CountBySource(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if c == 3 {
			count = c
			break
		}
		<-sleepNow()
	}
	if count != 3 {
		t.Fatalf("count after relocate = %d, want 3", count)
	}
}
