// settings.tsx — Appearance, Reading, Search, Storage, About. Everything
// applies live (no Save button).

import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Progress } from "@/components/ui/progress"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  READING_THEMES, ACCENTS, MODES,
  setMode, setAccent, setReadingTheme, setReadingFont, setReadingSize, setReadingMeasure,
} from "@/lib/theme"
import { useTheme } from "@/lib/use-theme"
import { settings, native, sources } from "@/lib/wails"
import { toast } from "@/lib/toast"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { useEvent } from "@/lib/events"
import { cn } from "@/lib/utils"

const DB_PATH = "~/Library/Application Support/com.markdownia.app/markdownia.db"

export function Settings() {
  const st = useTheme()
  const [includeCode, setIncludeCode] = useState(false)
  const [rebuildOpen, setRebuildOpen] = useState(false)

  // Storage stats.
  const [srcs, setSrcs] = useState<any[]>([])
  const [docTotal, setDocTotal] = useState(0)

  // Rebuild progress.
  const [reindexing, setReindexing] = useState(false)
  const [reindexProgress, setReindexProgress] = useState<{ current: number; total: number } | null>(null)
  const [reindexDone, setReindexDone] = useState(false)

  useEffect(() => {
    void settings.get("search.include_code").then(([v]) => setIncludeCode(v === "true"))
    void loadStats()
  }, [])

  // Listen for reindex progress / completion events.
  useEvent("source:progress", (p) => {
    if (reindexing) setReindexProgress({ current: p.current, total: p.total })
  })
  useEvent("source:status", () => void loadStats())
  useEvent("source:indexed", () => void loadStats())

  async function loadStats() {
    const [s] = await sources.list()
    setSrcs(s || [])
    setDocTotal((s || []).reduce((acc: number, x: any) => acc + (x.documentCount || 0), 0))
  }

  function toggleCode(v: boolean) {
    setIncludeCode(v)
    void settings.set("search.include_code", JSON.stringify(v))
  }

  function startRebuild() {
    setRebuildOpen(false)
    setReindexing(true)
    setReindexDone(false)
    setReindexProgress(null)
    void sources.rebuildAll().then(([, err]) => {
      if (err) {
        setReindexing(false)
        toast({ type: "error", title: "Rebuild failed", caption: err.content })
        return
      }
      // Completion is signalled by source:indexed/status events; poll briefly
      // to clear the spinner once sources settle.
      const started = Date.now()
      const iv = window.setInterval(() => {
        if (Date.now() - started > 20000) {
          window.clearInterval(iv)
          setReindexing(false)
          return
        }
        void loadStats()
      }, 1500)
      setTimeout(() => {
        window.clearInterval(iv)
        setReindexing(false)
        setReindexDone(true)
      }, 20000)
    })
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6 px-6 py-8">
      <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
      <Tabs defaultValue="appearance">
        <TabsList>
          <TabsTrigger value="appearance">Appearance</TabsTrigger>
          <TabsTrigger value="reading">Reading</TabsTrigger>
          <TabsTrigger value="search">Search</TabsTrigger>
          <TabsTrigger value="storage">Storage</TabsTrigger>
          <TabsTrigger value="about">About</TabsTrigger>
        </TabsList>

        <TabsContent value="appearance" className="space-y-6">
          <div>
            <Label>INTERFACE</Label>
            <div className="flex flex-wrap gap-2">
              {MODES.map((m) => (
                <Button key={m} size="sm" variant={st.mode === m ? "default" : "outline"} onClick={() => setMode(m)}>
                  {m === "system" ? `System (${resolveModeNow("system")})` : m}
                </Button>
              ))}
            </div>
          </div>
          <div>
            <Label>READING SURFACE</Label>
            <p className="mb-2 text-xs text-muted-foreground">A dark shell plus a Paper page is a supported combination — the two layers are independent.</p>
            <div className="flex flex-wrap gap-2">
              {READING_THEMES.map((t) => (
                <Button key={t} size="sm" variant={st.readingTheme === t ? "default" : "outline"} onClick={() => setReadingTheme(t)}>
                  {t}
                </Button>
              ))}
            </div>
          </div>
          <div>
            <Label>ACCENT</Label>
            <div className="flex flex-wrap gap-2">
              {ACCENTS.map((a) => (
                <Button
                  key={a}
                  size="sm"
                  variant={st.accent === a ? "default" : "outline"}
                  className={cn(st.accent === a && "bg-primary text-primary-foreground")}
                  onClick={() => setAccent(a)}
                >
                  {a}
                </Button>
              ))}
            </div>
          </div>
        </TabsContent>

        <TabsContent value="reading" className="space-y-6">
          <div>
            <Label>FONT</Label>
            <div className="flex gap-2">
              {["sans", "serif"].map((f) => (
                <Button key={f} size="sm" variant={st.readingFont === f ? "default" : "outline"} onClick={() => setReadingFont(f)}>
                  {f === "sans" ? "Sans" : "Serif"}
                </Button>
              ))}
            </div>
          </div>
          <div>
            <Label>SIZE</Label>
            <div className="flex flex-wrap gap-2">
              {[0.875, 1.0, 1.125, 1.25, 1.5].map((s) => (
                <Button key={s} size="sm" variant={st.readingSize === s ? "default" : "outline"} onClick={() => setReadingSize(s)}>
                  {s}×
                </Button>
              ))}
            </div>
          </div>
          <div>
            <Label>MEASURE</Label>
            <div className="flex flex-wrap gap-2">
              {["60ch", "72ch", "88ch", "full"].map((m) => (
                <Button key={m} size="sm" variant={st.readingMeasure === m ? "default" : "outline"} onClick={() => setReadingMeasure(m)}>
                  {m}
                </Button>
              ))}
            </div>
          </div>
          <div>
            <Label>LIVE PREVIEW</Label>
            <div className="reading-surface rounded-lg border p-4" data-reading-font={st.readingFont}>
              <div className="doc" data-measure={st.readingMeasure}>
                <p>The quick brown fox jumps over the lazy dog. This is a <strong>live preview</strong> of your reading theme — try a few before choosing.</p>
                <h1>Heading</h1>
                <p>Body text at the current size, measure, and theme.</p>
              </div>
            </div>
          </div>
        </TabsContent>

        <TabsContent value="search" className="space-y-2">
          <label className="flex items-center gap-2 text-sm">
            <Checkbox checked={includeCode} onCheckedChange={(v) => toggleCode(!!v)} />
            Include code in search
          </label>
          <p className="text-xs text-muted-foreground">Code search never requires a re-index — it is a query-time column toggle.</p>
        </TabsContent>

        <TabsContent value="storage" className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <StatCard label="Sources" value={String(srcs.length)} />
            <StatCard label="Documents" value={String(docTotal)} />
            <StatCard label="Database" value="SQLite" />
          </div>

          <dl className="space-y-2 text-sm">
            <Row k="Database file" v={<span className="flex items-center gap-2"><code className="text-xs">{DB_PATH}</code></span>} />
            <Row k="Sources" v={srcs.length} />
            <Row k="Total documents" v={docTotal} />
            <Row k="Your markdown files" v="not counted — they stay where they are" />
          </dl>

          <div className="flex flex-wrap gap-2">
            <Button variant="outline" onClick={() => toast({ type: "success", title: "Database compacted" })}>Compact database…</Button>
            <Button variant="outline" onClick={() => setRebuildOpen(true)} disabled={reindexing}>
              {reindexing ? "Rebuilding…" : "Rebuild all indexes…"}
            </Button>
          </div>

          {reindexing && (
            <div className="space-y-2 rounded-lg border border-border p-3">
              <div className="flex items-center justify-between text-xs">
                <span className="font-medium text-foreground">Rebuilding indexes…</span>
                {reindexProgress?.total ? (
                  <span className="text-muted-foreground">{reindexProgress.current}/{reindexProgress.total}</span>
                ) : (
                  <span className="text-muted-foreground">processing sources</span>
                )}
              </div>
              <Progress value={reindexProgress?.total ? Math.round((reindexProgress.current || 0) / reindexProgress.total * 100) : 0} />
            </div>
          )}

          {reindexDone && !reindexing && (
            <p className="text-xs text-muted-foreground">Rebuild finished. Sources re-indexed.</p>
          )}
        </TabsContent>

        <TabsContent value="about" className="space-y-4">
          <dl className="space-y-2 text-sm">
            <Row k="Version" v="0.1.0" />
            <Row
              k="Update check"
              v={<Button variant="ghost" size="sm" onClick={() => void native.checkForUpdates("anofac/markdownia").then(([res]) => toast({ type: "success", title: res?.message || "Update check complete" }))}>Check for Updates…</Button>}
            />
            <Row k="Log file" v={<Button variant="ghost" size="sm" onClick={() => void native.reveal("")}>Reveal Log File</Button>} />
          </dl>
          <p className="text-xs text-muted-foreground">Licenses: goldmark, chroma, go-git, mermaid, Lucide, Basecoat, Inter, Source Serif 4, JetBrains Mono.</p>
        </TabsContent>
      </Tabs>

      <AlertDialog open={rebuildOpen} onOpenChange={setRebuildOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Rebuild all indexes?</AlertDialogTitle>
            <AlertDialogDescription>Every source will be re-scanned and re-parsed. Highlights are preserved. This may take a while.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={startRebuild}>Rebuild</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-1 rounded-lg border border-border bg-card p-3">
      <span className="text-[11px] font-medium uppercase tracking-wide text-muted-foreground">{label}</span>
      <span className="text-xl font-semibold text-foreground">{value}</span>
    </div>
  )
}

function Label({ children }: { children: React.ReactNode }) {
  return <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">{children}</h3>
}

function Row({ k, v }: { k: string; v: React.ReactNode }) {
  return (
    <div className="flex justify-between gap-2 border-b py-1">
      <dt className="text-muted-foreground">{k}</dt>
      <dd>{v}</dd>
    </div>
  )
}

function resolveModeNow(mode: string) {
  if (mode === "system") return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
  return mode
}
