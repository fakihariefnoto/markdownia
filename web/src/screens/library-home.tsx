// library-home.tsx — a modern library dashboard: Continue Reading cards,
// Collections and Sources grids. Each section scrolls internally.

import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { BookOpen, FolderPlus, FolderGit2, ArrowRight, Search } from "lucide-react"
import { EmptyState, SourceStatusBadge, SkeletonList } from "@/components/rows"
import { library, sources, collections } from "@/lib/wails"
import { dispatchAction } from "@/lib/shortcuts"
import { useEvent } from "@/lib/events"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"

export function LibraryHome() {
  const navigate = useNavigate()
  const [recents, setRecents] = useState<any[]>([])
  const [cols, setCols] = useState<any[]>([])
  const [srcs, setSrcs] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  async function load(silent = false) {
    if (!silent) setLoading(true)
    const [r] = await library.recent(8)
    const [c] = await collections.list()
    const [s] = await sources.list()
    setRecents(r || [])
    setCols(c || [])
    setSrcs(s || [])
    setLoading(false)
  }

  useEffect(() => {
    void load()
    const onColsChanged = () => void load(true)
    window.addEventListener("markdownia:collections-changed", onColsChanged)
    return () => window.removeEventListener("markdownia:collections-changed", onColsChanged)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEvent("source:indexed", () => void load(true))
  useEvent("source:progress", () => void load(true))

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center p-8">
        <SkeletonList rows={5} height={64} />
      </div>
    )
  }

  const hasAny = recents.length + cols.length + srcs.length > 0

  if (!hasAny) {
    return <WelcomeEmpty />
  }

  return (
    <div className="mx-auto flex h-full w-full max-w-5xl flex-col gap-4 overflow-hidden px-6 py-5">
      {/* Header row */}
      <div className="flex shrink-0 items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-foreground">Library</h1>
          <p className="mt-0.5 text-[13px] text-muted-foreground">
            {srcs.length} source{srcs.length === 1 ? "" : "s"} · {recents.length > 0 ? "resume where you left off" : "your reading space"}
          </p>
        </div>
        <button
          onClick={() => navigate("/search")}
          className="flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-[13px] text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <Search className="size-3.5" />
          Search
        </button>
      </div>

      {/* Continue Reading */}
      <HomeSection title="Continue Reading" count={recents.length} hint="pick up where you left off" className="shrink-0">
        {recents.length === 0 ? (
          <EmptyState
            icon="📖"
            headline="Nothing read yet"
            body="Open a document from the tree or search to start."
            action={{ label: "Search Library", onClick: () => navigate("/search") }}
          />
        ) : (
          <div className="flex gap-3 overflow-x-auto pb-1">
            {recents.map((r) => (
              <RecentCard key={r.documentId} r={r} onClick={() => navigate(`/doc/${r.documentId}`)} />
            ))}
          </div>
        )}
      </HomeSection>

      {/* Collections */}
      <HomeSection title="Collections" count={cols.length} className="min-h-0 flex-1">
        {cols.length === 0 ? (
          <EmptyState
            icon="🗂"
            headline="No collections yet"
            body="Group documents from any source into curated reading lists."
            action={{ label: "New Collection", onClick: () => dispatchAction("new-collection") }}
          />
        ) : (
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            {cols.map((c) => (
              <CollectionCard key={c.id} c={c} onClick={() => navigate(`/collection/${c.id}`)} />
            ))}
          </div>
        )}
      </HomeSection>

      {/* Sources */}
      <HomeSection title="Sources" count={srcs.length} className="min-h-0 flex-1">
        {srcs.length === 0 ? (
          <EmptyState
            icon="📁"
            headline="Import a source to begin"
            body="A folder, a git repo, or a zip of markdown files."
            action={{ label: "Import", onClick: () => dispatchAction("import-folder") }}
          />
        ) : (
          <div className="grid grid-cols-1 gap-2">
            {srcs.map((s) => (
              <SourceRow key={s.id} s={s} onClick={() => navigate(`/source/${s.id}`)} />
            ))}
          </div>
        )}
      </HomeSection>
    </div>
  )
}

// HomeSection — a bordered, titled block with internal scrolling.
function HomeSection({ title, count, hint, className, children }: { title: string; count: number; hint?: string; className?: string; children: React.ReactNode }) {
  return (
    <section className={cn("flex min-h-0 flex-col overflow-hidden rounded-xl border border-border bg-card", className)}>
      <div className="flex shrink-0 items-baseline justify-between border-b border-border px-4 py-2.5">
        <div className="flex items-baseline gap-2">
          <h2 className="text-[13px] font-semibold text-foreground">{title}</h2>
          {count > 0 && <span className="text-xs tabular-nums text-muted-foreground">{count}</span>}
        </div>
        {hint && <span className="text-[11px] text-muted-foreground">{hint}</span>}
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-3">{children}</div>
    </section>
  )
}

function RecentCard({ r, onClick }: { r: any; onClick: () => void }) {
  const pct = Math.round((r.furthestScrollPct || 0) * 100)
  return (
    <button
      onClick={onClick}
      className="group flex w-52 shrink-0 cursor-pointer flex-col justify-between gap-2 rounded-xl border border-border bg-card p-3.5 text-left shadow-sm transition-all hover:-translate-y-0.5 hover:border-primary/50 hover:shadow-md"
    >
      <div className="flex items-start gap-2.5">
        <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <BookOpen className="size-4" />
        </span>
        <div className="min-w-0">
          <div className="truncate text-[13px] font-medium text-foreground">{r.title}</div>
          <div className="truncate text-[11px] text-muted-foreground">{r.relPath}</div>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <span className="h-1 flex-1 overflow-hidden rounded-full bg-muted">
          <span className="block h-full rounded-full bg-primary" style={{ width: `${pct}%` }} />
        </span>
        <span className="text-[10px] tabular-nums text-muted-foreground">{pct}%</span>
      </div>
    </button>
  )
}

function CollectionCard({ c, onClick }: { c: any; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="group flex cursor-pointer items-center gap-3 rounded-lg border border-border bg-card p-3 text-left transition-colors hover:border-primary/50 hover:bg-accent"
    >
      <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
        <FolderPlus className="size-4" />
      </span>
      <div className="min-w-0 flex-1">
        <div className="truncate text-[13px] font-medium text-foreground">{c.name}</div>
        <div className="text-[11px] text-muted-foreground">{c.documentCount} documents</div>
      </div>
      <ArrowRight className="size-3.5 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
    </button>
  )
}

function SourceRow({ s, onClick }: { s: any; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="group flex w-full cursor-pointer items-center gap-3 rounded-lg border border-border bg-card p-3 text-left transition-colors hover:border-primary/50 hover:bg-accent"
    >
      <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
        <FolderGit2 className="size-4" />
      </span>
      <div className="min-w-0 flex-1">
        <div className="truncate text-[13px] font-medium text-foreground">{s.name}</div>
        <div className="truncate text-[11px] text-muted-foreground">
          {s.kind === "git" ? s.originUrl || "git repo" : s.kind} · {s.documentCount} docs
        </div>
      </div>
      <SourceStatusBadge status={s.status} errorMessage={s.errorMessage} />
    </button>
  )
}

function WelcomeEmpty() {
  const navigate = useNavigate()
  return (
    <div className="flex h-full items-center justify-center p-8">
      <div className="mx-auto flex max-w-md flex-col items-center gap-5 text-center">
        <span className="flex size-16 items-center justify-center rounded-2xl bg-primary/10 text-primary">
          <BookOpen className="size-8" />
        </span>
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-foreground">Welcome to Markdownia</h1>
          <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
            Your offline reading library. Import a folder, a git repository, or a zip of markdown files to get started.
          </p>
        </div>
        <div className="grid w-full grid-cols-1 gap-2 sm:grid-cols-3">
          <button
            onClick={() => dispatchAction("import-folder")}
            className="flex cursor-pointer items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
          >
            <FolderGit2 className="size-4" /> Import Folder
          </button>
          <button
            onClick={() => dispatchAction("import-git")}
            className="flex cursor-pointer items-center justify-center gap-2 rounded-lg border border-border bg-card px-4 py-2.5 text-sm font-medium text-foreground transition-colors hover:bg-muted"
          >
            <FolderPlus className="size-4" /> Git Repo
          </button>
          <button
            onClick={() => dispatchAction("import-zip")}
            className="flex cursor-pointer items-center justify-center gap-2 rounded-lg border border-border bg-card px-4 py-2.5 text-sm font-medium text-foreground transition-colors hover:bg-muted"
          >
            <Search className="size-4" /> Zip
          </button>
        </div>
      </div>
    </div>
  )
}
