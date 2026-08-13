// copy-dist.mjs — copies the web/ build output into frontend/dist for the
// Go embed (main.go uses //go:embed all:frontend/dist). The web/ frontend is
// the source of truth; this is the only glue.

import { rmSync, cpSync, mkdirSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = join(dirname(fileURLToPath(import.meta.url)), '..')
// root = frontend → web is ../web relative to root
const webDist = join(root, '..', 'web', 'dist')
const embedDist = join(root, 'dist')

mkdirSync(dirname(embedDist), { recursive: true })
rmSync(embedDist, { recursive: true, force: true })
cpSync(webDist, embedDist, { recursive: true })

console.log(`copied ${webDist} → ${embedDist}`)
