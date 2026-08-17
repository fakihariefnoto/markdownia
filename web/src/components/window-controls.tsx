// window-controls.tsx — custom window chrome for the frameless window.
// A slim draggable titlebar with close / minimise / maximise controls styled
// from the app theme. On macOS the controls sit on the left (traffic-light
// position); elsewhere on the right (Windows convention).

import { useEffect, useState } from "react"
import { Minus, Square, Copy, X } from "lucide-react"
import { isMac } from "@/lib/shortcuts"
import { cn } from "@/lib/utils"

// Runtime window surface injected by the Wails runtime (/wails/runtime.js).
interface RuntimeWindow {
  Minimise: () => Promise<unknown>
  ToggleMaximise: () => Promise<unknown>
  Close: () => Promise<unknown>
  IsMaximised: () => Promise<boolean>
}

function rw(): RuntimeWindow | undefined {
  return (window as any)?.wails?.Window
}

// hasRuntime reports whether the Wails runtime is present (desktop build).
export function hasWindowRuntime() {
  return typeof window !== "undefined" && !!rw()
}

export function Titlebar() {
  return (
    <header
      className="grid h-10 shrink-0 select-none grid-cols-[1fr_auto_1fr] items-center border-b border-border bg-sidebar"
      style={{ "--wails-draggable": "drag" } as React.CSSProperties}
      onDoubleClick={(e) => {
        if (isMac || e.defaultPrevented) return
        void rw()?.ToggleMaximise()
      }}
    >
      <div className="flex h-full items-stretch justify-start">{isMac ? <WindowControls /> : null}</div>
      <span className="truncate px-4 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
        Markdownia
      </span>
      <div className="flex h-full items-stretch justify-end">{isMac ? null : <WindowControls />}</div>
    </header>
  )
}

function WindowControls() {
  const [maximised, setMaximised] = useState(false)

  useEffect(() => {
    if (!hasWindowRuntime()) return
    let alive = true
    void rw()?.IsMaximised().then((m) => {
      if (alive) setMaximised(!!m)
    }).catch(() => {})
    const events = (window as any)?.wails?.Events
    if (!events?.On) return () => { alive = false }
    const offMax = events.On("common:WindowMaximise", () => alive && setMaximised(true))
    const offUn = events.On("common:WindowUnMaximise", () => alive && setMaximised(false))
    const offRestore = events.On("common:WindowRestore", () => alive && setMaximised(false))
    return () => {
      alive = false
      offMax?.()
      offUn?.()
      offRestore?.()
    }
  }, [])

  const toggleMaximise = () => {
    void rw()?.ToggleMaximise().then(() => rw()?.IsMaximised().then((m) => setMaximised(!!m))).catch(() => {})
  }

  if (!hasWindowRuntime()) return null

  const base = "flex h-full w-11 items-center justify-center text-muted-foreground transition-colors"

  return (
    <div
      className="flex h-full items-stretch"
      style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}
      onDoubleClick={(e) => e.stopPropagation()}
    >
      <button
        type="button"
        className={cn(base, "hover:bg-destructive hover:text-destructive-foreground")}
        onClick={() => void rw()?.Close()}
        title="Close"
        aria-label="Close"
      >
        <X className="size-3.5" />
      </button>
      <button
        type="button"
        className={cn(base, "hover:bg-muted hover:text-foreground")}
        onClick={() => void rw()?.Minimise()}
        title="Minimize"
        aria-label="Minimize"
      >
        <Minus className="size-3.5" />
      </button>
      <button
        type="button"
        className={cn(base, "hover:bg-muted hover:text-foreground")}
        onClick={toggleMaximise}
        title={maximised ? "Restore" : "Maximize"}
        aria-label={maximised ? "Restore" : "Maximize"}
      >
        {maximised ? <Copy className="size-3" /> : <Square className="size-3" />}
      </button>
    </div>
  )
}
