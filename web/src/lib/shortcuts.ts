// shortcuts.ts — global action registry + key map. Native menu items, command
// palette, and keyboard shortcuts all dispatch the same action objects.

type Handler = (...args: any[]) => any

const actionRegistry = new Map<string, Handler>()

export function registerAction(name: string, handler: Handler) {
  actionRegistry.set(name, handler)
  return () => actionRegistry.delete(name)
}

export function dispatchAction(name: string, ...args: any[]) {
  const h = actionRegistry.get(name)
  if (!h) {
    console.warn(`shortcuts.ts: no handler for action "${name}"`)
    return false
  }
  h(...args)
  return true
}

export function hasAction(name: string) {
  return actionRegistry.has(name)
}

export function allActions() {
  return [...actionRegistry.keys()]
}

export const isMac = typeof navigator !== "undefined" && /Mac|iP(hone|ad|od)/.test(navigator.platform || "")

export function modKey() {
  return isMac ? "⌘" : "Ctrl"
}

type KeySpec = {
  mod?: boolean
  shift?: boolean
  alt?: boolean
  key: string
  action: string
  prevent?: boolean
}

const keyMap: KeySpec[] = []

export function bindKey(spec: KeySpec) {
  keyMap.push(spec)
}

export function initShortcuts() {
  window.addEventListener("keydown", (e) => {
    const tag = (e.target as HTMLElement)?.tagName
    if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") {
      if (e.key === "Escape") (e.target as HTMLElement).blur()
      return
    }
    if (document.querySelector('[data-focus-trap="true"]') && e.key === "Escape") {
      return
    }
    for (const spec of keyMap) {
      const modMatch = !spec.mod || e.metaKey || e.ctrlKey
      const shiftMatch = !spec.shift || e.shiftKey
      const altMatch = !spec.alt || e.altKey
      if (modMatch && shiftMatch && altMatch && e.key.toLowerCase() === spec.key.toLowerCase()) {
        if (spec.prevent !== false) e.preventDefault()
        dispatchAction(spec.action)
        return
      }
    }
  })
}

export function registerDefaults() {
  bindKey({ mod: true, key: "k", action: "command-palette" })
  bindKey({ mod: true, shift: true, key: "f", action: "search-library" })
  bindKey({ mod: true, key: "f", action: "find-in-document" })
  bindKey({ mod: true, key: "[", action: "history-back" })
  bindKey({ mod: true, key: "]", action: "history-forward" })
  bindKey({ mod: true, key: "b", action: "toggle-sidebar" })
  bindKey({ mod: true, shift: true, key: "b", action: "toggle-outline" })
  bindKey({ mod: true, key: "\\", action: "toggle-split" })
  bindKey({ mod: true, shift: true, key: "h", action: "library-home" })
  bindKey({ mod: true, key: "+", action: "zoom-in" })
  bindKey({ mod: true, key: "=", action: "zoom-in" })
  bindKey({ mod: true, key: "-", action: "zoom-out" })
  bindKey({ mod: true, key: "0", action: "zoom-reset" })
  bindKey({ mod: true, key: "r", action: "refresh-source" })
  bindKey({ mod: true, key: "p", action: "export-pdf" })
  bindKey({ mod: true, key: "w", action: "close-tab" })
  bindKey({ mod: true, key: "o", action: "import-folder" })
  bindKey({ mod: true, shift: true, key: "o", action: "import-git" })
  bindKey({ mod: true, key: "n", action: "new-collection" })
  bindKey({ mod: true, key: ",", action: "settings" })
  bindKey({ mod: false, key: "Escape", action: "close-overlay" })
}
