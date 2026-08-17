// App.tsx — root component. Registers the nav bridge, hosts the Shell, and
// renders the route tree.

import { useEffect } from "react"
import { Routes, Route, Navigate, useNavigate, useLocation } from "react-router-dom"
import { setNavigate } from "@/lib/nav"
import { Shell } from "@/components/shell"
import { FirstRun } from "@/screens/first-run"
import { LibraryHome } from "@/screens/library-home"
import { Reader } from "@/screens/reader"
import { SearchResults } from "@/screens/search-results"
import { SourceOverview } from "@/screens/source-overview"
import { CollectionView } from "@/screens/collection-view"
import { Annotations } from "@/screens/annotations"
import { Settings } from "@/screens/settings"
import { CommandPaletteOverlay } from "@/screens/command-palette"
import { DialogHost } from "@/components/dialogs"
import { ImportGitHost } from "@/screens/import-git"
import { Titlebar } from "@/components/window-controls"

export default function App({ startPath }: { startPath: string }) {
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    setNavigate(navigate)
  }, [navigate])

  return (
    <>
      <div className="flex h-screen w-full flex-col overflow-hidden bg-background text-foreground">
        <Titlebar />
        <div className="min-h-0 flex-1">
          <Routes>
            <Route path="/" element={<Shell />}>
              <Route index element={<LibraryHome />} />
              <Route path="welcome" element={<FirstRun />} />
              <Route path="search" element={<SearchResults />} />
              <Route path="annotations" element={<Annotations />} />
              <Route path="settings" element={<Settings />} />
              <Route path="source/:sourceId" element={<SourceOverview />} />
              <Route path="collection/:collectionId" element={<CollectionView />} />
              <Route path="doc/:documentId" element={<Reader />} />
              <Route path="doc/:leftId/split/:rightId" element={<Reader split />} />
            </Route>
            <Route path="*" element={<Navigate to={startPath} replace />} />
          </Routes>
        </div>
      </div>
      <CommandPaletteOverlay key={location.pathname} />
      <DialogHost />
      <ImportGitHost />
    </>
  )
}
