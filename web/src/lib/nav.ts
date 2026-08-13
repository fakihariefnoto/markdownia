// nav.ts — module-level navigation bridge. React Router's navigate instance is
// registered once from <App/>; action handlers, shortcuts, and screens that are
// outside component scope call navigate() here. Keeps the vanilla port's call
// sites unchanged.

import type { NavigateFunction } from "react-router-dom"

let nav: NavigateFunction | null = null

export function setNavigate(fn: NavigateFunction) {
  nav = fn
}

export function navigate(to: string | number, opts?: { replace?: boolean; state?: any }) {
  if (nav) {
    nav(to as any, opts as any)
  } else {
    if (typeof to === "number") window.history.go(to)
    else window.location.assign(to)
  }
}

// Convenience alias matching the vanilla router signature.
export function go(path: string, opts?: { push?: boolean }) {
  navigate(path, { replace: opts?.push === false })
}
