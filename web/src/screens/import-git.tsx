// import-git.tsx — dialog. URL field accepting HTTPS, SSH, owner/repo
// shorthand; pasted /tree/<branch> extracts the branch. Dialog stays open
// during cloning.

import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Progress } from "@/components/ui/progress"
import { sources } from "@/lib/wails"
import { toast } from "@/lib/toast"

let openFn: ((open: boolean) => void) | null = null
let state: { open: boolean } = { open: false }

export function registerImportGitHost(fn: (open: boolean) => void) {
  openFn = fn
}

export function openImportGit() {
  state = { open: true }
  openFn?.(true)
}

export function ImportGitHost() {
  const [open, setOpen] = useState(false)
  useEffect(() => {
    registerImportGitHost((o) => setOpen(o))
  }, [])
  if (!open) return null
  return <ImportGitDialog open={open} onOpenChange={(o) => { setOpen(o); state.open = o }} />
}

function ImportGitDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (o: boolean) => void }) {
  const navigate = useNavigate()
  const [url, setUrl] = useState("")
  const [branch, setBranch] = useState("")
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const valid = /^(https?:\/\/|git@|[^/\s]+\/[^/\s]+)/.test(url.trim())

  useEffect(() => {
    const m = url.match(/\/tree\/([^/]+)/)
    if (m && !branch) setBranch(m[1])
  }, [url, branch])

  function start() {
    if (!url.trim() || busy) return
    setBusy(true)
    setError(null)
    void sources.importGit(url.trim(), branch.trim()).then(([id, err]) => {
      if (err) {
        setBusy(false)
        setError(err.content || "Clone failed")
        return
      }
      toast({ type: "success", title: "Clone started" })
      onOpenChange(false)
      navigate(`/source/${id}`)
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Import Git Repository</DialogTitle>
          <DialogDescription>This is the only step that uses the network.</DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Repository URL</label>
            <Input type="url" placeholder="https://github.com/owner/repo or owner/repo" value={url} onChange={(e) => setUrl(e.target.value)} autoFocus />
          </div>
          <div className="space-y-1.5">
            <label className="text-xs font-medium text-muted-foreground">Branch (optional)</label>
            <Input type="text" placeholder="main" value={branch} onChange={(e) => setBranch(e.target.value)} />
          </div>
          {busy && <Progress value={0} />}
          {error && (
            <div className="rounded border border-destructive/40 px-3 py-2 text-sm text-destructive">{error}</div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button disabled={!valid || busy} onClick={start}>{busy ? "Cloning…" : "Clone"}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
