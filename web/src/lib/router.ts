// router.ts — routing bridge. Path shapes are STABLE (open_tabs and
// reading_state persist route targets across restarts), so a path shape change
// is a migration, not a refactor. Screens navigate through React Router; this
// module keeps the same paths the vanilla frontend used.

import { reading } from "./wails"

export const ROUTES = [
  { name: "first-run", pattern: /^\/welcome$/ },
  { name: "library-home", pattern: /^\/$/ },
  { name: "reader", pattern: /^\/doc\/(\d+)(?:\/(.*))?$/ },
  { name: "reader-split", pattern: /^\/doc\/(\d+)\/split\/(\d+)$/ },
  { name: "search-results", pattern: /^\/search$/ },
  { name: "source-overview", pattern: /^\/source\/(\d+)$/ },
  { name: "collection-view", pattern: /^\/collection\/(\d+)$/ },
  { name: "annotations", pattern: /^\/annotations$/ },
  { name: "settings", pattern: /^\/settings$/ },
]

export interface RouteParams {
  raw: string
  documentId?: number
  leftId?: number
  rightId?: number
  sourceId?: number
  collectionId?: number
}

export interface ParsedRoute {
  name: string
  params: RouteParams
}

// parsePath resolves a location string into {name, params}.
export function parsePath(path: string): ParsedRoute {
  for (const r of ROUTES) {
    const m = path.match(r.pattern)
    if (m) {
      const params: RouteParams = { raw: path }
      if (r.name === "reader") params.documentId = Number(m[1])
      if (r.name === "reader-split") {
        params.leftId = Number(m[1])
        params.rightId = Number(m[2])
      }
      if (r.name === "source-overview") params.sourceId = Number(m[1])
      if (r.name === "collection-view") params.collectionId = Number(m[1])
      return { name: r.name, params }
    }
  }
  return { name: "library-home", params: { raw: path } }
}

// launchResolve implements the three-way restore: last reading state →
// library-home → first-run (when zero sources exist).
export async function resolveLaunch(sourceCount: number): Promise<string> {
  if (sourceCount === 0) return "/welcome"
  const [state, err] = await reading.lastContext()
  if (err || !state || !state.documentId) return "/"
  return state.contextType === "collection"
    ? `/collection/${state.contextId}`
    : state.contextType === "source"
      ? `/source/${state.contextId}`
      : `/doc/${state.documentId}`
}
