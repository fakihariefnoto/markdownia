// source-overview.tsx — status-driven rendering for all seven statuses,
// delete preview, unavailable/error states.

import { useEffect, useState } from "react"
import { useNavigate, useParams } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { sources, native } from "@/lib/wails"
import { toast } from "@/lib/toast"
import { useEvent } from "@/lib/events"

export function SourceOverview() {
  const { sourceId } = useParams()
  const id = Number(sourceId)
  const navigate = useNavigate()
  const [src, setSrc] = useState<any>(null)
  const [progress, setProgress] = useState<{ current?: number; total?: number } | null>(null)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [renameOpen, setRenameOpen] = useState(false)
  const [relocateOpen, setRelocateOpen] = useState(false)

  async function load() {
    const [list] = await sources.list()
    setSrc((list || []).find((s: any) => s.id === id) || null)
  }

  useEffect(() => {
    void load()
    const onAction = (e: Event) => {
      const action = (e as CustomEvent).detail?.action
      if (action === "delete") setDeleteOpen(true)
      else if (action === "rename") setRenameOpen(true)
      else if (action === "relocate") setRelocateOpen(true)
      else if (action === "reveal") {
        if (src?.rootPath) void native.reveal(src.rootPath)
      }
    }
    window.addEventListener("markdownia:source-action", onAction)
    return () => window.removeEventListener("markdownia:source-action", onAction)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, src])

  useEvent("source:progress", (p) => {
    if (p.sourceId === id) setProgress({ current: p.current, total: p.total })
  })
  useEvent("source:status", (p) => {
    if (p.sourceId === id) {
      if (p.status === "error") toast({ type: "error", title: "Source error", caption: p.error, sticky: true })
      void load()
    }
  })
  useEvent("source:indexed", (p) => {
    if (p.sourceId === id) {
      if (p.indexed > 0) toast({ type: "success", title: `Indexed ${p.indexed} documents` })
      void load()
    }
  })

  if (!src) {
    return (
      <div className="flex h-full items-center justify-center p-6 text-sm text-muted-foreground">
        Loading source…
      </div>
    )
  }

  const isBusy = ["cloning", "extracting", "indexing"].includes(src.status)

  return (
    <div className="mx-auto max-w-3xl space-y-6 px-6 py-8">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{src.name}</h1>
          <Badge variant={src.status === "error" ? "destructive" : src.status === "ready" ? "secondary" : "outline"} className="mt-1">
            {src.status}
          </Badge>
        </div>
      </header>

      <dl className="grid grid-cols-1 gap-2 text-sm sm:grid-cols-2">
        <Meta k="Kind" v={src.kind} />
        <Meta k="Path" v={src.rootPath} />
        <Meta k="Origin" v={src.originUrl || "—"} />
        <Meta k="Branch" v={src.gitBranch || "—"} />
        <Meta k="Commit" v={(src.gitCommit || "—").slice(0, 8)} />
        <Meta k="Documents" v={String(src.documentCount)} />
        <Meta k="Indexed" v={src.indexedAt || "never"} />
        <Meta k="Managed" v={src.isManaged ? "yes (app-owned)" : "no (in place)"} />
      </dl>

      {isBusy && (
        <div className="space-y-2">
          <Progress value={progress?.total ? Math.round((progress.current || 0) / progress.total * 100) : 0} />
          <p className="text-xs text-muted-foreground">Working…</p>
        </div>
      )}

      {src.status === "error" && (
        <div className="rounded-lg border border-destructive/40 p-4 text-sm">
          <p className="font-medium">Indexing failed</p>
          <p className="text-muted-foreground">{src.errorMessage || "An error occurred"}</p>
        </div>
      )}

      <div className="flex flex-wrap gap-2">
        <Button variant="outline" disabled={src.kind === "zip"} onClick={() => void sources.refresh(id)}>
          Refresh
        </Button>
        <Button variant="outline" onClick={() => setRelocateOpen(true)}>Relocate…</Button>
        <Button variant="outline" onClick={() => setRenameOpen(true)}>Rename…</Button>
        <Button variant="destructive" onClick={() => setDeleteOpen(true)}>Delete…</Button>
      </div>

      <DeleteDialog id={id} open={deleteOpen} onOpenChange={setDeleteOpen} onDeleted={() => navigate("/")} />
      <RenameDialog id={id} name={src.name} open={renameOpen} onOpenChange={setRenameOpen} onDone={() => void load()} />
      <RelocateDialog id={id} open={relocateOpen} onOpenChange={setRelocateOpen} onDone={() => void load()} />
    </div>
  )
}

function Meta({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex justify-between gap-2 border-b py-1">
      <dt className="text-muted-foreground">{k}</dt>
      <dd className="truncate font-medium">{v}</dd>
    </div>
  )
}

function DeleteDialog({ id, open, onOpenChange, onDeleted }: { id: number; open: boolean; onOpenChange: (o: boolean) => void; onDeleted: () => void }) {
  const [preview, setPreview] = useState<any>(null)
  useEffect(() => {
    if (open) void sources.deletionPreview(id).then(([p]) => setPreview(p))
  }, [open, id])
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete source?</AlertDialogTitle>
          <AlertDialogDescription>
            {preview?.deletesFilesOnDisk ? (
              <>
                <div className="mb-2 rounded-md border px-3 py-2 text-xs"
                  style={{ borderColor: "var(--color-error)", background: "color-mix(in srgb, var(--color-error) 10%, transparent)", color: "var(--color-error)" }}>
                  ⚠️ This source is app-managed — its extracted files on disk WILL be deleted.
                </div>
                <span>
                  This removes {preview.documents} documents, {preview.highlights} highlights, and {preview.collectionEntries} collection entries, and deletes the extracted files.
                </span>
              </>
            ) : (
              <>
                <div className="mb-2 rounded-md border px-3 py-2 text-xs"
                  style={{ borderColor: "var(--color-success)", background: "color-mix(in srgb, var(--color-success) 10%, transparent)", color: "var(--color-success)" }}>
                  ✓ This removes the source from your library only. Your files on disk are NOT deleted.
                </div>
                <span>
                  This removes {preview?.documents} documents, {preview?.highlights} highlights, and {preview?.collectionEntries} collection entries from the library.
                </span>
              </>
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            className="bg-destructive text-white hover:bg-destructive/90"
            onClick={() => {
              void sources.del(id).then(([, err]) => {
                if (err) toast({ type: "error", title: "Delete failed", caption: err.content })
                else onDeleted()
              })
            }}
          >
            Delete Source
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

function RenameDialog({ id, name, open, onOpenChange, onDone }: { id: number; name: string; open: boolean; onOpenChange: (o: boolean) => void; onDone: () => void }) {
  const [val, setVal] = useState(name)
  useEffect(() => setVal(name), [name, open])
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Rename source</DialogTitle>
          <DialogDescription>Change the display name only — files on disk are untouched.</DialogDescription>
        </DialogHeader>
        <Input value={val} onChange={(e) => setVal(e.target.value)} autoFocus />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={() => { void sources.rename(id, val).then(() => { onOpenChange(false); onDone() }) }}>Rename</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function RelocateDialog({ id, open, onOpenChange, onDone }: { id: number; open: boolean; onOpenChange: (o: boolean) => void; onDone: () => void }) {
  function pick() {
    void native.pickFolder().then(([result, err]) => {
      if (err || !result) return
      const [path, ok] = result
      if (!ok || !path) return
      void sources.relocate(id, path).then(([, e2]) => {
        if (e2) toast({ type: "error", title: "Relocate failed", caption: e2.content })
        else { onOpenChange(false); onDone() }
      })
    })
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Relocate source</DialogTitle>
          <DialogDescription>Point this source at its new location. Documents, highlights, and bookmarks survive.</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={pick}>Choose Folder…</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
