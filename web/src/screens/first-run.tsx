// first-run.tsx — three import cards stating what each kind actually does.

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Folder, GitBranch, Package } from "lucide-react"
import { importFolder, importZip } from "@/components/import-menu"
import { dispatchAction } from "@/lib/shortcuts"

export function FirstRun() {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-8 px-6 py-10">
      <div className="text-center">
        <h1 className="text-3xl font-semibold tracking-tight">Markdownia</h1>
        <p className="mt-2 max-w-lg text-sm text-muted-foreground">
          Turn any pile of markdown — a folder, a git repo, a zip — into a beautifully rendered reading library. Fully offline.
        </p>
      </div>
      <div className="grid max-w-4xl grid-cols-1 gap-4 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2"><Folder className="size-4" /> Import a Folder</CardTitle>
            <CardDescription>
              Read your notes in place. Files are <strong>referenced, never copied</strong> — edit them outside and changes appear on the next scan.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={importFolder} className="w-full">Choose Folder…</Button>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2"><GitBranch className="size-4" /> Import a Git Repository</CardTitle>
            <CardDescription>
              Paste a URL and the docs are cloned locally. <strong>This is the only step that touches the network</strong>.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={() => dispatchAction("import-git")} className="w-full">Paste a URL…</Button>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2"><Package className="size-4" /> Import a Zip</CardTitle>
            <CardDescription>
              Downloaded a docs archive? It is <strong>extracted into the app's storage</strong> so the index stays fresh.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button onClick={importZip} className="w-full">Choose Zip…</Button>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
