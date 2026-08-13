package markdown

// RendererVersion is cache-invalidation axis 2 (architecture.md §5). It is
// bumped whenever goldmark config, chroma, the sanitizer policy, block
// hashing, or outline extraction changes behavior. A mismatch forces a
// re-render on next open regardless of file_hash.
//
// RULE: bump this constant whenever any behavior in this package changes.
const RendererVersion = 1
