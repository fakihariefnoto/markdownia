// reading-surface.tsx — the reading pane. Injects cached HTML, applies the
// measure cap, owns scroll tracking, highlights, find, code blocks, mermaid.
// Ported from the vanilla reading-surface.js.

import { useEffect, useRef, useState } from "react"
import { useNavigate } from "react-router-dom"
import { library, annotations, reading, native } from "@/lib/wails"
import { dispatchAction, registerAction } from "@/lib/shortcuts"
import { toast } from "@/lib/toast"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { ArrowDown, ArrowUp, X } from "lucide-react"

export function ReadingSurface({ docId, contextType, contextId }: { docId: number; contextType?: string; contextId?: number }) {
  const navigate = useNavigate()
  const elRef = useRef<HTMLDivElement>(null)
  const docRef = useRef<HTMLDivElement>(null)
  const highlightsRef = useRef<any[]>([])
  const saveTimer = useRef<number | null>(null)
  const [findOpen, setFindOpen] = useState(false)
  const [findQuery, setFindQuery] = useState("")
  const [findCount, setFindCount] = useState("")
  const matchesRef = useRef<Array<{ node: Node; idx: number }>>([])
  const currentRef = useRef(-1)

  useEffect(() => {
    const el = elRef.current!
    const doc = docRef.current!
    let cancelled = false

    async function open() {
      const [data, err] = await library.open(docId, contextType || "library", contextId || 0)
      if (cancelled) return
      if (err) {
        renderError(doc, err)
        return
      }
      doc.innerHTML = data.renderedHtml
      highlightsRef.current = data.highlights || []
      applyMeasure(doc)
      renderHighlights(doc)
      attachBehavior(doc)
      window.dispatchEvent(new CustomEvent("markdownia:doc-opened", {
        detail: { documentId: docId, title: data.title, relPath: data.relPath, pane: 0 },
      }))
      if (data.missingFile) {
        const banner = document.createElement("div")
        banner.className = "missing-banner"
        banner.textContent = "This file no longer exists in the source. Cached content is shown; annotation is disabled."
        el.prepend(banner)
      }
      restoreScroll(el)
    }

    void open()
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [docId, contextType, contextId])

  useEffect(() => {
    const el = elRef.current!
    const handler = () => {
      if (saveTimer.current) window.clearTimeout(saveTimer.current)
      saveTimer.current = window.setTimeout(() => {
        const pct = el.scrollHeight ? el.scrollTop / el.scrollHeight : 0
        void reading.saveScroll(contextType || "library", contextId || 0, docId, pct)
      }, 500)
    }
    el.addEventListener("scroll", handler)
    return () => el.removeEventListener("scroll", handler)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [docId, contextType, contextId])

  // Find bar registration (per reading surface instance).
  useEffect(() => {
    const unregOpen = registerAction("open-find-bar", () => {
      setFindOpen(true)
    })
    const unregNext = registerAction("find-next", () => cycle(1))
    const unregPrev = registerAction("find-previous", () => cycle(-1))
    return () => {
      unregOpen?.()
      unregNext?.()
      unregPrev?.()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [docId])

  function runFind(query: string) {
    const doc = docRef.current
    if (!doc) return
    matchesRef.current = []
    currentRef.current = -1
    if (!query) {
      setFindCount("")
      return
    }
    const walker = document.createTreeWalker(doc, NodeFilter.SHOW_TEXT)
    let node: Node | null
    while ((node = walker.nextNode())) {
      const idx = node.nodeValue!.toLowerCase().indexOf(query.toLowerCase())
      if (idx >= 0) matchesRef.current.push({ node, idx })
    }
    setFindCount(matchesRef.current.length ? `1 of ${matchesRef.current.length}` : "No matches")
    if (matchesRef.current.length) cycle(0)
  }

  function cycle(dir: number) {
    const doc = docRef.current
    const input = findQuery
    if (!doc || !matchesRef.current.length) return
    currentRef.current = (currentRef.current + dir + matchesRef.current.length) % matchesRef.current.length
    const { node, idx } = matchesRef.current[currentRef.current]
    const range = document.createRange()
    range.setStart(node, idx)
    range.setEnd(node, idx + (input?.length || 0))
    const sel = window.getSelection()
    sel?.removeAllRanges()
    sel?.addRange(range)
    ;(node.parentElement as HTMLElement)?.scrollIntoView({ block: "center" })
    setFindCount(`${currentRef.current + 1} of ${matchesRef.current.length}`)
  }

  function closeFind() {
    setFindOpen(false)
    setFindQuery("")
    setFindCount("")
    window.getSelection()?.removeAllRanges()
  }

  function applyMeasure(doc: HTMLElement) {
    const measure = document.documentElement.dataset.readingMeasure || "72ch"
    doc.dataset.measure = measure
  }

  function renderHighlights(doc: HTMLElement) {
    doc.querySelectorAll("mark.hl").forEach((m) => {
      const parent = m.parentElement
      m.replaceWith(document.createTextNode(m.textContent))
      parent?.normalize()
    })
    if (!highlightsRef.current.length) return
    const blocks = doc.querySelectorAll("[data-block-hash]")
    highlightsRef.current.forEach((h) => {
      blocks.forEach((b) => {
        const el = b as HTMLElement
        if (el.dataset.blockHash === h.blockHash && Number(el.dataset.blockIndex) === h.blockIndex) {
          paintOnBlock(el, h)
        }
      })
    })
  }

  function paintOnBlock(block: HTMLElement, h: any) {
    const walker = document.createTreeWalker(block, NodeFilter.SHOW_TEXT)
    let offset = 0
    let node: Node | null
    while ((node = walker.nextNode())) {
      const len = node.nodeValue!.length
      const start = h.startOffset - offset
      const end = h.endOffset - offset
      if (start < len && end > 0) {
        const s = Math.max(0, start)
        const e = Math.min(len, end)
        const range = document.createRange()
        range.setStart(node, s)
        range.setEnd(node, e)
        const mark = document.createElement("mark")
        mark.className = `hl ${h.color}`
        mark.dataset.highlightId = h.id
        mark.title = h.note || "Highlight"
        range.surroundContents(mark)
        mark.addEventListener("click", () => dispatchAction("highlight-clicked", h))
      }
      offset += len
    }
  }

  function attachBehavior(doc: HTMLElement) {
    doc.querySelectorAll("a[href]").forEach((a) => {
      a.addEventListener("click", (e) => {
        e.preventDefault()
        const href = a.getAttribute("href")!
        void library.resolveLink(docId, href).then(([target]) => {
          if (!target) {
            dispatchAction("link-not-in-library", href)
          } else if (target.folder) {
            // Reveal the folder in the source tree (expand + highlight).
            window.dispatchEvent(new CustomEvent("markdownia:reveal-folder", {
              detail: { sourceId: target.sourceId, folderRel: target.folderRel },
            }))
          } else if (target.internal) {
            if (target.anchor) navigate(`/doc/${target.documentId}#${target.anchor}`)
            else navigate(`/doc/${target.documentId}`)
          } else if (target.external) {
            void native.openExternal(target.url)
          }
        })
      })
    })

    doc.querySelectorAll('input[type="checkbox"]').forEach((cb) => {
      ;(cb as HTMLInputElement).disabled = true
    })

    doc.querySelectorAll("pre code.language-mermaid, .mermaid-source").forEach(renderMermaid)
    doc.querySelectorAll('code[class*="language-"]').forEach(attachCodeBlock)

    doc.querySelectorAll("img").forEach((img) => {
      if ((img as HTMLImageElement).complete && (img as HTMLImageElement).naturalWidth === 0) {
        replaceBrokenImage(img as HTMLImageElement)
      } else {
        img.addEventListener("error", () => replaceBrokenImage(img as HTMLImageElement))
      }
    })

    doc.querySelectorAll("h1,h2,h3,h4,h5,h6").forEach((h) => {
      const id = h.getAttribute("id")
      if (!id) return
      const a = document.createElement("a")
      a.className = "heading-anchor"
      a.href = "#" + id
      a.textContent = "#"
      h.appendChild(a)
    })
  }

  function renderMermaid(node: Element) {
    const source = node.textContent
    const placeholder = document.createElement("div")
    placeholder.className = "mermaid"
    placeholder.style.minHeight = "120px"
    placeholder.textContent = "Loading diagram…"
    node.parentElement!.replaceWith(placeholder)

    const io = new IntersectionObserver((entries) => {
      if (!entries[0].isIntersecting) return
      io.disconnect()
      void import("mermaid").then(({ default: mermaid }) => {
        mermaid.initialize({ startOnLoad: false, theme: "base" })
        mermaid.render(`mermaid-${Math.random().toString(36).slice(2)}`, source)
          .then(({ svg }) => { placeholder.innerHTML = svg })
          .catch(() => {
            placeholder.innerHTML = `<div class="mermaid-fallback">Diagram failed to render</div><pre><code>${esc(source)}</code></pre>`
          })
      })
    }, { rootMargin: "200px" })
    io.observe(placeholder)
  }

  function attachCodeBlock(node: Element) {
    const pre = node.closest("pre")
    if (!pre) return
    const lang = (node.className || "").replace("language-", "")
    if (lang === "mermaid") return
    const label = document.createElement("div")
    label.className = "lang-label"
    label.textContent = lang || ""
    pre.appendChild(label)

    const copy = document.createElement("button")
    copy.className = "copy-btn ghost-btn"
    copy.textContent = "Copy"
    pre.appendChild(copy)
    copy.addEventListener("click", () => {
      void navigator.clipboard.writeText(node.textContent || "").then(() => {
        copy.textContent = "✓"
        setTimeout(() => { copy.textContent = "Copy" }, 1500)
      })
    })
  }

  function replaceBrokenImage(img: HTMLImageElement) {
    const ph = document.createElement("div")
    ph.className = "img-placeholder"
    ph.textContent = `Image not found: ${img.getAttribute("src") || ""}`
    img.replaceWith(ph)
  }

  function restoreScroll(el: HTMLElement) {
    void reading.getState(contextType || "library", contextId || 0).then(([st]) => {
      if (st && st.scrollPct > 0) {
        el.scrollTop = st.scrollPct * el.scrollHeight
      }
    })
  }

  function renderError(doc: HTMLElement, err: any) {
    doc.innerHTML = `
      <div class="read-error">
        <h2>Couldn't open this document</h2>
        <p>${esc(err?.content || err?.message || "Unknown error")}</p>
        <button class="btn-secondary" data-reindex>Re-index source</button>
      </div>
    `
    doc.querySelector("[data-reindex]")?.addEventListener("click", () => {
      dispatchAction("source-refresh")
    })
  }

  return (
    <div
      ref={elRef}
      className="reading-surface relative h-full overflow-y-auto"
      data-reading-font={document.documentElement.dataset.readingFont || "sans"}
    >
      {findOpen && (
        <div
          className="sticky top-0 z-10 flex items-center gap-2 border-b px-3 py-1.5"
          style={{ borderColor: "var(--read-rule)", background: "var(--read-background)" }}
        >
          <Input
            autoFocus
            value={findQuery}
            onChange={(e) => {
              setFindQuery(e.target.value)
              runFind(e.target.value)
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault()
                cycle(e.shiftKey ? -1 : 1)
              }
              if (e.key === "Escape") closeFind()
            }}
            placeholder="Find in document"
            className="h-7 w-48"
            style={{ borderColor: "var(--read-rule)", background: "var(--read-background)", color: "var(--read-text)" }}
          />
          <span className="text-xs" style={{ color: "var(--read-muted)" }}>{findCount}</span>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => cycle(-1)}><ArrowUp className="size-3.5" /></Button>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => cycle(1)}><ArrowDown className="size-3.5" /></Button>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={closeFind}><X className="size-3.5" /></Button>
        </div>
      )}
      <div ref={docRef} className="doc" data-measure={document.documentElement.dataset.readingMeasure || "72ch"} />
    </div>
  )
}

function esc(s: unknown) {
  return String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c] as string))
}
