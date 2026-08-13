// import-menu.tsx — the import kind actions (folder/git/zip).

import { native, sources } from "@/lib/wails"
import { navigate } from "@/lib/nav"
import { toast } from "@/lib/toast"
import { dispatchAction } from "@/lib/shortcuts"

function log(...args: unknown[]) {
  console.log("[import]", ...args)
}

export function importFolder() {
  log("importFolder start")
  void native.pickFolder().then(([result, err]) => {
    log("pickFolder result", JSON.stringify(result), "err", err)
    if (err || !result) return
    const [path, ok] = result
    if (!ok || !path) {
      log("pickFolder cancelled or empty", path, ok)
      return
    }
    log("picked folder path:", path)
    void sources.importFolder(path).then(([id, importErr]) => {
      log("importFolder id:", id, "err:", importErr)
      if (importErr) {
        toast({ type: "error", title: "Import failed", caption: importErr.content })
        return
      }
      if (id) navigate(`/source/${id}`)
    })
  })
}

export function importZip() {
  log("importZip start")
  void native.pickZip().then(([result, err]) => {
    log("pickZip result", JSON.stringify(result), "err", err)
    if (err || !result) return
    const [path, ok] = result
    if (!ok || !path) {
      log("pickZip cancelled or empty", path, ok)
      return
    }
    log("picked zip path:", path)
    void sources.importZip(path).then(([id, importErr]) => {
      log("importZip id:", id, "err:", importErr)
      if (importErr) {
        toast({ type: "error", title: "Import failed", caption: importErr.content })
        return
      }
      if (id) navigate(`/source/${id}`)
    })
  })
}

export { dispatchAction }
