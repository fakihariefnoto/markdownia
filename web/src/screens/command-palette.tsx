// command-palette.tsx — overlay. Not a route; Esc leaves the previous state
// untouched. Four modes by prefix: none (mixed), > commands, # headings,
// @ sources and collections.

import { useEffect, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Command, CommandDialog, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command"
import { library, sources, collections } from "@/lib/wails"
import { dispatchAction, registerAction } from "@/lib/shortcuts"

export function CommandPaletteOverlay() {
  const [open, setOpen] = useState(false)
  const navigate = useNavigate()

  useEffect(() => {
    const unregister = registerAction("open-command-palette-overlay", () => setOpen((o) => !o))
    return () => {
      unregister?.()
    }
  }, [])

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <Palette onClose={() => setOpen(false)} navigate={navigate} />
    </CommandDialog>
  )
}

function Palette({ onClose, navigate }: { onClose: () => void; navigate: (p: string) => void }) {
  const [q, setQ] = useState("")
  const [items, setItems] = useState<Array<{ label: string; path?: string; run: () => void }>>([])

  useEffect(() => {
    async function run() {
      if (q.startsWith(">")) {
        setItems(await commandItems(q.slice(1)))
      } else if (q.startsWith("#")) {
        setItems(await headingItems(q.slice(1)))
      } else if (q.startsWith("@")) {
        setItems(await sourceItems(q.slice(1)))
      } else if (!q) {
        setItems(await recentItems())
      } else {
        setItems(await docItems(q))
      }
    }
    void run()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q, navigate])

  async function commandItems(filter: string): Promise<Array<{ label: string; run: () => void }>> {
    const all = [
      { label: "Search Library", run: () => { navigate("/search"); onClose() } },
      { label: "Import Folder…", run: () => { dispatchAction("import-folder"); onClose() } },
      { label: "Import Git Repository…", run: () => { dispatchAction("import-git"); onClose() } },
      { label: "Import Zip…", run: () => { dispatchAction("import-zip"); onClose() } },
      { label: "New Collection…", run: () => { dispatchAction("new-collection"); onClose() } },
      { label: "Settings", run: () => { navigate("/settings"); onClose() } },
      { label: "Annotations", run: () => { navigate("/annotations"); onClose() } },
    ]
    if (!filter) return all
    return all.filter((c) => c.label.toLowerCase().includes(filter.toLowerCase()))
  }

  async function docItems(q: string) {
    const [recent] = await library.recent(6)
    const [srcs] = await sources.list()
    const items: Array<{ label: string; path: string; run: () => void }> = []
    for (const s of srcs || []) {
      const [nodes] = await library.tree(s.id)
      ;(nodes || []).filter((n: any) => !n.isFolder && n.title.toLowerCase().includes(q.toLowerCase())).slice(0, 8).forEach((n: any) => {
        items.push({ label: n.title, path: `${s.name} · ${n.relPath}`, run: () => { navigate(`/doc/${n.id}`); onClose() } })
      })
    }
    if (!items.length && recent?.length) {
      recent.forEach((r: any) => {
        if (r.title.toLowerCase().includes(q.toLowerCase())) {
          items.push({ label: r.title, path: "recent", run: () => { navigate(`/doc/${r.documentId}`); onClose() } })
        }
      })
    }
    return items.slice(0, 20)
  }

  async function headingItems(q: string) {
    const outline = (window as any).__currentOutline || []
    return outline
      .filter((h: any) => h.text.toLowerCase().includes(q.toLowerCase()))
      .map((h: any) => ({
        label: `${"#".repeat(h.level)} ${h.text}`,
        run: () => {
          const el = document.getElementById(h.anchor)
          el?.scrollIntoView({ behavior: "smooth", block: "start" })
          onClose()
        },
      }))
  }

  async function sourceItems(q: string) {
    const [srcs] = await sources.list()
    const [cols] = await collections.list()
    const items: Array<{ label: string; path: string; run: () => void }> = []
    ;(srcs || []).filter((s: any) => s.name.toLowerCase().includes(q.toLowerCase())).forEach((s: any) => {
      items.push({ label: s.name, path: "source", run: () => { navigate(`/source/${s.id}`); onClose() } })
    })
    ;(cols || []).filter((c: any) => c.name.toLowerCase().includes(q.toLowerCase())).forEach((c: any) => {
      items.push({ label: c.name, path: "collection", run: () => { navigate(`/collection/${c.id}`); onClose() } })
    })
    return items
  }

  async function recentItems() {
    const [recent] = await library.recent(8)
    return (recent || []).map((r: any) => ({
      label: r.title,
      path: "recently read",
      run: () => { navigate(`/doc/${r.documentId}`); onClose() },
    }))
  }

  return (
    <>
      <CommandInput placeholder="Type a command or document title…" value={q} onValueChange={setQ} />
      <CommandList>
        <CommandEmpty>No matches — press Enter to search the library for "{q}"</CommandEmpty>
        <CommandGroup heading="Commands">
          {items.map((item, i) => (
            <CommandItem key={i} value={item.label} onSelect={() => item.run()}>
              <span>{item.label}</span>
              {item.path && <span className="ml-auto text-xs text-muted-foreground">{item.path}</span>}
            </CommandItem>
          ))}
        </CommandGroup>
      </CommandList>
    </>
  )
}
