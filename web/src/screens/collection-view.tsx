// collection-view.tsx — rows show their source breadcrumb; remove is [x];
// manual ordering; empty state states the model outright.

import { useEffect, useState } from "react"
import { useNavigate, useParams } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { DocumentRow, EmptyState } from "@/components/rows"
import { collections, sources, library } from "@/lib/wails"
import { toast } from "@/lib/toast"
import { Checkbox } from "@/components/ui/checkbox"

export function CollectionView() {
  const { collectionId } = useParams()
  const id = Number(collectionId)
  const navigate = useNavigate()
  const [col, setCol] = useState<any>(null)
  const [rows, setRows] = useState<any[]>([])
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [addOpen, setAddOpen] = useState(false)

  async function load() {
    const [cols] = await collections.list()
    setCol((cols || []).find((c: any) => c.id === id) || null)
    const [r] = await collections.listDocuments(id)
    setRows(r || [])
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  if (!col) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <EmptyState icon="🗂" headline="Collection not found" body="It may have been deleted." action={{ label: "Library Home", onClick: () => navigate("/") }} />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6 px-6 py-8">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold tracking-tight">{col.name}</h1>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setAddOpen(true)}>Add documents…</Button>
          <Button variant="outline">Manual</Button>
          <Button variant="destructive" onClick={() => setDeleteOpen(true)}>Delete collection</Button>
        </div>
      </div>

      <div className="space-y-1">
        {rows.length === 0 ? (
          <EmptyState
            icon="🗂"
            headline="This collection is empty"
            body="Adding a document never moves or copies the file — it only lists it here."
            action={{ label: "Add documents…", onClick: () => setAddOpen(true) }}
          />
        ) : (
          rows.map((r) => (
            <DocumentRow
              key={r.documentId}
              title={r.title}
              relPath={r.relPath}
              sourceName={r.sourceName}
              onClick={() => navigate(`/doc/${r.documentId}?ctx=collection:${id}`)}
              meta={
                <button
                  className="text-muted-foreground hover:text-foreground"
                  title="Remove from this collection"
                  onClick={(e) => {
                    e.stopPropagation()
                    void collections.removeDocuments(id, [r.documentId]).then(() => void load())
                  }}
                >
                  ✕
                </button>
              }
            />
          ))
        )}
      </div>

      <DeleteCollectionDialog id={id} name={col.name} open={deleteOpen} onOpenChange={setDeleteOpen} onDeleted={() => navigate("/")} />
      <AddDocsDialog id={id} open={addOpen} onOpenChange={setAddOpen} onDone={() => void load()} />
    </div>
  )
}

function DeleteCollectionDialog({ id, name, open, onOpenChange, onDeleted }: { id: number; name: string; open: boolean; onOpenChange: (o: boolean) => void; onDeleted: () => void }) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete collection?</AlertDialogTitle>
          <AlertDialogDescription>
            No documents and no files are affected — only this list is removed. "{name}" and its membership will be deleted.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction className="bg-destructive text-white hover:bg-destructive/90" onClick={() => {
            void collections.del(id).then(() => {
              window.dispatchEvent(new CustomEvent("markdownia:collections-changed"))
              onDeleted()
            })
          }}>
            Delete Collection
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

function AddDocsDialog({ id, open, onOpenChange, onDone }: { id: number; open: boolean; onOpenChange: (o: boolean) => void; onDone: () => void }) {
  const [docs, setDocs] = useState<any[]>([])
  const [existing, setExisting] = useState<Set<number>>(new Set())
  const [filter, setFilter] = useState("")
  const [selected, setSelected] = useState<Set<number>>(new Set())

  useEffect(() => {
    if (!open) return
    setSelected(new Set())
    void collections.listDocuments(id).then(([r]) => setExisting(new Set((r || []).map((x: any) => x.documentId))))
    void sources.list().then(([srcs]) => {
      const jobs = (srcs || []).map((s: any) => library.tree(s.id))
      void Promise.all(jobs).then((trees) => {
        setDocs(trees.flat().filter((n: any) => !n.isFolder))
      })
    })
  }, [open, id])

  const f = filter.toLowerCase()
  const visible = docs.filter((d) => !f || d.title.toLowerCase().includes(f) || d.relPath.toLowerCase().includes(f))

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Add documents</DialogTitle>
          <DialogDescription>Select documents to add. Already-present rows show [-] and are not selectable.</DialogDescription>
        </DialogHeader>
        <Input value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="Filter…" />
        <div className="flex max-h-72 flex-col gap-0.5 overflow-auto">
          {visible.map((d) => {
            const present = existing.has(d.id)
            const isSel = selected.has(d.id)
            return (
              <label
                key={d.id}
                className={`flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-accent ${present ? "opacity-50" : ""}`}
              >
                <Checkbox
                  checked={isSel}
                  disabled={present}
                  onCheckedChange={(v) => {
                    setSelected((prev) => {
                      const next = new Set(prev)
                      if (v) next.add(d.id)
                      else next.delete(d.id)
                      return next
                    })
                  }}
                />
                <span className="truncate">{present ? `${d.title} [-]` : d.title}</span>
              </label>
            )
          })}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button
            disabled={!selected.size}
            onClick={() => {
              void collections.addDocuments(id, [...selected]).then(() => {
                onOpenChange(false)
                onDone()
              })
            }}
          >
            Add selected ({selected.size})
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
