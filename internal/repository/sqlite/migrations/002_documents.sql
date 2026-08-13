-- 002_documents.sql
-- Documents: one markdown file, carrying the caches that make the hot path fast.
CREATE TABLE documents (
    id               INTEGER PRIMARY KEY,
    source_id        INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    rel_path         TEXT NOT NULL,
    title            TEXT NOT NULL,
    title_source     TEXT NOT NULL CHECK (title_source IN ('frontmatter','h1','filename')),
    file_hash        TEXT NOT NULL,
    file_mtime       INTEGER NOT NULL,
    file_size        INTEGER NOT NULL DEFAULT 0,
    rendered_html    TEXT NOT NULL,
    plain_text       TEXT NOT NULL DEFAULT '',
    code_text        TEXT NOT NULL DEFAULT '',
    outline_json     TEXT NOT NULL DEFAULT '[]',
    frontmatter_json TEXT,
    renderer_version INTEGER NOT NULL DEFAULT 0,
    word_count       INTEGER NOT NULL DEFAULT 0,
    has_mermaid      INTEGER NOT NULL DEFAULT 0,
    indexed_at       TEXT NOT NULL,
    UNIQUE (source_id, rel_path)
);

CREATE INDEX idx_documents_source ON documents(source_id);
CREATE INDEX idx_documents_title ON documents(title COLLATE NOCASE);
