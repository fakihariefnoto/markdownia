-- 005_collection_documents.sql
CREATE TABLE collection_documents (
    collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    document_id   INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    sort_order    INTEGER NOT NULL DEFAULT 0,
    added_at      TEXT NOT NULL,
    PRIMARY KEY (collection_id, document_id)
);

CREATE INDEX idx_collection_documents_document ON collection_documents(document_id);
