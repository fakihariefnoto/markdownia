// shell.tsx — the app shell. A distinct sidebar column (brand, navigation,
// source tree) on the left; a content column on the right with its own header
// (title + prev/next + tabs) and the routed main content.

import { useEffect, useState } from "react"
import { Outlet, useLocation, useNavigate } from "react-router-dom"
import { Separator } from "@/components/ui/separator"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Input } from "@/components/ui/input"
import { Library, FolderGit2, Archive, Bookmark, Settings as SettingsIcon, Import, Files, Search, LayoutGrid, ChevronLeft, ChevronRight } from "lucide-react"
import { SourceTree } from "@/components/source-tree"
import { TabStrip } from "@/components/tab-strip"
import { registerAction } from "@/lib/shortcuts"
import { isMac } from "@/lib/shortcuts"
import { navigate } from "@/lib/nav"
import { cn } from "@/lib/utils"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { library, sources } from "@/lib/wails"
import markdowniaIcon from "@/assets/markdownia.png"

const SIDEBAR_WIDTH = 260

export function Shell() {
  return <ShellLayout />
}

function ShellLayout() {
  const location = useLocation()
  const route = location.pathname
  const isFirstRun = route === "/welcome"

  const [ctx, setCtx] = useState<"library" | "sources" | "collections">("library")
  const [outlineVisible, setOutlineVisible] = useState(true)

  useEffect(() => {
    const unregs = [registerAction("outline-toggle", () => setOutlineVisible((v) => !v))]
    return () => unregs.forEach((u) => u?.())
  }, [])

  return (
    <div className={cn("flex h-screen w-full overflow-hidden bg-background text-foreground", isFirstRun && "hidden")}>
      {/* Sidebar column — its own surface colour, separated by a border. */}
      <aside
        className="flex h-full shrink-0 flex-col border-r border-border bg-sidebar"
        style={{ width: SIDEBAR_WIDTH }}
      >
        {/* macOS traffic lights overlay the top-left; keep them clear. */}
        {isMac && <div className="h-10 shrink-0" style={{ "--wails-draggable": "drag" } as React.CSSProperties} />}

        <div className="flex h-14 shrink-0 items-center gap-2.5 px-4">
          <span className="flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-lg">
            <img src={markdowniaIcon} alt="Markdownia" className="size-full object-cover" />
          </span>
          <div className="min-w-0">
            <div className="truncate text-[15px] font-semibold leading-tight">Markdownia</div>
            <div className="truncate text-[11px] text-muted-foreground">Reading library</div>
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-2">
          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton isActive={route === "/"} onClick={() => navigate("/")} tooltip="Home">
                  <Library />
                  <span>Home</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton isActive={route.startsWith("/search")} onClick={() => navigate("/search")} tooltip="Search">
                  <Search />
                  <span>Search</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton isActive={route.startsWith("/annotations")} onClick={() => navigate("/annotations")} tooltip="Annotations">
                  <Bookmark />
                  <span>Annotations</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>

          <SidebarSeparator />

          <SidebarGroup>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton isActive={ctx === "library"} onClick={() => setCtx("library")} tooltip="Library">
                  <LayoutGrid />
                  <span>Library</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton isActive={ctx === "sources"} onClick={() => setCtx("sources")} tooltip="Sources">
                  <FolderGit2 />
                  <span>Sources</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <SidebarMenuButton isActive={ctx === "collections"} onClick={() => setCtx("collections")} tooltip="Collections">
                  <Files />
                  <span>Collections</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroup>

          <SidebarSeparator />

          <SidebarGroup>
            <SidebarMenu>
              <ImportMenuItems />
            </SidebarMenu>
          </SidebarGroup>

          <SidebarSeparator />

          <SourceTreePane ctx={ctx} route={route} />
        </div>

        <div className="shrink-0 border-t border-border p-2">
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton onClick={() => navigate("/settings")} isActive={route.startsWith("/settings")} tooltip="Settings">
                <SettingsIcon />
                <span>Settings</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </div>
      </aside>

      {/* Content column — its own surface colour. */}
      <div className="flex min-w-0 flex-1 flex-col">
        <ContentHeader route={route} />
      <div className="flex h-9 min-w-0 shrink-0 items-center gap-1 px-2">
        <TabStrip />
      </div>
        <div className="flex min-h-0 flex-1">
          <main className="min-w-0 flex-1 overflow-y-auto">
            <Outlet />
          </main>
          <OutlinePane route={route} visible={outlineVisible} />
        </div>
      </div>
    </div>
  )
}

function SidebarGroup({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn("flex flex-col gap-0.5 py-2", className)}>{children}</div>
}

function SidebarSeparator() {
  return <div className="mx-2 h-px bg-border" />
}

function SidebarMenu({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-0.5">{children}</div>
}

function SidebarMenuItem({ children }: { children: React.ReactNode }) {
  return <div className="relative">{children}</div>
}

function SidebarMenuButton({
  isActive,
  onClick,
  tooltip,
  children,
}: {
  isActive?: boolean
  onClick?: () => void
  tooltip?: string
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      title={tooltip}
      className={cn(
        "flex w-full cursor-pointer items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] font-medium transition-colors",
        isActive
          ? "bg-primary/10 text-primary"
          : "text-muted-foreground hover:bg-muted hover:text-foreground"
      )}
    >
      {children}
    </button>
  )
}

// ContentHeader — the current document title + prev/next navigation. A
// placeholder shows when nothing is open yet.
function ContentHeader({ route }: { route: string }) {
  const [sourceName, setSourceName] = useState<string | null>(null)
  const [sourceKind, setSourceKind] = useState<string | null>(null)
  const [prevDoc, setPrevDoc] = useState<{ id: number; title: string } | null>(null)
  const [nextDoc, setNextDoc] = useState<{ id: number; title: string } | null>(null)

  // Determine the active source from the URL context and the source list.
  useEffect(() => {
    let cancelled = false
    async function resolve() {
      const ctxMatch = window.location.search.match(/[?&]ctx=(source|collection):(\d+)/)
      const [list] = await sources.list()
      const srcs = list || []
      let src = null
      if (ctxMatch && ctxMatch[1] === "source") {
        src = srcs.find((s: any) => s.id === Number(ctxMatch[2])) || null
      } else if (route.startsWith("/source/")) {
        const id = Number(route.split("/")[2])
        src = srcs.find((s: any) => s.id === id) || null
      } else if (route.startsWith("/collection/")) {
        // Collection context: show the collection name is not available here;
        // keep the welcome message as the breadcrumb.
        src = null
      }
      if (!cancelled) {
        setSourceName(src?.name || null)
        setSourceKind(src?.kind || null)
      }
    }
    void resolve()
    return () => {
      cancelled = true
    }
  }, [route])

  // Reset when leaving a reader/source route.
  useEffect(() => {
    if (!route.startsWith("/doc") && !route.startsWith("/source")) {
      setSourceName(null)
      setSourceKind(null)
    }
  }, [route])

  // Build prev/next from the active source tree (ordered by title).
  useEffect(() => {
    const onOpen = (e: Event) => {
      const d = (e as CustomEvent).detail
      if (d?.documentId) void buildNav(d.documentId)
    }
    window.addEventListener("markdownia:doc-opened", onOpen)
    return () => window.removeEventListener("markdownia:doc-opened", onOpen)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [route])

  async function buildNav(docId: number) {
    const ctxMatch = window.location.search.match(/[?&]ctx=source:(\d+)/)
    if (ctxMatch) {
      const [nodes] = await library.tree(Number(ctxMatch[1]))
      const docs = (nodes || []).filter((n: any) => !n.isFolder).sort((a: any, b: any) => a.title.localeCompare(b.title))
      const idx = docs.findIndex((d: any) => d.id === docId)
      if (idx > 0) {
        const p = docs[idx - 1]
        setPrevDoc({ id: p.id, title: p.title })
      } else setPrevDoc(null)
      if (idx >= 0 && idx < docs.length - 1) {
        const n = docs[idx + 1]
        setNextDoc({ id: n.id, title: n.title })
      } else setNextDoc(null)
      return
    }
    const [recent] = await library.recent(20)
    const list = recent || []
    const idx = list.findIndex((r: any) => r.documentId === docId)
    if (idx > 0) {
      const p = list[idx - 1]
      setPrevDoc({ id: p.documentId, title: p.title })
    } else setPrevDoc(null)
    if (idx >= 0 && idx < list.length - 1) {
      const n = list[idx + 1]
      setNextDoc({ id: n.documentId, title: n.title })
    } else setNextDoc(null)
  }

  const kindLabel = sourceKind === "git" ? "Git Repository" : sourceKind === "zip" ? "Zip Archive" : "Folder"

  return (
    <header className="flex h-14 shrink-0 items-center justify-between gap-3 bg-sidebar px-4">
      {sourceName ? (
        <div className="flex min-w-0 items-center gap-2">
          {sourceKind === "git" ? (
            <FolderGit2 className="size-4 shrink-0 text-primary" />
          ) : sourceKind === "zip" ? (
            <Archive className="size-4 shrink-0 text-primary" />
          ) : (
            <FolderGit2 className="size-4 shrink-0 text-primary" />
          )}
          <span className="truncate text-sm font-medium text-foreground">{sourceName}</span>
          <span className="shrink-0 rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
            {kindLabel}
          </span>
        </div>
      ) : (
        <span className="min-w-0 truncate text-sm text-muted-foreground">
          Turn any pile of markdown into a beautifully
     rendered reading library. | open a document to start reading.
        </span>
      )}
      <div className="flex shrink-0 items-center gap-0.5">
        <NavButton
          disabled={!prevDoc}
          tooltip={prevDoc?.title ? `Previous: ${prevDoc.title}` : undefined}
          onClick={() => prevDoc && navigate(`/doc/${prevDoc.id}`)}
        >
          <ChevronLeft className="size-4" />
        </NavButton>
        <NavButton
          disabled={!nextDoc}
          tooltip={nextDoc?.title ? `Next: ${nextDoc.title}` : undefined}
          onClick={() => nextDoc && navigate(`/doc/${nextDoc.id}`)}
        >
          <ChevronRight className="size-4" />
        </NavButton>
      </div>
    </header>
  )
}

function NavButton({ disabled, tooltip, onClick, children }: { disabled?: boolean; tooltip?: string; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      disabled={disabled}
      onClick={onClick}
      title={tooltip}
      className="flex size-8 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-40"
    >
      {children}
    </button>
  )
}

function ImportMenuItems() {
  const [open, setOpen] = useState(false)
  return (
    <SidebarMenuItem>
      <SidebarMenuButton onClick={() => setOpen(true)} tooltip="Import source">
        <Import />
        <span>Import…</span>
      </SidebarMenuButton>
      <ImportDialog open={open} onOpenChange={setOpen} />
    </SidebarMenuItem>
  )
}

function ImportDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Import a source</DialogTitle>
          <DialogDescription className="text-muted-foreground">
            Choose how you want to bring markdown into your library.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-3 sm:grid-cols-3">
          <ImportCard
            icon={<FolderGit2 className="size-6" />}
            title="Folder"
            description="Read notes in place. Files are referenced, never copied."
            onClick={() => { onOpenChange(false); void import("@/components/import-menu").then((m) => m.importFolder()) }}
          />
          <ImportCard
            icon={<Import className="size-6" />}
            title="Git Repository"
            description="Paste a URL and the docs are cloned locally."
            onClick={() => { onOpenChange(false); void import("@/screens/import-git").then((m) => m.openImportGit()) }}
          />
          <ImportCard
            icon={<Archive className="size-6" />}
            title="Zip Archive"
            description="Extract a downloaded docs archive into the app's storage."
            onClick={() => { onOpenChange(false); void import("@/components/import-menu").then((m) => m.importZip()) }}
          />
        </div>
      </DialogContent>
    </Dialog>
  )
}

function ImportCard({ icon, title, description, onClick }: { icon: React.ReactNode; title: string; description: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="group flex cursor-pointer flex-col items-start gap-2 rounded-lg border border-border bg-card p-4 text-left transition-colors hover:border-primary/60 hover:bg-accent"
    >
      <span className="flex size-10 items-center justify-center rounded-md bg-primary/10 text-primary transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
        {icon}
      </span>
      <span className="font-medium">{title}</span>
      <span className="text-xs leading-relaxed text-muted-foreground">{description}</span>
    </button>
  )
}

function SourceTreePane({ ctx, route }: { ctx: string; route: string }) {
  const [query, setQuery] = useState("")
  const label = ctx === "collections" ? "Collections" : ctx === "sources" ? "Sources" : "Library"
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex items-center justify-between px-3 pb-1 pt-2">
        <span className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">{label}</span>
      </div>
      <div className="px-3 pb-1">
        <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Filter tree…" className="h-7 text-xs" />
      </div>
      <ScrollArea className="min-h-0 flex-1">
        <div className="px-1 pb-2">
          <SourceTree ctx={ctx as any} filter={query} route={route} />
        </div>
      </ScrollArea>
    </div>
  )
}

function OutlinePane({ route, visible }: { route: string; visible: boolean }) {
  const [headings, setHeadings] = useState<Array<{ anchor: string; text: string; level: number }>>([])

  useEffect(() => {
    if (!route.startsWith("/doc")) return
    const collect = () => {
      const doc = document.querySelector(".reading-surface .doc")
      if (!doc) return
      const hs = Array.from(doc.querySelectorAll("h1,h2,h3,h4,h5,h6")).map((h) => ({
        anchor: h.getAttribute("id") || "",
        text: (h.textContent || "").trim(),
        level: Number(h.tagName[1]),
      })).filter((h) => h.anchor)
      setHeadings(hs)
      window.__currentOutline = hs
    }
    const t = setTimeout(collect, 100)
    return () => {
      clearTimeout(t)
      window.__currentOutline = []
    }
  }, [route])

  if (!route.startsWith("/doc") || !visible) return null

  return (
    <aside className="hidden w-56 shrink-0 border-l border-border bg-card lg:flex flex-col">
      <div className="px-4 py-2 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">Outline</div>
      <ScrollArea className="flex-1">
        <div className="px-2 pb-4">
          {headings.length === 0 && <p className="px-2 py-1 text-xs text-muted-foreground">No headings</p>}
          {headings.map((h) => (
            <button
              key={h.anchor}
              onClick={() => {
                document.getElementById(h.anchor)?.scrollIntoView({ behavior: "smooth", block: "start" })
              }}
              className="block w-full truncate rounded px-2 py-1 text-left text-xs text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              style={{ paddingLeft: `${(h.level - 1) * 12 + 8}px` }}
            >
              {h.text}
            </button>
          ))}
        </div>
      </ScrollArea>
    </aside>
  )
}
