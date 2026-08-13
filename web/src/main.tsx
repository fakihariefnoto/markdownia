import React from "react"
import { createRoot } from "react-dom/client"
import { BrowserRouter } from "react-router-dom"
import "./index.css"
import App from "./App"
import { initTheme, setMode, setAccent, setReadingTheme, setReadingFont } from "./lib/theme"
import { initEvents } from "./lib/events"
import { initShortcuts, registerDefaults, dispatchAction } from "./lib/shortcuts"
import { sources } from "./lib/wails"
import { registerCommands } from "./lib/commands"
import { registerSharedActions } from "./lib/actions"
import { Toaster } from "@/components/ui/sonner"

const SPLASH_MIN_MS = 3000

function hideSplash() {
  const splash = document.getElementById("splash")
  if (!splash) return
  const shown = performance.now() - (window.__splashStart || 0)
  const wait = Math.max(0, SPLASH_MIN_MS - shown)
  setTimeout(() => {
    splash.classList.add("hidden")
    setTimeout(() => splash.remove(), 400)
  }, wait)
}

window.__splashStart = performance.now()

// Handle native menu actions dispatched from Go (menu.go → execJS).
window.addEventListener("menu-action", (e) => {
  const detail = (e as CustomEvent).detail
  if (typeof detail === "string") {
    dispatchAction(detail)
    return
  }
  if (detail && detail.action) {
    const { action, value } = detail
    if (action === "set-reading-theme") setReadingTheme(value)
    else if (action === "set-appearance") setMode(value)
    else if (action === "set-accent") setAccent(value)
    else if (action === "set-reading-font") setReadingFont(value)
    else dispatchAction(action, value)
  }
})

async function boot() {
  initEvents()
  registerCommands()
  registerSharedActions()
  registerDefaults()
  initShortcuts()

  // Theme first — the reading surface depends on the attribute writes.
  await initTheme()

  // Resolve the landing route.
  const [sourceList] = await sources.list()
  const count = Array.isArray(sourceList) ? sourceList.length : 0
  const startPath = await resolveLaunch(count)

  const container = document.getElementById("app")!
  const root = createRoot(container)
  root.render(
    <React.StrictMode>
      <BrowserRouter>
        <App startPath={startPath} />
        <Toaster position="top-right" />
      </BrowserRouter>
    </React.StrictMode>
  )
  hideSplash()
}

async function resolveLaunch(sourceCount: number): Promise<string> {
  const { resolveLaunch } = await import("./lib/router")
  return resolveLaunch(sourceCount)
}

boot().catch((err) => {
  console.error("markdownia boot failed", err)
  const app = document.getElementById("app")
  if (app) app.textContent = "Markdownia failed to start. See Help → Reveal Log File for details."
  hideSplash()
})
