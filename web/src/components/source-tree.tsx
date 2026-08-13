// source-tree.tsx — the sidebar tree. The backend returns a flat, depth-tagged
// list (folder nodes injected before their docs). This component derives the
// hierarchy, renders collapsible folders, and navigates on doc click.

import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Folder, FolderOpen, FileText, ChevronRight } from "lucide-react"
import { sources, library, collections } from "@/lib/wails"
import { cn } from "@/lib/utils"

interface TreeNode {
  id: number
  relPath: string
  title: string
  depth: number
  isFolder: boolean
}

interface TreeState {
  kind: "library" | "sources" | "collections"
  sourceItems: any[]
  collectionItems: any[]
  loaded: Set<number>
  nodes: Record<number, TreeNode[]>
  // openFolders: expanded folder relPaths per source
  openFolders: Record<number, Set<string>>
}

export function SourceTree({ ctx, filter, route }: { ctx: "library" | "sources" | "collections"; filter: string; route: string }) {
  const navigate = useNavigate()
  const [state, setState] = useState<TreeState>({
    kind: ctx,
    sourceItems: [],
    collectionItems: [],
    loaded: new Set(),
    nodes: {},
    openFolders: {},
  })
  // A folder to reveal (from a doc link): expand its path and highlight it.
  const [reveal, setReveal] = useState<{ sourceId: number; folderRel: string } | null>(null)

  useEffect(() => {
    let cancelled = false
    setState((s) => ({ ...s, kind: ctx }))
    async function load() {
      if (ctx === "collections") {
        const [cols] = await collections.list()
        if (!cancelled) setState((s) => ({ ...s, kind: ctx, collectionItems: cols || [] }))
      } else {
        const [list] = await sources.list()
        if (!cancelled) setState((s) => ({ ...s, kind: ctx, sourceItems: list || [] }))
      }
    }
    void load()
    const onColsChanged = () => {
      if (ctx === "collections") void load()
    }
    window.addEventListener("markdownia:collections-changed", onColsChanged)
    return () => {
      cancelled = true
      window.removeEventListener("markdownia:collections-changed", onColsChanged)
    }
  }, [ctx])

  // Reveal a folder when a doc link targets it: ensure the source + every
  // ancestor folder is expanded, then highlight.
  useEffect(() => {
    const onReveal = (e: Event) => {
      const d = (e as CustomEvent).detail
      if (!d?.sourceId || !d?.folderRel) return
      setReveal({ sourceId: d.sourceId, folderRel: d.folderRel })
      setState((s) => {
        const open = new Set(s.openFolders[d.sourceId] || [])
        open.add("") // source root
        let cur = d.folderRel
        while (cur !== "") {
          open.add(cur)
          const i = cur.lastIndexOf("/")
          cur = i >= 0 ? cur.slice(0, i) : ""
        }
        const loaded = new Set(s.loaded)
        if (!loaded.has(d.sourceId)) {
          loaded.add(d.sourceId)
          void library.tree(d.sourceId).then(([nodes]) => {
            setState((prev) => ({ ...prev, nodes: { ...prev.nodes, [d.sourceId]: nodes || [] } }))
          })
        }
        return { ...s, openFolders: { ...s.openFolders, [d.sourceId]: open }, loaded }
      })
      // Clear after a moment so the highlight can fade.
      setTimeout(() => setReveal(null), 3000)
    }
    window.addEventListener("markdownia:reveal-folder", onReveal)
    return () => window.removeEventListener("markdownia:reveal-folder", onReveal)
  }, [])

  // When the route changes to a document, auto-expand the tree to reveal and
  // highlight that document in the sidebar.
  useEffect(() => {
    const m = route.match(/^\/doc\/(\d+)/)
    if (!m) return
    const docId = Number(m[1])
    // Find the source whose nodes contain this doc; load the tree on demand.
    const entries = Object.entries(state.nodes)
    for (const [sidStr, nodes] of entries) {
      const srcId = Number(sidStr)
      const doc = (nodes || []).find((n) => !n.isFolder && n.id === docId)
      if (!doc) {
        continue
      }
      revealDocPath(srcId, doc.relPath)
      return
    }
    // Not found in loaded trees — fetch each source's tree until it matches.
    let cancelled = false
    void (async () => {
      for (const src of state.sourceItems) {
        const [nodes] = await library.tree(src.id)
        if (cancelled) return
        const doc = (nodes || []).find((n: any) => !n.isFolder && n.id === docId)
        if (!doc) continue
        setState((prev) => ({ ...prev, nodes: { ...prev.nodes, [src.id]: nodes || [] } }))
        revealDocPath(src.id, doc.relPath)
        return
      }
    })()
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [route, state.sourceItems])

  function revealDocPath(srcId: number, relPath: string) {
    setState((s) => {
      const open = new Set(s.openFolders[srcId] || [])
      open.add("")
      let cur = relPath
      while (cur !== "") {
        const i = cur.lastIndexOf("/")
        cur = i >= 0 ? cur.slice(0, i) : ""
        if (cur !== "") open.add(cur)
      }
      return { ...s, openFolders: { ...s.openFolders, [srcId]: open } }
    })
  }

  function toggleSource(srcId: number) {
    setState((s) => {
      const open = new Set(s.openFolders[srcId] || [])
      if (open.has("")) {
        open.delete("")
        return { ...s, openFolders: { ...s.openFolders, [srcId]: open } }
      }
      open.add("")
      if (!s.loaded.has(srcId)) {
        const loaded = new Set(s.loaded)
        loaded.add(srcId)
        void library.tree(srcId).then(([nodes]) => {
          setState((prev) => ({ ...prev, nodes: { ...prev.nodes, [srcId]: nodes || [] } }))
        })
        return { ...s, openFolders: { ...s.openFolders, [srcId]: open }, loaded }
      }
      return { ...s, openFolders: { ...s.openFolders, [srcId]: open } }
    })
  }

  function toggleFolder(srcId: number, relPath: string) {
    setState((s) => {
      const open = new Set(s.openFolders[srcId] || [])
      if (open.has(relPath)) open.delete(relPath)
      else open.add(relPath)
      return { ...s, openFolders: { ...s.openFolders, [srcId]: open } }
    })
  }

  const f = filter.toLowerCase()

  if (ctx === "collections") {
    return (
      <div className="flex flex-col gap-0.5">
        {state.collectionItems.length === 0 && (
          <p className="px-2 py-1 text-xs text-muted-foreground">No collections yet.</p>
        )}
        {state.collectionItems
          .filter((c) => !f || c.name.toLowerCase().includes(f))
          .map((c) => (
            <button
              key={c.id}
              onClick={() => navigate(`/collection/${c.id}`)}
              className={cn(
                "flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
                route === `/collection/${c.id}` && "bg-sidebar-accent font-medium text-sidebar-accent-foreground"
              )}
            >
              <Folder className="size-4 shrink-0" />
              <span className="truncate">{c.name}</span>
              <span className="ml-auto text-xs text-muted-foreground">{c.documentCount}</span>
            </button>
          ))}
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      {state.sourceItems.length === 0 && (
        <p className="px-2 py-1 text-xs text-muted-foreground">
          {ctx === "sources" ? "No sources yet." : "Import a source to get started."}
        </p>
      )}
      {state.sourceItems.map((src) => {
        const srcOpen = (state.openFolders[src.id] || new Set()).has("")
        const isActive = route === `/source/${src.id}`
        const nodes = state.nodes[src.id] || []
        return (
          <div key={src.id}>
            <button
              onClick={() => toggleSource(src.id)}
              className={cn(
                "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
                isActive && "bg-sidebar-accent font-medium text-sidebar-accent-foreground"
              )}
            >
              <ChevronRight className={cn("size-3.5 shrink-0 text-muted-foreground transition-transform", srcOpen && "rotate-90")} />
              <Folder className={cn("size-4 shrink-0", srcOpen && "text-primary")} />
              <span className="truncate">{src.name}</span>
              <span className="ml-auto text-xs text-muted-foreground">{src.documentCount}</span>
            </button>
            {srcOpen && (
              <div className="ml-2">
                <FolderTree
                  nodes={nodes}
                  filter={f}
                  route={route}
                  srcId={src.id}
                  openFolders={state.openFolders[src.id] || new Set()}
                  revealRel={reveal && reveal.sourceId === src.id ? reveal.folderRel : undefined}
                  onToggleFolder={(rel) => toggleFolder(src.id, rel)}
                  onOpenDoc={(n) => {
                    navigate(`/doc/${n.id}?ctx=source:${src.id}`)
                    window.dispatchEvent(new CustomEvent("markdownia:doc-opened", { detail: { documentId: n.id, title: n.title, relPath: n.relPath, pane: 0 } }))
                  }}
                />
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

// FolderTree renders a flat, depth-tagged node list as a collapsible hierarchy.
// A node is visible when every ancestor folder is open.
function FolderTree({
  nodes,
  filter,
  route,
  srcId,
  openFolders,
  revealRel,
  onToggleFolder,
  onOpenDoc,
}: {
  nodes: TreeNode[]
  filter: string
  route: string
  srcId: number
  openFolders: Set<string>
  revealRel?: string
  onToggleFolder: (rel: string) => void
  onOpenDoc: (n: TreeNode) => void
}) {
  // Filter: if a query is active, show everything matching, regardless of
  // folder state (flatten).
  const filteredNodes = filter
    ? nodes.filter((n) => n.title.toLowerCase().includes(filter) || n.relPath.toLowerCase().includes(filter))
    : nodes

  if (filter) {
    return (
      <div className="flex flex-col gap-0.5">
        {filteredNodes.map((n) => (
          <button
            key={n.relPath}
            onClick={() => !n.isFolder && onOpenDoc(n)}
            className="flex w-full items-center gap-1.5 rounded-md px-2 py-1 text-xs text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
            style={{ paddingLeft: `${n.depth * 12 + 8}px` }}
          >
            {n.isFolder ? <Folder className="size-3.5 shrink-0" /> : <FileText className="size-3.5 shrink-0" />}
            <span className="truncate">{n.title}</span>
          </button>
        ))}
      </div>
    )
  }

  // Walk the flat list; hide a node when any ancestor folder (including its
  // immediate parent) is closed.
  const visible: Array<{ node: TreeNode; hiddenAncestor: boolean }> = []
  for (const n of nodes) {
    let hiddenAncestor = false
    for (const anc of ancestorDirs(n.relPath)) {
      if (anc !== "" && !openFolders.has(anc)) {
        hiddenAncestor = true
        break
      }
    }
    visible.push({ node: n, hiddenAncestor })
  }

  return (
    <div className="flex flex-col gap-0.5">
      {visible.length === 0 && <p className="px-2 py-0.5 text-xs text-muted-foreground">Empty</p>}
      {visible.map(({ node: n, hiddenAncestor }) => {
        if (hiddenAncestor) return null
        const isOpen = openFolders.has(n.relPath)
        const isDocActive = route === `/doc/${n.id}`
        const isRevealed = revealRel != null && n.isFolder && n.relPath === revealRel
        if (n.isFolder) {
          return (
            <div key={n.relPath}>
              <button
                onClick={() => onToggleFolder(n.relPath)}
                className={cn(
                  "flex w-full items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
                  isRevealed && "bg-primary/15 text-primary ring-1 ring-primary/40"
                )}
                style={{ paddingLeft: `${n.depth * 12 + 8}px` }}
              >
                <ChevronRight className={cn("size-3 shrink-0 transition-transform", isOpen && "rotate-90")} />
                {isOpen ? <FolderOpen className="size-3.5 shrink-0 text-primary" /> : <Folder className="size-3.5 shrink-0" />}
                <span className="truncate">{n.title}</span>
              </button>
            </div>
          )
        }
        return (
          <button
            key={n.relPath}
            onClick={() => onOpenDoc(n)}
            className={cn(
              "flex w-full items-center gap-1.5 rounded-md px-2 py-1 text-xs text-muted-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
              isDocActive && "bg-sidebar-accent font-medium text-sidebar-accent-foreground"
            )}
            style={{ paddingLeft: `${n.depth * 12 + 8}px` }}
          >
            <FileText className="size-3.5 shrink-0" />
            <span className="truncate">{n.title}</span>
          </button>
        )
      })}
    </div>
  )
}

// ancestorDirs returns every ancestor directory of a rel path, deepest first,
// excluding the path itself. The nearest ancestor (the node's parent folder) is
// included, so a node is hidden whenever any folder on its path is collapsed.
function ancestorDirs(rel: string): string[] {
  const out: string[] = []
  let cur = rel
  while (cur !== "") {
    const i = cur.lastIndexOf("/")
    if (i < 0) break
    cur = cur.slice(0, i)
    out.push(cur)
  }
  return out
}
