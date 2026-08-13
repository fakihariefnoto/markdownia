-- 001_sources.sql
-- Sources: an imported folder, git repo, or zip (the physical browsing axis).
CREATE TABLE sources (
    id              INTEGER PRIMARY KEY,
    kind            TEXT NOT NULL CHECK (kind IN ('folder', 'git', 'zip')),
    name            TEXT NOT NULL,
    root_path       TEXT NOT NULL,
    origin_url      TEXT,
    git_branch      TEXT,
    git_commit      TEXT,
    is_managed      INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','cloning','extracting','indexing','ready','unavailable','error')),
    error_message   TEXT,
    document_count  INTEGER NOT NULL DEFAULT 0,
    ignore_globs    TEXT,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    indexed_at      TEXT
);
