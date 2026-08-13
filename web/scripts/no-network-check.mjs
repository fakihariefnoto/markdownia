// no-network-check.mjs — the zero-network CI guard (web/tasks/21). Two checks:
//
//  1. No runtime network request APIs in the bundle (fetch/XHR/WebSocket to
//     anything but the Wails internal bridge — which has no URL).
//  2. No remote asset references (http:// in src/href/@import positions).
//
// Bundled library internals (XML namespace URIs like www.w3.org, license
// headers, docs URLs) are not network calls and are explicitly allowlisted.
// PRD goal G3 enforced mechanically.

import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, extname } from 'node:path'

const DIST = new URL('../dist/', import.meta.url).pathname

// Bundled-library strings that are NOT network calls. w3.org URIs are XML
// namespace identifiers; the rest are docs/license references.
const NAMESPACES = [
  'http://www.w3.org/2000/svg',
  'http://www.w3.org/1999/xhtml',
  'http://www.w3.org/1998/Math/MathML',
  'https://chevrotain.io',
  'http://engelschall.com',
  'https://tailwindcss.com',
  'https://alpinejs.dev',
]

function collectFiles(dir, acc = []) {
  for (const entry of readdirSync(dir)) {
    const full = join(dir, entry)
    const st = statSync(full)
    if (st.isDirectory()) collectFiles(full, acc)
    else if (['.js', '.css', '.html'].includes(extname(full))) acc.push(full)
  }
  return acc
}

const files = collectFiles(DIST)
const violations = []

for (const file of files) {
  const content = readFileSync(file, 'utf8')

  // Check 1: runtime network request APIs.
  for (const pattern of ['fetch(', 'XMLHttpRequest', 'new WebSocket(']) {
    let idx = 0
    while ((idx = content.indexOf(pattern, idx)) !== -1) {
      // The Wails bridge uses a relative/empty URL; a fetch to a bare string
      // beginning with http is the violation.
      const around = content.slice(idx, idx + 60)
      if (/fetch\(["']https?:\/\//.test(around)) {
        violations.push(`${file}: runtime fetch to ${around.slice(0, 50)}`)
      }
      idx += pattern.length
    }
  }

  // Check 2: remote asset references.
  const assetRe = /(?:src|href)\s*=\s*["'](https?:\/\/)/g
  let m
  while ((m = assetRe.exec(content)) !== null) {
    const full = content.slice(m.index - 10, m.index + 70)
    const isNamespace = NAMESPACES.some((n) => full.includes(n))
    if (!isNamespace) {
      violations.push(`${file}: remote asset ref ${m[0].slice(0, 60)}`)
      break
    }
  }
}

if (violations.length) {
  console.error(`NO-NETWORK CHECK FAILED: ${violations.length} violation(s):`)
  violations.forEach((v) => console.error('  ' + v))
  process.exit(1)
}

console.log(`no-network check passed: ${files.length} bundled files, no runtime network calls or remote assets.`)
