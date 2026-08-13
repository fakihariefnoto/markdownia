package library

import "testing"

func TestBuildTreeDepthFirst(t *testing.T) {
	docs := []TreeNode{
		{ID: 4, RelPath: "a/b/deep.md", Title: "deep", Depth: 2},
		{ID: 3, RelPath: "a/mid.md", Title: "mid", Depth: 1},
		{ID: 1, RelPath: "top.md", Title: "top", Depth: 0},
		{ID: 2, RelPath: "a/b/c/x.md", Title: "x", Depth: 3},
		{ID: 5, RelPath: "z.md", Title: "z", Depth: 0},
	}
	tree := buildTree(docs)

	// Expect depth-first pre-order: folders interleaved with their children.
	got := make([]string, 0, len(tree))
	for _, n := range tree {
		got = append(got, n.RelPath)
	}
	want := []string{"a", "a/b", "a/b/c", "a/b/c/x.md", "a/b/deep.md", "a/mid.md", "top.md", "z.md"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d\n got: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pos %d = %q, want %q\n got: %v", i, got[i], want[i], got)
		}
	}
}

func TestBuildTreeEmpty(t *testing.T) {
	if got := buildTree(nil); len(got) != 0 {
		t.Fatalf("expected empty tree, got %v", got)
	}
}
