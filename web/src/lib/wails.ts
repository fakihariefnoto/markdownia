// wails.ts — Wails binding layer. Uses window.wails.Call.ByName (the injected
// /wails/runtime.js global) — the same proven mechanism the original vanilla
// frontend used. Every backend call returns [data, error] with the error
// normalized to {code, content}.
//
// In a plain browser (no Wails runtime) these fall back to an in-memory mock
// so screens stay developable without the Go layer.

// Fully-qualified service method base for the bound services.
const PKG = "github.com/anofac/markdownia/internal/binding"

export type Err = { code: string; content: string } | null

// normalizeError mirrors the original bindings.js normalization.
function normalizeError(err: unknown): Err {
  if (err == null) return null
  if (typeof err === "string") {
    try {
      const parsed = JSON.parse(err)
      if (parsed && parsed.content) return { code: parsed.code || "error", content: parsed.content }
    } catch {
      /* not JSON */
    }
    return { code: "error", content: err }
  }
  const e = err as { content?: string; code?: string; message?: string }
  if (e.content) return { code: e.code || "error", content: e.content }
  if (e.message) return { code: "error", content: e.message }
  return { code: "error", content: "Something went wrong. See Help → Reveal Log File for details." }
}

// hasRuntime reports whether the Wails runtime is available.
function hasRuntime() {
  return typeof window !== "undefined" && !!(window as any).wails?.Call?.ByName
}

// call invokes a bound service method by FQN.
function call(service: string, method: string, ...args: any[]): Promise<any> {
  const fqn = `${PKG}.${service}.${method}`
  const runtime = (window as any).wails
  if (runtime?.Call?.ByName) {
    return runtime.Call.ByName(fqn, ...args)
  }
  return devCall(service, method, ...args)
}

// invoke resolves a backend call into [data, err].
async function invoke(service: string, method: string, ...args: any[]): Promise<[any, Err]> {
  const fqn = `${PKG}.${service}.${method}`
  try {
    const data = await call(service, method, ...args)
    if (service === "NativeService" && method === "PickFolder") console.log("[wails] PickFolder →", JSON.stringify(data))
    if (service === "SourceService" && method === "ImportFolder") console.log("[wails] ImportFolder →", JSON.stringify(data))
    return [data, null]
  } catch (err) {
    console.log(`[wails] ${method} error:`, err)
    return [null, normalizeError(err)]
  }
}

export const sources = {
  list: () => invoke("SourceService", "ListSources"),
  importFolder: (path: string) => invoke("SourceService", "ImportFolder", path),
  importGit: (url: string, branch: string) => invoke("SourceService", "ImportGit", url, branch),
  importZip: (path: string) => invoke("SourceService", "ImportZip", path),
  refresh: (id: number) => invoke("SourceService", "RefreshSource", id),
  relocate: (id: number, path: string) => invoke("SourceService", "RelocateSource", id, path),
  rename: (id: number, name: string) => invoke("SourceService", "RenameSource", id, name),
  del: (id: number) => invoke("SourceService", "DeleteSource", id),
  deletionPreview: (id: number) => invoke("SourceService", "SourceDeletionPreview", id),
  cancel: (id: number) => invoke("SourceService", "CancelSourceOperation", id),
  rebuildAll: () => invoke("SourceService", "RebuildAll"),
}

export const library = {
  tree: (sourceId: number) => invoke("LibraryService", "GetTree", sourceId),
  open: (docId: number, contextType: string, contextId: number) =>
    invoke("LibraryService", "OpenDocument", docId, contextType, contextId),
  meta: (docId: number) => invoke("LibraryService", "GetDocumentMeta", docId),
  resolveLink: (fromDocId: number, href: string) => invoke("LibraryService", "ResolveLink", fromDocId, href),
  asset: (docId: number, relPath: string) => invoke("LibraryService", "GetAsset", docId, relPath),
  recent: (limit: number) => invoke("LibraryService", "ListRecent", limit),
}

export const search = {
  run: (q: string, scopeType: string, scopeId: number, includeCode: boolean, offset: number) =>
    invoke("SearchService", "Search", q, { type: scopeType, id: scopeId }, includeCode, offset),
}

export const collections = {
  list: () => invoke("CollectionService", "ListCollections"),
  create: (name: string, description: string) =>
    invoke("CollectionService", "CreateCollection", name, description || ""),
  rename: (id: number, name: string) => invoke("CollectionService", "RenameCollection", id, name),
  del: (id: number) => invoke("CollectionService", "DeleteCollection", id),
  addDocuments: (id: number, docIds: number[]) => invoke("CollectionService", "AddDocuments", id, docIds),
  removeDocuments: (id: number, docIds: number[]) => invoke("CollectionService", "RemoveDocuments", id, docIds),
  reorder: (id: number, orderedIds: number[]) => invoke("CollectionService", "ReorderDocuments", id, orderedIds),
  listDocuments: (id: number) => invoke("CollectionService", "ListCollectionDocuments", id),
  forDocument: (docId: number) => invoke("CollectionService", "CollectionsForDocument", docId),
}

export const annotations = {
  addBookmark: (docId: number, headingAnchor: string, note: string) =>
    invoke("AnnotationService", "AddBookmark", docId, headingAnchor || "", note || ""),
  removeBookmark: (id: number) => invoke("AnnotationService", "RemoveBookmark", id),
  listBookmarks: () => invoke("AnnotationService", "ListBookmarks"),
  addHighlight: (docId: number, anchor: any, color: string, note: string) =>
    invoke("AnnotationService", "AddHighlight", docId, anchor, color, note || ""),
  updateHighlight: (id: number, color: string, note: string) =>
    invoke("AnnotationService", "UpdateHighlight", id, color || "", note || ""),
  removeHighlight: (id: number) => invoke("AnnotationService", "RemoveHighlight", id),
  listHighlights: (docId: number) => invoke("AnnotationService", "ListHighlights", docId),
  all: () => invoke("AnnotationService", "ListAllAnnotations"),
}

export const reading = {
  getState: (contextType: string, contextId: number) =>
    invoke("ReadingService", "GetReadingState", contextType, contextId),
  saveScroll: (contextType: string, contextId: number, docId: number, pct: number) =>
    invoke("ReadingService", "SaveScrollPosition", contextType, contextId, docId, pct),
  getTabs: (contextType: string, contextId: number) =>
    invoke("ReadingService", "GetOpenTabs", contextType, contextId),
  saveTabs: (contextType: string, contextId: number, tabs: any[]) =>
    invoke("ReadingService", "SaveOpenTabs", contextType, contextId, tabs),
  lastContext: () => invoke("ReadingService", "GetLastContext"),
}

export const settings = {
  getAll: () => invoke("SettingsService", "GetAll"),
  get: (key: string) => invoke("SettingsService", "Get", key),
  set: (key: string, value: string) => invoke("SettingsService", "Set", key, value),
  reset: (key: string) => invoke("SettingsService", "Reset", key),
}

export const exportApi = {
  prepare: (target: any) => invoke("ExportService", "PrepareExport", target),
  exportHTML: (target: any, destPath: string) => invoke("ExportService", "ExportHTML", target, destPath),
}

export const native = {
  pickFolder: () => invoke("NativeService", "PickFolder"),
  pickZip: () => invoke("NativeService", "PickZipFile"),
  pickSave: (name: string) => invoke("NativeService", "PickSaveLocation", name),
  openExternal: (url: string) => invoke("NativeService", "OpenExternal", url),
  reveal: (path: string) => invoke("NativeService", "RevealInFileManager", path),
  getWindowState: () => invoke("NativeService", "GetWindowState"),
  saveWindowState: (state: any) => invoke("NativeService", "SaveWindowState", state),
  checkForUpdates: (repo: string) => invoke("NativeService", "CheckForUpdates", repo),
}

/* ============================================================
   Dev fallback — in-memory mock so screens work in a plain browser.
   ============================================================ */
const devStore: Record<string, any> = {
  sources: [
    { id: 1, kind: "folder", name: "Notes", rootPath: "/Users/me/notes", status: "ready", documentCount: 24 },
    { id: 2, kind: "git", name: "docs-repo", rootPath: "/Users/me/docs-repo", originUrl: "https://github.com/x/docs", status: "ready", documentCount: 340, gitBranch: "main", gitCommit: "abc123" },
    { id: 3, kind: "zip", name: "archive-docs", rootPath: "/tmp/extracted/src-3", isManaged: true, status: "ready", documentCount: 12 },
  ],
  collections: [
    { id: 1, name: "Onboarding", description: "Things to read before the review", documentCount: 5 },
  ],
  settings: {
    "appearance.mode": '"system"',
    "appearance.accent": '"teal"',
    "reading.theme": '"paper"',
    "reading.font": '"sans"',
    "reading.size": "1.0",
    "reading.measure": '"72ch"',
  },
}

async function devCall(service: string, method: string, ...args: any[]): Promise<any> {
  await sleep(30)
  const key = `${service}.${method}`
  switch (key) {
    case "Settings.GetAll":
      return { ...devStore.settings }
    case "Settings.Get":
      return devStore.settings[args[0] as string] ?? null
    case "Settings.Set":
      devStore.settings[args[0] as string] = args[1]
      return null
    case "Source.ListSources":
      return devStore.sources
    case "Collection.ListCollections":
      return devStore.collections
    case "Source.SourceDeletionPreview":
      return { documents: 5, highlights: 2, bookmarks: 0, collectionEntries: 1, deletesFilesOnDisk: args[0] === 3 }
    case "Native.PickFolder":
      return ["/Users/me/mock-folder", false]
    case "Native.PickZipFile":
      return ["/Users/me/mock.zip", false]
    case "Native.PickSaveLocation":
      return [`/Users/me/${args[0] || "export"}`, false]
    default:
      throw new Error(`dev: no mock for ${key} (${args.join(", ")})`)
  }
}

function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms))
}

// Export __devStore for tests that need to override the dev path.
export const __devStore = devStore
