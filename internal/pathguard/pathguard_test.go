package pathguard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJoinRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	bad := []string{
		"../etc/passwd",
		"../../etc/passwd",
		"a/../../etc/passwd",
		"..",
		"../",
	}
	for _, b := range bad {
		if Join(root, b) != "" {
			t.Errorf("traversal %q not rejected", b)
		}
	}
}

func TestJoinRejectsAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	if Join(root, "/etc/passwd") != "" {
		t.Error("absolute path not rejected")
	}
	if Join(root, filepath.Join(root, "outside.txt")) != "" {
		t.Error("absolute path inside root should also be rejected (candidates must be relative)")
	}
}

func TestJoinRejectsWindowsSeparators(t *testing.T) {
	root := t.TempDir()
	// A Windows-style traversal must be rejected on non-Windows hosts.
	if os.Getenv("GOOS") != "windows" {
		if Join(root, `..\..\etc\passwd`) != "" {
			t.Error("windows separator traversal not rejected")
		}
	}
}

func TestJoinRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if Join(root, "link/secret.txt") != "" {
		t.Error("symlink escape not rejected")
	}
}

func TestJoinAllowsContained(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "docs", "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got := Join(root, "docs/nested/file.md")
	if got == "" {
		t.Fatal("contained path rejected")
	}
	want := filepath.Join(root, "docs", "nested", "file.md")
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestIsContained(t *testing.T) {
	root := t.TempDir()
	if !IsContained(root, "a/b.md") {
		t.Error("contained reported as escaping")
	}
	if IsContained(root, "../x") {
		t.Error("escaping reported as contained")
	}
}

func TestEnsureDirectoryRejectsEscape(t *testing.T) {
	root := t.TempDir()
	if _, err := EnsureDirectory(root, "../evil"); err == nil {
		t.Error("EnsureDirectory accepted an escaping path")
	}
	if _, err := EnsureDirectory(root, "good/sub"); err != nil {
		t.Errorf("EnsureDirectory rejected a contained path: %v", err)
	}
}
