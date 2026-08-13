-- 006_highlights.sql
-- Highlights anchored to a containing block's hash (PRD decision D4).
CREATE TABLE highlights (
    id          INTEGER PRIMARY KEY,
    document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    block_hash  TEXT NOT NULL,
    block_index INTEGER NOT NULL,
    start_offset INTEGER NOT NULL,
    end_offset  INTEGER NOT NULL,
    quoted_text TEXT NOT NULL,
    color       TEXT NOT NULL CHECK (color IN ('yellow','green','blue','pink','orange')),
    note        TEXT,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

CREATE INDEX idx_highlights_document ON highlights(document_id);
CREATE INDEX idx_highlights_block ON highlights(block_hash);
