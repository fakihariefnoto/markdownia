// events.ts — subscribes ONCE to the backend event stream and fans out to
// React subscribers via a pub/sub. One subscription per event, not one per
// screen.

type Handler = (payload: any) => void

const listeners = new Map<string, Set<Handler>>()

export function on(event: string, handler: Handler) {
  if (!listeners.has(event)) listeners.set(event, new Set())
  listeners.get(event)!.add(handler)
  return () => {
    listeners.get(event)?.delete(handler)
  }
}

export function emit(event: string, payload: any) {
  listeners.get(event)?.forEach((h) => {
    try {
      h(payload)
    } catch (err) {
      console.error(`events.ts: handler for "${event}" threw`, err)
    }
  })
}

const EVENT_NAMES = ["source:progress", "source:status", "source:indexed", "search:invalidated"]

// initEvents subscribes to the Wails event stream once.
export function initEvents() {
  const wailsEvents = (window as any)?.wails?.Events
  if (!wailsEvents || typeof wailsEvents.On !== "function") return
  EVENT_NAMES.forEach((name) => wailsEvents.On(name, (payload: any) => emit(name, payload)))
}

// React hooks.
import { useEffect } from "react"

export function useEvent(event: string, handler: Handler) {
  useEffect(() => on(event, handler), [event, handler])
}
