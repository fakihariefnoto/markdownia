// commands.ts — the command registry. The SAME source the native menu
// dispatches from. A command in one and not the other is the inconsistency
// nobody notices until a user asks.

import { registerAction, dispatchAction } from "./shortcuts"
import { navigate } from "./nav"
import { setMode, setAccent, setReadingFont, setReadingTheme } from "./theme"
import { native } from "./wails"
import { toastHelper } from "./toast"

export function registerCommands() {
  registerAction("command-palette", () => dispatchAction("open-command-palette"))
  registerAction("search-library", () => dispatchAction("open-search"))
  registerAction("find-in-document", () => dispatchAction("open-find-bar"))
  registerAction("history-back", () => navigate(-1))
  registerAction("history-forward", () => navigate(1))
  registerAction("toggle-sidebar", () => dispatchAction("sidebar-toggle"))
  registerAction("toggle-outline", () => dispatchAction("outline-toggle"))
  registerAction("toggle-split", () => dispatchAction("split-toggle"))
  registerAction("library-home", () => navigate("/"))
  registerAction("zoom-in", () => dispatchAction("reading-zoom", 0.125))
  registerAction("zoom-out", () => dispatchAction("reading-zoom", -0.125))
  registerAction("zoom-reset", () => dispatchAction("reading-zoom-reset"))

  // Layout toggles (the shell subscribes to these).
  registerAction("sidebar-toggle", () => dispatch("sidebar-toggle"))
  registerAction("outline-toggle", () => dispatch("outline-toggle"))
  registerAction("split-toggle", () => dispatch("split-toggle"))
  registerAction("tab-close", () => dispatch("tab-close"))
  registerAction("close-all-tabs", () => dispatch("close-all-tabs"))

  // Reading zoom + export.
  registerAction("reading-zoom", (delta: number) => void import("@/lib/theme").then((t) => {
    const next = Math.max(0.875, Math.min(1.5, t.getState().readingSize + delta))
    t.setReadingSize(Number(next.toFixed(3)))
  }))
  registerAction("reading-zoom-reset", () => void import("@/lib/theme").then((t) => t.setReadingSize(1.0)))
  registerAction("open-export", (_format: string, _scopeId?: number) => toastHelper.success("Export", { description: "Use File → Export as PDF/HTML." }))

  registerAction("refresh-source", () => dispatchAction("source-refresh"))
  registerAction("export-pdf", () => dispatchAction("open-export", "pdf"))
  registerAction("export-html", () => dispatchAction("open-export", "html"))
  registerAction("close-tab", () => dispatchAction("tab-close"))
  registerAction("close-all-tabs", () => dispatchAction("close-all-tabs"))
  registerAction("import-folder", () => dispatchAction("import-folder"))
  registerAction("import-git", () => dispatchAction("import-git"))
  registerAction("new-collection", () => dispatchAction("new-collection"))
  registerAction("settings", () => navigate("/settings"))
  registerAction("annotations", () => navigate("/annotations"))
  registerAction("close-overlay", () => dispatchAction("close-overlay"))
  registerAction("help-shortcuts", () => dispatchAction("open-help"))
  registerAction("open-about", () => toastHelper.success("Markdownia", { description: "Offline-first markdown reader · 0.1.0" }))
  registerAction("check-updates", () => dispatchAction("check-updates"))
  registerAction("reveal-log", () => dispatchAction("reveal-log"))

  // Theme setters (native menu values).
  registerAction("set-appearance", (v: string) => setMode(v))
  registerAction("set-accent", (v: string) => setAccent(v))
  registerAction("set-reading-font", (v: string) => setReadingFont(v))
  registerAction("set-reading-theme", (v: string) => setReadingTheme(v))

  // Source operations — open the relevant dialog in the source overview.
  registerAction("rename-source", () => window.dispatchEvent(new CustomEvent("markdownia:source-action", { detail: { action: "rename" } })))
  registerAction("relocate-source", () => window.dispatchEvent(new CustomEvent("markdownia:source-action", { detail: { action: "relocate" } })))
  registerAction("delete-source", () => window.dispatchEvent(new CustomEvent("markdownia:source-action", { detail: { action: "delete" } })))
  registerAction("reveal-source", () => window.dispatchEvent(new CustomEvent("markdownia:source-action", { detail: { action: "reveal" } })))

  // Find next/previous (find bar cycles).
  registerAction("find-next", () => dispatchAction("find-next"))
  registerAction("find-previous", () => dispatchAction("find-previous"))
}

// dispatch forwards to the action registry (avoids import cycles).
function dispatch(name: string, ...args: any[]) {
  dispatchAction(name, ...args)
}
