// reader.tsx — the product screen. Single OpenDocument call → HTML + outline +
// highlights. Toolbar, reading zoom, link interception.

import { useEffect, useState } from "react"
import { useParams } from "react-router-dom"
import { ReadingSurface } from "@/components/reading-surface"
import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Bookmark, FolderPlus, Download, MoreHorizontal, Search, FolderOpen, Copy } from "lucide-react"
import { annotations, library, native } from "@/lib/wails"
import { toast } from "@/lib/toast"
import { dispatchAction } from "@/lib/shortcuts"

export function Reader({ split = false }: { split?: boolean }) {
  const params = useParams()
  const docId = split ? Number(params.leftId) : Number(params.documentId)
  const rightId = split ? Number(params.rightId) : docId
  const [title, setTitle] = useState("Loading…")
  const [bookmarked, setBookmarked] = useState(false)
  const ctx = parseCtx(window.location.search)

  useEffect(() => {
    const onOpen = (e: Event) => {
      const detail = (e as CustomEvent).detail
      if (detail && detail.documentId === docId) setTitle(detail.title || "Document")
    }
    window.addEventListener("markdownia:doc-opened", onOpen)
    return () => window.removeEventListener("markdownia:doc-opened", onOpen)
  }, [docId])

  // Load bookmark state for this document.
  useEffect(() => {
    let cancelled = false
    void annotations.listBookmarks().then(([bms]) => {
      if (cancelled) return
      setBookmarked(!!(bms || []).find((b: any) => b.documentId === docId))
    })
    return () => {
      cancelled = true
    }
  }, [docId])

  function toggleBookmark() {
    if (bookmarked) {
      void annotations.listBookmarks().then(([bms]) => {
        const bm = (bms || []).find((b: any) => b.documentId === docId)
        if (!bm) {
          setBookmarked(false)
          return
        }
        void annotations.removeBookmark(bm.id).then(([, err]) => {
          if (err) toast({ type: "error", title: "Remove bookmark failed", caption: err.content })
          else {
            setBookmarked(false)
            toast({ type: "success", title: "Bookmark removed" })
          }
        })
      })
      return
    }
    void annotations.addBookmark(docId, "", "").then(([, err]) => {
      if (err) toast({ type: "error", title: "Bookmark failed", caption: err.content })
      else {
        setBookmarked(true)
        toast({ type: "success", title: "Bookmarked" })
      }
    })
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex h-11 shrink-0 items-center justify-between border-b px-3">
        <div className="truncate text-sm font-medium">{title}</div>
        <div className="flex items-center gap-1">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant={bookmarked ? "secondary" : "ghost"}
                size="icon"
                onClick={toggleBookmark}
                className={bookmarked ? "text-primary" : ""}
              >
                <Bookmark className={bookmarked ? "size-4 fill-current" : "size-4"} />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{bookmarked ? "Remove bookmark" : "Bookmark"}</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" onClick={() => dispatchAction("open-find-bar")}><Search className="size-4" /></Button>
            </TooltipTrigger>
            <TooltipContent>Find in document</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" onClick={() => dispatchAction("add-to-collection", docId)}><FolderPlus className="size-4" /></Button>
            </TooltipTrigger>
            <TooltipContent>Add to collection</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" onClick={() => dispatchAction("open-export", "pdf")}><Download className="size-4" /></Button>
            </TooltipTrigger>
            <TooltipContent>Export</TooltipContent>
          </Tooltip>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon"><MoreHorizontal className="size-4" /></Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem
                onClick={() =>
                  void library.meta(docId).then(([m]) => {
                    if (m?.relPath) void native.reveal(m.relPath)
                  })
                }
              >
                <FolderOpen /> Reveal in Finder
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() =>
                  void library.meta(docId).then(([m]) => navigator.clipboard.writeText(m?.relPath || ""))
                }
              >
                <Copy /> Copy path
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
      <div className="flex min-h-0 flex-1">
        <div className="min-w-0 flex-1 overflow-hidden">
          <ReadingSurface docId={docId} contextType={ctx.type} contextId={ctx.id} />
        </div>
        {split && (
          <>
            <div className="w-1 shrink-0 cursor-col-resize bg-border" />
            <div className="min-w-0 flex-1 overflow-hidden">
              <ReadingSurface docId={rightId} contextType={ctx.type} contextId={ctx.id} />
            </div>
          </>
        )}
      </div>
    </div>
  )
}

function parseCtx(search: string) {
  const m = search.match(/[?&]ctx=(source|collection):(\d+)/)
  return m ? { type: m[1], id: Number(m[2]) } : { type: "library", id: 0 }
}
