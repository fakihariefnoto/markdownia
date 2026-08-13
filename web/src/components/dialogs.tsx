// dialogs.tsx — shared modal dialogs built on shadcn primitives.

import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { collections } from "@/lib/wails"
import { toast } from "@/lib/toast"
import { dispatchAction } from "@/lib/shortcuts"

// A small imperative dialog store so action handlers can open dialogs.
type DialogSpec =
  | { kind: "new-collection"; open: boolean }
  | { kind: "add-to-collection"; open: boolean; docId?: number }

let setDialogFn: ((spec: DialogSpec) => void) | null = null
let currentSpec: DialogSpec = { kind: "new-collection", open: false }

export function registerDialogHost(fn: (spec: DialogSpec) => void) {
  setDialogFn = fn
}

export function openDialog(spec: DialogSpec) {
  currentSpec = spec
  setDialogFn?.({ ...spec, open: true })
}

export function DialogHost() {
  const [spec, setSpec] = useState<DialogSpec>({ kind: "new-collection", open: false })

  useEffect(() => {
    registerDialogHost((s) => setSpec(s))
  }, [])

  const close = () => setSpec((s) => ({ ...s, open: false }))

  if (spec.kind === "new-collection") {
    return <NewCollectionDialog open={spec.open} onOpenChange={close} />
  }
  return <AddToCollectionDialog open={spec.open} docId={spec.docId} onOpenChange={close} />
}

function NewCollectionDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const [name, setName] = useState("")
  function submit() {
    void collections.create(name, "").then(([, err]) => {
      if (err) toast({ type: "error", title: "Could not create collection", caption: err.content })
      else {
        setName("")
        onOpenChange(false)
        window.dispatchEvent(new CustomEvent("markdownia:collections-changed"))
      }
    })
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New Collection</DialogTitle>
          <DialogDescription>Collections are curated reading lists that can hold documents from any source.</DialogDescription>
        </DialogHeader>
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Collection name"
          autoFocus
          onKeyDown={(e) => {
            if (e.key === "Enter") submit()
          }}
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={submit}>
            Create
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function AddToCollectionDialog({ open, docId, onOpenChange }: { open: boolean; docId?: number; onOpenChange: (o: boolean) => void }) {
  const [cols, setCols] = useState<any[]>([])
  const [checked, setChecked] = useState<number[]>([])

  useEffect(() => {
    if (open) {
      void collections.list().then(([data]) => setCols(data || []))
    }
  }, [open])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add to collection</DialogTitle>
        </DialogHeader>
        <div className="flex max-h-72 flex-col gap-1 overflow-auto">
          {cols.length === 0 && <p className="text-sm text-muted-foreground">No collections yet.</p>}
          {cols.map((c) => (
            <label key={c.id} className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-accent">
              <Checkbox
                checked={checked.includes(c.id)}
                onCheckedChange={(v) => {
                  setChecked((prev) =>
                    v ? [...prev, c.id] : prev.filter((x) => x !== c.id)
                  )
                }}
              />
              <span className="truncate">{c.name}</span>
              <span className="ml-auto text-xs text-muted-foreground">{c.documentCount}</span>
            </label>
          ))}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button
            disabled={!checked.length}
            onClick={() => {
              if (docId != null && checked.length) {
                void collections.addDocuments(checked[0], [docId]).then(() => {
                  onOpenChange(false)
                })
              }
            }}
          >
            Add
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export { dispatchAction }
