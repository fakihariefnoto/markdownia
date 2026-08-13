// tab-strip.tsx — reading tabs for the active context. Tabs are NOT routes:
// closing a tab does not pop history, and Back does not close a tab.
//
// Tab labels: for documents whose basename collides with another open tab
// (e.g. many README.md), the tab shows "folder/README.md" so they can be told
// apart. Otherwise just the title. Overflow scrolls with prev/next controls.

import { useEffect, useRef, useState } from "react"
import { useLocation, useNavigate } from "react-router-dom"
import { X, ChevronLeft, ChevronRight } from "lucide-react"
import { reading } from "@/lib/wails"
import { registerAction } from "@/lib/shortcuts"
import { cn } from "@/lib/utils"

interface Tab {
  documentId: number
  title?: string
  relPath?: string
  pane: number
  isActive?: boolean
}

export function TabStrip() {
  const navigate = useNavigate()
  const location = useLocation()
  const [tabs, setTabs] = useState<Tab[]>([])
  const tabsRef = useRef<Tab[]>([])
  const stripRef = useRef<HTMLDivElement>(null)
  const [canScrollLeft, setCanScrollLeft] = useState(false)
  const [canScrollRight, setCanScrollRight] = useState(false)

  // Keep a live ref so persist always sees the current list (no stale closure).
  useEffect(() => {
    tabsRef.current = tabs
  }, [tabs])

  // Update overflow affordances. Runs on tab change (after paint) and on a
  // short poll so the buttons enable the moment content overflows — a
  // ResizeObserver on the strip alone misses this because the strip's own box
  // does not change when tabs are added inside an overflow container.
  useEffect(() => {
    if (tabs.length === 0) return
    updateScrollButtons()
    const raf = window.requestAnimationFrame(() => {
      updateScrollButtons()
      const activeEl = stripRef.current?.querySelector('[data-active="true"]')
      activeEl?.scrollIntoView({ block: "nearest", inline: "nearest" })
      window.setTimeout(updateScrollButtons, 60)
    })
    return () => window.cancelAnimationFrame(raf)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tabs])

  // Poll overflow while tabs exist so the buttons enable reliably even when
  // tab count/widths change without resizing the strip's own box.
  useEffect(() => {
    if (tabs.length === 0) return
    updateScrollButtons()
    const iv = window.setInterval(updateScrollButtons, 400)
    return () => window.clearInterval(iv)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tabs.length > 0])

  // Observe window size too (outline pane show/hide resizes the strip's box).
  useEffect(() => {
    const onResize = () => updateScrollButtons()
    window.addEventListener("resize", onResize)
    return () => window.removeEventListener("resize", onResize)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function updateScrollButtons() {
    const strip = stripRef.current
    if (!strip) return
    setCanScrollLeft(strip.scrollLeft > 2)
    setCanScrollRight(strip.scrollLeft + strip.clientWidth < strip.scrollWidth - 2)
  }

  function scrollBy(dx: number) {
    const strip = stripRef.current
    if (!strip) return
    strip.scrollBy({ left: dx, behavior: "smooth" })
    window.setTimeout(updateScrollButtons, 200)
  }

  function persist() {
    // Tabs are global (not per source/collection) so they persist across
    // navigations. Always stored under the library context.
    const snapshot = tabsRef.current.map((t) => ({ documentId: t.documentId, pane: t.pane, isActive: !!t.isActive }))
    void reading.saveTabs("library", 0, snapshot)
  }

  // Load tabs — global strip, independent of the current route context.
  useEffect(() => {
    void reading.getTabs("library", 0).then(([data]) => {
      if (data) {
        const loaded = data.map((t: any) => ({ documentId: t.documentId, pane: t.pane, isActive: t.isActive, title: t.title, relPath: t.relPath }))
        setTabs(loaded)
      }
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.pathname])

  function addTab(docId: number, pane: number, title?: string, relPath?: string) {
    setTabs((prev) => {
      const next = prev.map((t) => (t.pane === pane ? { ...t, isActive: false } : t))
      const existing = next.find((t) => t.documentId === docId && t.pane === pane)
      if (existing) {
        return next.map((t) => (t === existing ? { ...t, isActive: true, title: title || t.title, relPath: relPath || t.relPath } : t))
      }
      return [...next, { documentId: docId, pane, isActive: true, title, relPath }]
    })
    // persist after state settles so the newly added tab is included
    requestAnimationFrame(() => persist())
  }

  function closeTabDirect(tab: Tab) {
    setTabs((prev) => prev.filter((t) => !(t.documentId === tab.documentId && t.pane === tab.pane)))
    requestAnimationFrame(() => persist())
  }

  // Listen for documents opened from the tree / reader.
  useEffect(() => {
    const onOpen = (e: Event) => {
      const detail = (e as CustomEvent).detail
      if (!detail) return
      addTab(detail.documentId, detail.pane || 0, detail.title, detail.relPath)
    }
    window.addEventListener("markdownia:doc-opened", onOpen)

    const unregClose = registerAction("tab-close", () => {
      const active = tabsRef.current.find((t) => t.isActive)
      if (active) closeTabDirect(active)
    })
    const unregCloseAll = registerAction("close-all-tabs", () => {
      setTabs([])
      requestAnimationFrame(() => persist())
    })

    return () => {
      window.removeEventListener("markdownia:doc-opened", onOpen)
      unregClose?.()
      unregCloseAll?.()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function openTab(tab: Tab) {
    if (tab.pane === 1) navigate(`/doc/${tab.documentId}/split/${tab.documentId}`)
    else navigate(`/doc/${tab.documentId}`)
  }

  function closeTab(tab: Tab, e: React.MouseEvent) {
    e.stopPropagation()
    closeTabDirect(tab)
  }

  if (tabs.length === 0) return null

  // Compute a label per tab: if the basename collides with another open tab,
  // prefix the parent folder so "README.md" from different folders is distinct.
  function labelFor(tab: Tab, all: Tab[]): string {
    const title = tab.title || "Document"
    const base = tab.relPath ? tab.relPath.split("/").pop() : ""
    if (!base) return title
    const collisions = all.filter((t) => t.documentId !== tab.documentId)
      .some((t) => (t.relPath ? t.relPath.split("/").pop() : "") === base)
    if (!collisions) return title
    const parts = tab.relPath!.split("/")
    return parts.length > 1 ? `${parts[parts.length - 2]}/${base}` : base
  }

  return (
    <div className="flex w-full min-w-0 items-center gap-1">
      <button
        onClick={() => scrollBy(-160)}
        disabled={!canScrollLeft}
        className="flex size-5 shrink-0 items-center justify-center rounded text-muted-foreground hover:bg-muted disabled:pointer-events-none disabled:opacity-30"
        aria-label="Scroll tabs left"
      >
        <ChevronLeft className="size-3.5" />
      </button>
      <div
        ref={stripRef}
        onScroll={updateScrollButtons}
        className="flex min-w-0 flex-1 items-center gap-0.5 overflow-x-auto scrollbar-none"
      >
        {tabs.map((tab) => (
          <button
            key={`${tab.documentId}-${tab.pane}`}
            data-active={tab.isActive ? "true" : undefined}
            onClick={() => openTab(tab)}
            className={cn(
              "group flex h-7 max-w-52 min-w-24 shrink-0 items-center gap-1 rounded-md border px-2 text-xs",
              tab.isActive
                ? "border-primary/40 bg-primary/10 text-primary"
                : "border-transparent text-muted-foreground hover:bg-muted hover:text-foreground"
            )}
          >
            <span className="min-w-0 truncate">{labelFor(tab, tabsRef.current)}</span>
            <span
              role="button"
              className="flex size-4 items-center justify-center rounded-sm text-muted-foreground opacity-0 group-hover:opacity-100 hover:bg-muted-foreground/20"
              onClick={(e) => closeTab(tab, e)}
            >
              <X className="size-3" />
            </span>
          </button>
        ))}
      </div>
      <button
        onClick={() => scrollBy(160)}
        disabled={!canScrollRight}
        className="flex size-5 shrink-0 items-center justify-center rounded text-muted-foreground hover:bg-muted disabled:pointer-events-none disabled:opacity-30"
        aria-label="Scroll tabs right"
      >
        <ChevronRight className="size-3.5" />
      </button>
    </div>
  )
}
