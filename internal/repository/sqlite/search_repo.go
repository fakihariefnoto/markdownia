package sqlite

import (
	"context"
	"fmt"
	"strings"

	"github.com/anofac/markdownia/internal/domain"
)

// SearchRepository is the only place FTS5 MATCH syntax is constructed. No
// other package may build a MATCH string.
type SearchRepository struct{ db *DB }

// NewSearchRepository constructs the SQLite search repository.
func NewSearchRepository(db *DB) *SearchRepository { return &SearchRepository{db: db} }

// SearchResult is one ranked FTS hit joined with its document and source.
type SearchResult struct {
	DocumentID int64
	Title      string
	RelPath    string
	SourceName string
	Snippet    string
	Rank       float64
}

// Query is a sanitized search request.
type Query struct {
	Text        string
	Scope       domain.ContextType
	ScopeID     int64
	IncludeCode bool
	Limit       int
	Offset      int
}

// sanitizeQuery converts user text into valid FTS5 MATCH syntax. Each word is
// quoted and joined with AND, so operators and stray punctuation cannot
// produce a syntax error or be injected.
func sanitizeQuery(q string) string {
	fields := strings.FieldsFunc(q, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n'
	})
	quoted := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.Trim(f, `"'*+-:(){}[]`)
		if f == "" {
			continue
		}
		quoted = append(quoted, `"`+strings.ReplaceAll(f, `"`, `""`) + `"`)
	}
	if len(quoted) == 0 {
		return ""
	}
	return strings.Join(quoted, " AND ")
}

// Search runs a full-text query across the indexed columns.
func (r *SearchRepository) Search(ctx context.Context, q Query) ([]SearchResult, error) {
	match := sanitizeQuery(q.Text)
	if match == "" {
		return nil, nil
	}

	cols := "{title headings body}"
	if q.IncludeCode {
		cols = "{title headings body code}"
	}
	// Column restriction and the query are one FTS5 MATCH expression.
	ftsMatch := cols + " : " + match

	// modernc's FTS5 cannot run snippet()/highlight() against external-content
	// tables, so the plain text is selected and the snippet built Go-side from
	// the document row. This keeps the ERD's external-content design (no double
	// storage) while staying within the query budget.
	selectPart := `
		SELECT d.id, d.title, d.rel_path, s.name, d.plain_text,
		       bm25(documents_fts, 7.0, 3.0, 1.0, 0.3) AS rank
		FROM documents_fts
		JOIN documents d ON d.id = documents_fts.rowid
		JOIN sources s ON s.id = d.source_id
		WHERE documents_fts MATCH ?`

	args := []any{ftsMatch}

	switch q.Scope {
	case domain.ContextSource:
		selectPart += ` AND d.source_id = ?`
		args = append(args, q.ScopeID)
	case domain.ContextCollection:
		selectPart += ` AND d.id IN (SELECT document_id FROM collection_documents WHERE collection_id=?)`
		args = append(args, q.ScopeID)
	}

	selectPart += ` ORDER BY rank LIMIT ? OFFSET ?`
	args = append(args, q.Limit, q.Offset)

	rows, err := r.db.Reader.QueryContext(ctx, selectPart, args...)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	terms := matchTerms(q.Text)

	var out []SearchResult
	for rows.Next() {
		var res SearchResult
		var plain string
		if err := rows.Scan(&res.DocumentID, &res.Title, &res.RelPath, &res.SourceName,
			&plain, &res.Rank); err != nil {
			return nil, err
		}
		res.Snippet = buildSnippet(plain, terms)
		out = append(out, res)
	}
	return out, rows.Err()
}

// matchTerms returns the lowercase terms that will appear in the snippet.
func matchTerms(q string) []string {
	var out []string
	for _, f := range strings.Fields(q) {
		f = strings.Trim(f, `"'*+-:(){}[]`)
		if f != "" {
			out = append(out, strings.ToLower(f))
		}
	}
	return out
}

// buildSnippet produces a <mark>-highlighted excerpt around the first matched
// term. The emphasis range always matches what the index matched because the
// same sanitized terms drive both.
func buildSnippet(plain string, terms []string) string {
	lower := strings.ToLower(plain)
	if len(terms) == 0 || plain == "" {
		return truncate(plain, 24)
	}

	best := -1
	for _, t := range terms {
		if i := strings.Index(lower, t); i >= 0 && (best < 0 || i < best) {
			best = i
		}
	}
	if best < 0 {
		return truncate(plain, 24)
	}

	// Window around the match: roughly 120 chars with the term centered.
	const window = 60
	start := best - window
	if start < 0 {
		start = 0
	}
	end := best + window
	if end > len(plain) {
		end = len(plain)
	}

	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = "… "
	}
	if end < len(plain) {
		suffix = " …"
	}

	body := plain[start:end]
	for _, t := range terms {
		if t == "" {
			continue
		}
		body = highlightAll(body, t)
	}
	return prefix + body + suffix
}

// highlightAll wraps every case-insensitive occurrence of term in <mark>.
func highlightAll(s, term string) string {
	var b strings.Builder
	lower := strings.ToLower(s)
	pos := 0
	for {
		i := strings.Index(lower[pos:], term)
		if i < 0 {
			b.WriteString(s[pos:])
			break
		}
		i += pos
		b.WriteString(s[pos:i])
		b.WriteString("<mark>")
		b.WriteString(s[i : i+len(term)])
		b.WriteString("</mark>")
		pos = i + len(term)
	}
	return b.String()
}

func truncate(s string, maxWords int) string {
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return s
	}
	return strings.Join(words[:maxWords], " ") + " …"
}
