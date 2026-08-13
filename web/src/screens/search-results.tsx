// search-results.tsx — query in the URL, debounced, scope + "In code" toggle,
// FTS snippets with matches already emphasized.

import { useEffect, useRef, useState } from "react"
import { useLocation, useNavigate } from "react-router-dom"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Checkbox } from "@/components/ui/checkbox"
import { EmptyState, DocumentRow } from "@/components/rows"
import { search } from "@/lib/wails"
import { useEvent } from "@/lib/events"

export function SearchResults() {
  const location = useLocation()
  const navigate = useNavigate()
  const [query, setQuery] = useState(() => new URLSearchParams(location.search).get("q") || "")
  const [scope, setScope] = useState("library")
  const [includeCode, setIncludeCode] = useState(false)
  const [rows, setRows] = useState<any[]>([])
  const [meta, setMeta] = useState("")
  const [offset, setOffset] = useState(0)
  const [more, setMore] = useState(false)
  const [stale, setStale] = useState(false)
  const timer = useRef<number | null>(null)

  const ctx = parseCtx(location.search)
  useEffect(() => {
    if (ctx.type === "source") setScope("source")
    if (ctx.type === "collection") setScope("collection")
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEvent("search:invalidated", () => setStale(true))

  function run(q: string, append = false) {
    const queryStr = q.trim()
    if (!queryStr) {
      setRows([])
      setMeta("")
      return
    }
    const startOffset = append ? offset + 10 : 0
    void search.run(queryStr, scope, scope === "library" ? 0 : ctx.id, includeCode, startOffset).then(([data, err]) => {
      if (err) {
        setMeta(err.content)
        return
      }
      const results = data?.results || []
      setRows((prev) => (append ? [...prev, ...results] : results))
      setMeta(`${(append ? offset + 10 + results.length : results.length)} results · ${data?.elapsedMs}ms`)
      setMore(results.length === 10)
      setOffset(startOffset)
    })
  }

  // Prime from URL on mount.
  useEffect(() => {
    const q = new URLSearchParams(location.search).get("q")
    if (q) {
      setQuery(q)
      run(q)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <div className="mx-auto max-w-3xl space-y-4 px-6 py-8">
      <div className="flex flex-wrap items-center gap-2">
        <Input value={query} onChange={(e) => {
          setQuery(e.target.value)
          if (timer.current) window.clearTimeout(timer.current)
          timer.current = window.setTimeout(() => run(e.target.value), 150)
        }} placeholder="Search the library…" className="max-w-xs" />
        <Select value={scope} onValueChange={(v) => { setScope(v); run(query) }}>
          <SelectTrigger className="w-44"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="library">Library</SelectItem>
            <SelectItem value="source">Current source</SelectItem>
            <SelectItem value="collection">Current collection</SelectItem>
          </SelectContent>
        </Select>
        <label className="flex items-center gap-2 text-sm">
          <Checkbox checked={includeCode} onCheckedChange={(v) => { setIncludeCode(!!v); run(query) }} />
          Search in code
        </label>
      </div>

      <div className="text-xs text-muted-foreground">{meta}</div>
      {stale && <div className="rounded border border-warning/40 px-3 py-2 text-xs text-warning">Results may be out of date — the index changed.</div>}

      <div className="space-y-1">
        {rows.length === 0 && query && (
          <EmptyState
            icon="🔍"
            headline={`No results for "${query}"`}
            action={{ label: "Search in code", onClick: () => { setIncludeCode(true); run(query) } }}
          />
        )}
        {rows.length === 0 && !query && (
          <EmptyState icon="🔍" headline="Search your library" body="Type to search across every imported document." action={{ label: "Recent searches", onClick: () => {} }} />
        )}
        {rows.map((r) => (
          <div key={r.documentId}>
            <DocumentRow
              title={r.title}
              relPath={r.relPath}
              sourceName={r.sourceName}
              onClick={() => navigate(`/doc/${r.documentId}?q=${encodeURIComponent(query)}`)}
            />
            <div className="px-2 pb-1 text-xs text-muted-foreground" dangerouslySetInnerHTML={{ __html: r.snippet || "" }} />
          </div>
        ))}
      </div>

      {more && (
        <Button variant="outline" onClick={() => run(query, true)}>Load 10 more</Button>
      )}
    </div>
  )
}

function parseCtx(search: string) {
  const m = search.match(/[?&]ctx=(source|collection):(\d+)/)
  return m ? { type: m[1], id: Number(m[2]) } : { type: "library", id: 0 }
}
