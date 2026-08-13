-- 009_open_tabs.sql
CREATE TABLE open_tabs (
    id          INTEGER PRIMARY KEY,
    context_type TEXT NOT NULL CHECK (context_type IN ('library','source','collection')),
    context_id  INTEGER NOT NULL,
    document_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    pane        INTEGER NOT NULL DEFAULT 0,
    position    INTEGER NOT NULL DEFAULT 0,
    is_active   INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_open_tabs_context ON open_tabs(context_type, context_id, pane, position);

-- Enforce one active tab per pane in the database, not only in application code.
CREATE UNIQUE INDEX idx_open_tabs_active
    ON open_tabs(context_type, context_id, pane)
    WHERE is_active = 1;
