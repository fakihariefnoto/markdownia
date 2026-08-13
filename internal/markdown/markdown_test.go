package markdown

import (
	"strings"
	"testing"
)

func newTestParser(t *testing.T) *Parser {
	t.Helper()
	return NewParser()
}

// fixture renders a document exercising GFM features and asserts every output
// of the pipeline: HTML, plain text, code text, outline, block hashes.
const fixtureDoc = `---
title: Fixture Doc
tags: [go, markdown]
---
# Title One

A paragraph with **bold**, *italic*, and ` + "`inline code`" + `.

> A blockquote with a second line.

- list item one
- list item two

1. ordered one
2. ordered two

| Name  | Value |
|-------|-------|
| alpha | 1     |
| beta  | 2     |

` + "```go" + `
func main() { fmt.Println("hi") }
` + "```" + `

` + "```mermaid" + `
graph TD; A-->B;
` + "```" + `

Footnote here[^1].

[^1]: The footnote text.
`

func TestRenderProducesAllOutputs(t *testing.T) {
	p := newTestParser(t)
	res, err := p.Render([]byte(fixtureDoc))
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// HTML: sanitized, has structure.
	if !strings.Contains(res.HTML, "<p") {
		t.Error("HTML missing paragraph")
	}
	if !strings.Contains(res.HTML, "<h1") {
		t.Error("HTML missing heading")
	}
	if !strings.Contains(res.HTML, "<table>") {
		t.Error("HTML missing table")
	}
	if strings.Contains(res.HTML, "<script") {
		t.Error("HTML contains script tag")
	}

	// Title resolution: frontmatter wins.
	if res.Title != "Fixture Doc" || res.TitleSource != "frontmatter" {
		t.Errorf("title = %q (%q), want Fixture Doc (frontmatter)", res.Title, res.TitleSource)
	}

	// Outline: one heading, with an anchor.
	if len(res.Outline) != 1 {
		t.Fatalf("outline length = %d, want 1", len(res.Outline))
	}
	if res.Outline[0].Text != "Title One" {
		t.Errorf("outline[0].Text = %q", res.Outline[0].Text)
	}
	if res.Outline[0].Anchor == "" {
		t.Error("outline[0].Anchor empty — auto heading id missing")
	}

	// Plain text contains prose, not code.
	if !strings.Contains(res.PlainText, "A paragraph with") {
		t.Error("plain text missing paragraph prose")
	}
	if strings.Contains(res.PlainText, "func main()") {
		t.Error("plain text contains code fence contents (should be excluded)")
	}

	// Code text contains the fence, not prose.
	if !strings.Contains(res.CodeText, "func main()") {
		t.Error("code text missing go fence")
	}

	// Mermaid detected.
	if !res.HasMermaid {
		t.Error("has_mermaid false, want true")
	}

	// Block hashes exist and are stable-looking.
	if len(res.Blocks) == 0 {
		t.Fatal("no blocks extracted")
	}
	for _, b := range res.Blocks {
		if len(b.Hash) != 64 {
			t.Errorf("block hash %q not sha256 hex", b.Hash)
		}
	}
}

func TestBlockHashStability(t *testing.T) {
	p := newTestParser(t)
	src := []byte("# Doc\n\nSame paragraph here twice.\n\nSame paragraph here twice.\n")
	r1, err := p.Render(src)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := p.Render(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(r1.Blocks) != len(r2.Blocks) {
		t.Fatalf("block count differs between runs: %d vs %d", len(r1.Blocks), len(r2.Blocks))
	}
	for i := range r1.Blocks {
		if r1.Blocks[i].Hash != r2.Blocks[i].Hash {
			t.Errorf("block %d hash unstable: %s vs %s", i, r1.Blocks[i].Hash, r2.Blocks[i].Hash)
		}
	}
	// Identical paragraphs share a hash but differ by index.
	if r1.Blocks[0].Hash == r1.Blocks[1].Hash {
		t.Error("heading and paragraph should not share a hash")
	}
	if r1.Blocks[1].Hash != r1.Blocks[2].Hash {
		t.Error("identical paragraphs should share a hash")
	}
	if r1.Blocks[1].Index == r1.Blocks[2].Index {
		t.Error("identical paragraphs should have distinct block indexes")
	}
}

func TestTitleResolutionFallbacks(t *testing.T) {
	p := newTestParser(t)

	// First H1.
	res, _ := p.Render([]byte("Some prose.\n\n# The H1 Title\n\nMore prose.\n"))
	if res.Title != "The H1 Title" || res.TitleSource != "h1" {
		t.Errorf("title = %q (%q), want H1 fallback", res.Title, res.TitleSource)
	}

	// Filename fallback is decided by the caller; here we assert title is set.
	res, _ = p.Render([]byte("Prose only, no heading.\n"))
	if res.Title != "Untitled" {
		t.Errorf("no-heading title = %q", res.Title)
	}
}

func TestSanitizerStripsHostileMarkup(t *testing.T) {
	p := newTestParser(t)
	hostile := `<script>alert(1)</script>
<a href="javascript:alert(1)">click</a>
<img src="x" onerror="alert(1)">

Safe paragraph.
`
	res, err := p.Render([]byte(hostile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.HTML, "<script") {
		t.Error("script survived sanitization")
	}
	if strings.Contains(res.HTML, "javascript:") {
		t.Error("javascript: href survived sanitization")
	}
	if strings.Contains(res.HTML, "onerror") {
		t.Error("event handler survived sanitization")
	}
	if !strings.Contains(res.HTML, "Safe paragraph") {
		t.Error("safe content lost")
	}
}
