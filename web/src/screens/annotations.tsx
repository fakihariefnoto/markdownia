// annotations.tsx — bookmarks and highlights, filterable.

import { useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { EmptyState } from "@/components/rows"
import { annotations } from "@/lib/wails"
import { toast } from "@/lib/toast"
import { Bookmark, Highlighter } from "lucide-react"
import { cn } from "@/lib/utils"

export function Annotations() {
  const navigate = useNavigate()
  const [all, setAll] = useState<any>({ bookmarks: [], highlights: [] })
  const [filterText, setFilterText] = useState("")
  const [typeFilter, setTypeFilter] = useState("all")
  const [sort, setSort] = useState("newest")

  async function load() {
    const [data] = await annotations.all()
    setAll(data || { bookmarks: [], highlights: [] })
  }

  useEffect(() => {
    void load()
  }, [])

  const f = filterText.toLowerCase()
  const { rows, hadAny, hasContent } = useMemo(() => {
    let bms = [...(all.bookmarks || [])]
    let hls = [...(all.highlights || [])]
    if (f) {
      bms = bms.filter((b) => (b.title || "").toLowerCase().includes(f) || (b.note || "").toLowerCase().includes(f))
      hls = hls.filter((h) => (h.quotedText || "").toLowerCase().includes(f) || (h.note || "").toLowerCase().includes(f))
    }
    if (typeFilter === "bookmarks") hls = []
    if (typeFilter === "highlights") bms = []
    const cmp = (a: any, b: any) =>
      sort === "oldest"
        ? (a.createdAt || "").localeCompare(b.createdAt || "")
        : (b.createdAt || "").localeCompare(a.createdAt || "")
    bms.sort(cmp)
    hls.sort(cmp)
    const rows = [
      ...bms.map((b) => ({ type: "bookmark", item: b })),
      ...hls.map((h) => ({ type: "highlight", item: h })),
    ]
    return {
      rows,
      hadAny: (all.bookmarks?.length || 0) + (all.highlights?.length || 0) > 0,
      hasContent: rows.length > 0,
    }
  }, [all, filterText, typeFilter, sort])

  async function removeBookmark(b: any) {
    void annotations.removeBookmark(b.id)
    setAll((prev: any) => ({ ...prev, bookmarks: (prev.bookmarks || []).filter((x: any) => x.id !== b.id) }))
    toast({ type: "success", title: "Annotation removed", action: { label: "Undo", onClick: () => void annotations.addBookmark(b.documentId, b.headingAnchor || "", b.note || "") } })
  }

  async function removeHighlight(h: any) {
    void annotations.removeHighlight(h.id)
    setAll((prev: any) => ({ ...prev, highlights: (prev.highlights || []).filter((x: any) => x.id !== h.id) }))
    toast({
      type: "success",
      title: "Annotation removed",
      action: {
        label: "Undo",
        onClick: () =>
          void annotations.addHighlight(h.documentId, {
            blockHash: h.blockHash,
            blockIndex: h.blockIndex,
            startOffset: h.startOffset,
            endOffset: h.endOffset,
          }, h.color, h.note || ""),
      },
    })
  }

  return (
    <div className="mx-auto max-w-3xl space-y-4 px-6 py-8">
      <h1 className="text-2xl font-semibold tracking-tight">Annotations</h1>
      <div className="flex items-center gap-2">
        <Input value={filterText} onChange={(e) => setFilterText(e.target.value)} placeholder="Filter annotations…" className="max-w-xs" />
        <Select value={typeFilter} onValueChange={setTypeFilter}>
          <SelectTrigger className="w-36"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="bookmarks">Bookmarks</SelectItem>
            <SelectItem value="highlights">Highlights</SelectItem>
          </SelectContent>
        </Select>
        <Select value={sort} onValueChange={setSort}>
          <SelectTrigger className="w-36"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="newest">Newest</SelectItem>
            <SelectItem value="oldest">Oldest</SelectItem>
            <SelectItem value="by-doc">By document</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-1">
        {!hadAny ? (
          <EmptyState
            icon="📌"
            headline="No annotations yet"
            body="Bookmark documents or highlight passages while reading."
            action={{ label: "Library Home", onClick: () => navigate("/") }}
          />
        ) : !hasContent ? (
          <EmptyState
            icon="🔍"
            headline="Nothing matches your filters"
            body="Try clearing the filter."
            action={{ label: "Clear filter", onClick: () => { setFilterText(""); setTypeFilter("all") } }}
          />
        ) : (
          rows.map(({ type, item }) => (
            <AnnotationRow key={`${type}-${item.id}`} type={type} item={item} onRemove={() => (type === "bookmark" ? removeBookmark(item) : removeHighlight(item))} />
          ))
        )}
      </div>
    </div>
  )
}

function AnnotationRow({ type, item, onRemove }: { type: string; item: any; onRemove: () => void }) {
  const navigate = useNavigate()
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => (type === "bookmark" ? navigate(`/doc/${item.documentId}`) : navigate(`/doc/${item.documentId}#hl-${item.id}`))}
      onKeyDown={(e) => e.key === "Enter" && (type === "bookmark" ? navigate(`/doc/${item.documentId}`) : navigate(`/doc/${item.documentId}#hl-${item.id}`))}
      className="group flex cursor-pointer items-start gap-3 rounded-md px-2 py-2 hover:bg-accent"
    >
      {type === "bookmark" ? (
        <Bookmark className="mt-0.5 size-4 shrink-0 text-primary" />
      ) : (
        <span className={cn("mt-0.5 size-3 shrink-0 rounded-full", `bg-${item.color}`)} style={{ background: highlightColor(item.color) }} />
      )}
      <div className="min-w-0 flex-1">
        <div className="text-sm font-medium">{type === "bookmark" ? item.title : (item.quotedText || "…")}</div>
        <div className="truncate text-xs text-muted-foreground">{[item.sourceName, item.relPath].filter(Boolean).join(" · ")}</div>
        {item.note && <div className="mt-1 text-xs text-muted-foreground">{item.note}</div>}
      </div>
      <Button variant="ghost" size="icon" className="size-6 opacity-0 group-hover:opacity-100" onClick={(e) => { e.stopPropagation(); onRemove() }}>
        ✕
      </Button>
    </div>
  )
}

function highlightColor(color: string) {
  const map: Record<string, string> = {
    yellow: "var(--highlight-yellow)",
    green: "var(--highlight-green)",
    blue: "var(--highlight-blue)",
    pink: "var(--highlight-pink)",
    orange: "var(--highlight-orange)",
  }
  return map[color] || "var(--highlight-yellow)"
}
