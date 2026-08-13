import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath } from 'node:url'
// Wails serves the runtime at /wails/runtime.js (see internal/assetserver).
// Injected here so Vite never tries to resolve/bundle it, while the tag still
// lands in both dev and the built dist/index.html.
const injectWailsRuntime = () => ({
  name: 'inject-wails-runtime',
  transformIndexHtml() {
    return [
      {
        tag: 'script',
        attrs: { type: 'module', src: '/wails/runtime.js' },
        injectTo: 'head-prepend',
      },
    ]
  },
})

export default defineConfig({
  plugins: [react(), tailwindcss(), injectWailsRuntime()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    emptyOutDir: true,
  },
  // Everything is bundled — no external resources, no remote fonts, no CDN.
  // The zero-network check (scripts/no-network-check.mjs) enforces this on
  // the built output.
  base: './',
})
