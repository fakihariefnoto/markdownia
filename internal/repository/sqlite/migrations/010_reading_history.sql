-- 010_reading_history.sql
CREATE TABLE reading_history (
    document_id        INTEGER PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
    last_opened_at     TEXT NOT NULL,
    open_count         INTEGER NOT NULL DEFAULT 1,
    furthest_scroll_pct REAL NOT NULL DEFAULT 0
);

CREATE INDEX idx_reading_history_opened ON reading_history(last_opened_at DESC);

-- 011_settings.sql
-- Flat JSON key/value store; a new preference is an insert, not a migration.
CREATE TABLE settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
