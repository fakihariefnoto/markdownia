// theme.ts — applies both theme axes by setting data attributes on <html>.
// Ported from the vanilla frontend for React. A tiny pub/sub lets React hooks
// re-render when a setting changes.

import { settings } from "./wails"

export const READING_THEMES = ["paper", "sepia", "solarized", "nord", "dracula", "gruvbox"]
export const ACCENTS = ["teal", "indigo", "forest", "copper", "plum", "slate"]
export const MODES = ["light", "dark", "system"]

export interface ThemeState {
  mode: string
  accent: string
  readingTheme: string
  readingFont: string
  readingSize: number
  readingMeasure: string
}

let current: ThemeState = {
  mode: "system",
  accent: "teal",
  readingTheme: "paper",
  readingFont: "sans",
  readingSize: 1.0,
  readingMeasure: "72ch",
}

// localStorage cache of the last-known theme so the app applies it immediately
// on boot — before the async settings read finishes — avoiding a flash back to
// the hardcoded defaults in index.html.
const THEME_CACHE_KEY = "markdownia.theme.v1"

function loadCachedTheme(): void {
  try {
    const raw = localStorage.getItem(THEME_CACHE_KEY)
    if (!raw) return
    const parsed = JSON.parse(raw)
    if (parsed.mode) current.mode = parsed.mode
    if (parsed.accent) current.accent = parsed.accent
    if (parsed.readingTheme) current.readingTheme = parsed.readingTheme
    if (parsed.readingFont) current.readingFont = parsed.readingFont
    if (parsed.readingSize != null) current.readingSize = Number(parsed.readingSize)
    if (parsed.readingMeasure) current.readingMeasure = parsed.readingMeasure
  } catch {
    /* corrupt cache — defaults stand */
  }
}

function cacheTheme(): void {
  try {
    localStorage.setItem(THEME_CACHE_KEY, JSON.stringify(current))
  } catch {
    /* storage unavailable — settings DB is still the source of truth */
  }
}

const listeners = new Set<(s: ThemeState) => void>()

function notify() {
  const snap = getState()
  listeners.forEach((h) => {
    try {
      h(snap)
    } catch {
      /* subscriber errors never break theme */
    }
  })
}

export function subscribeTheme(h: (s: ThemeState) => void) {
  listeners.add(h)
  h(getState())
  return () => {
    listeners.delete(h)
  }
}

export function resolveMode(mode: string) {
  if (mode === "system") {
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
  }
  return mode
}

export function apply() {
  const mode = resolveMode(current.mode)
  const html = document.documentElement
  html.dataset.theme = mode
  html.dataset.readingTheme = current.readingTheme
  html.dataset.accent = current.accent
  html.dataset.readingFont = current.readingFont
  html.dataset.readingSize = String(current.readingSize)
  html.dataset.readingMeasure = current.readingMeasure
}

// Apply the cached theme immediately on import so the UI (and splash) never
// flashes the defaults before the async settings read completes.
loadCachedTheme()
apply()

export async function initTheme() {
  try {
    const [all] = await settings.getAll()
    if (all?.["appearance.mode"]) current.mode = JSON.parse(all["appearance.mode"])
    if (all?.["appearance.accent"]) current.accent = JSON.parse(all["appearance.accent"])
    if (all?.["reading.theme"]) current.readingTheme = JSON.parse(all["reading.theme"])
    if (all?.["reading.font"]) current.readingFont = JSON.parse(all["reading.font"])
    if (all?.["reading.size"] !== undefined) current.readingSize = Number(JSON.parse(all["reading.size"]) ?? 1)
    if (all?.["reading.measure"]) current.readingMeasure = JSON.parse(all["reading.measure"])
  } catch {
    /* defaults stand */
  }
  apply()
  cacheTheme()

  const darkMedia = window.matchMedia("(prefers-color-scheme: dark)")
  const sync = () => {
    if (current.mode === "system") apply()
  }
  darkMedia.addEventListener("change", sync)

  return getState()
}

export function setMode(mode: string) {
  current.mode = mode
  apply()
  notify()
  cacheTheme()
  void settings.set("appearance.mode", JSON.stringify(mode))
}

export function setAccent(accent: string) {
  current.accent = accent
  apply()
  notify()
  cacheTheme()
  void settings.set("appearance.accent", JSON.stringify(accent))
}

export function setReadingTheme(theme: string) {
  current.readingTheme = theme
  apply()
  notify()
  cacheTheme()
  void settings.set("reading.theme", JSON.stringify(theme))
}

export function setReadingFont(font: string) {
  current.readingFont = font
  apply()
  notify()
  cacheTheme()
  void settings.set("reading.font", JSON.stringify(font))
}

export function setReadingSize(multiplier: number) {
  current.readingSize = multiplier
  apply()
  notify()
  cacheTheme()
  void settings.set("reading.size", JSON.stringify(multiplier))
}

export function setReadingMeasure(measure: string) {
  current.readingMeasure = measure
  apply()
  cacheTheme()
  notify()
  void settings.set("reading.measure", JSON.stringify(measure))
}

export function getState() {
  return { ...current }
}
