-- 007_bookmarks.sql
CREATE TABLE bookmarks (
    id             INTEGER PRIMARY KEY,
    document_id    INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    heading_anchor TEXT,
    note           TEXT,
    created_at     TEXT NOT NULL
);

CREATE INDEX idx_bookmarks_document ON bookmarks(document_id);
