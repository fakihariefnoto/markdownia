import { useEffect, useState } from "react"
import { subscribeTheme, type ThemeState } from "./theme"

export function useTheme() {
  const [state, setState] = useState<ThemeState>(() => {
    // Deferred — SSR-unsafe access lives behind the DOM check.
    return { mode: "system", accent: "teal", readingTheme: "paper", readingFont: "sans", readingSize: 1, readingMeasure: "72ch" }
  })

  useEffect(() => {
    return subscribeTheme(setState)
  }, [])

  return state
}
