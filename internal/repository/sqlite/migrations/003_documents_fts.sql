-- 003_documents_fts.sql
-- FTS5 external-content table over documents. No triggers: the indexer is the
-- only writer and updates these rows explicitly in the same transaction.
CREATE VIRTUAL TABLE documents_fts USING fts5(
    title,
    headings,
    body,
    code,
    content='documents',
    content_rowid='id'
);
