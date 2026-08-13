// actions.ts — shared action handlers wired once, dispatched by shortcuts and
// the native menu.

import { registerAction } from "./shortcuts"
import { sources, collections, native, library } from "./wails"
import { navigate } from "./nav"
import { setReadingTheme } from "./theme"
import { toastHelper } from "./toast"

let currentDocId: number | null = null

export function registerSharedActions() {
  registerAction("import-folder", () => void import("@/components/import-menu").then((m) => m.importFolder()))
  registerAction("import-zip", () => void import("@/components/import-menu").then((m) => m.importZip()))
  registerAction("import-git", () => void import("@/screens/import-git").then((m) => m.openImportGit()))
  registerAction("open-command-palette", () => dispatch("open-command-palette-overlay"))
  registerAction("open-search", () => navigate("/search"))
  registerAction("nav-back", () => navigate(-1))
  registerAction("nav-forward", () => navigate(1))

  registerAction("source-refresh", async () => {
    const [list] = await sources.list()
    const active = list?.find((s: any) => window.location.pathname.includes(`/source/${s.id}`))
    if (active) {
      void sources.refresh(active.id)
    } else {
      toastHelper.warning("No source selected", { description: "Open a source first." })
    }
  })

  registerAction("new-collection", () => void import("@/components/dialogs").then((m) => m.openDialog({ kind: "new-collection", open: true })))
  registerAction("add-to-collection", (docId: number) => void import("@/components/dialogs").then((m) => m.openDialog({ kind: "add-to-collection", open: true, docId })))

  registerAction("bookmark-document", (docId: number) => {
    void library.meta(docId)
  })

  registerAction("set-reading-theme", setReadingTheme)
  registerAction("close-overlay", () => dispatch("close-overlay-overlay"))

  registerAction("check-updates", () => {
    void native.checkForUpdates("anofac/markdownia").then(([res]) => {
      toastHelper.success(res?.message || "Update check complete")
    })
  })

  registerAction("reveal-log", () => void native.reveal(""))
  registerAction("open-help", () => toastHelper.success("Keyboard Shortcuts", { description: "⌘K palette · ⌘⇧F search · ⌘+ / ⌘- zoom" }))
}

export function setCurrentDoc(id: number | null) {
  currentDocId = id
}

export function getCurrentDoc() {
  return currentDocId
}

// dispatch is a small forwarder so action handlers can fire other actions.
function dispatch(name: string, ...args: any[]) {
  import("./shortcuts").then((s) => s.dispatchAction(name, ...args))
}

export { currentDocId }
