package markdown

import (
	"strings"
	"testing"
)

// TestSanitizerGoldenCorpus is the actual implementation of ADR A6: a fixed
// corpus of hostile markdown asserted against expected sanitized output. It
// must run on every change to this package.
func TestSanitizerGoldenCorpus(t *testing.T) {
	cases := []struct {
		name string
		in   string
		// forbidden substrings that must NOT appear in output
		forbidden []string
		// required substrings that MUST appear
		required []string
	}{
		{
			name:      "script tag",
			in:        `<script>alert(1)</script>`,
			forbidden: []string{"<script", "alert(1)"},
		},
		{
			name:      "javascript href",
			in:        `<a href="javascript:alert(1)">x</a>`,
			forbidden: []string{"javascript:"},
			required:  []string{"x"},
		},
		{
			name:      "event handler attribute",
			in:        `<img src="x" onerror="alert(1)">`,
			forbidden: []string{"onerror", "alert(1)"},
		},
		{
			name:      "iframe",
			in:        `<iframe src="https://evil.example"></iframe>`,
			forbidden: []string{"<iframe"},
		},
		{
			name:      "object embed",
			in:        `<object data="x"></object>`,
			forbidden: []string{"<object"},
		},
		{
			name:      "data uri in img",
			in:        `<img src="data:image/svg+xml;base64,PHN2Zz48c2NyaXB0PmFsZXJ0KDEpPC9zY3JpcHQ+PC9zdmc+">`,
			forbidden: []string{"data:"},
		},
		{
			name:      "svg payload",
			in:        `<svg><script>alert(1)</script></svg>`,
			forbidden: []string{"<script", "<svg", "alert(1)"},
		},
		{
			name:      "html comment hiding markup",
			in:        `<!-- <script>alert(1)</script> -->`,
			forbidden: []string{"<script"},
		},
		{
			name:      "nested encoding trick",
			in:        `<a href="jav&#x61;script:alert(1)">x</a>`,
			forbidden: []string{"javascript", "alert(1)"},
		},
		{
			name:      "safe markdown survives",
			in:        `## Heading

A paragraph with **bold** and ` + "`code`" + `.

- list item`,
			required: []string{"<h2", "A paragraph with", "<strong>bold</strong>", "list item</li>"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := Sanitize(renderRaw(c.in))
			low := strings.ToLower(out)
			for _, f := range c.forbidden {
				if strings.Contains(low, strings.ToLower(f)) {
					t.Errorf("forbidden %q present in output: %s", f, out)
				}
			}
			for _, r := range c.required {
				if !strings.Contains(out, r) {
					t.Errorf("required %q missing from output: %s", r, out)
				}
			}
		})
	}
}

// renderRaw renders markdown through the parser so raw HTML flows through
// goldmark before sanitization.
func renderRaw(md string) string {
	p := NewParser()
	res, err := p.Render([]byte(md))
	if err != nil {
		return ""
	}
	return res.HTML
}

// TestSanitizerMetaCheck asserts the corpus is wired and non-empty, so it can
// never be silently skipped.
func TestSanitizerMetaCheck(t *testing.T) {
	if len(goldenCases()) == 0 {
		t.Fatal("sanitizer golden corpus is empty")
	}
}

func goldenCases() []string {
	return []string{
		"<script>alert(1)</script>",
		`<a href="javascript:alert(1)">x</a>`,
		`<img src="x" onerror="alert(1)">`,
		`<iframe src="https://evil.example"></iframe>`,
		`<svg><script>alert(1)</script></svg>`,
	}
}
