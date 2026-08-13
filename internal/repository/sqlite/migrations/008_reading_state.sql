-- 008_reading_state.sql
-- One resume row per reading context; context_id = 0 is the library sentinel.
CREATE TABLE reading_state (
    context_type TEXT NOT NULL CHECK (context_type IN ('library','source','collection')),
    context_id   INTEGER NOT NULL,
    document_id  INTEGER REFERENCES documents(id) ON DELETE SET NULL,
    scroll_pct   REAL NOT NULL DEFAULT 0,
    updated_at   TEXT NOT NULL,
    PRIMARY KEY (context_type, context_id)
);
