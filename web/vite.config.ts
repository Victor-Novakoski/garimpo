import { writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

// The build goes straight into the Go package that embeds it, so
// `garimpo serve` carries the panel inside the binary.
const outDir = resolve(import.meta.dirname, '../internal/panel/dist')

export default defineConfig({
  plugins: [
    react(),
    {
      // The folder must exist in git for the Go build to work without the
      // panel; Vite empties it, so the placeholder goes back in.
      name: 'keep-placeholder',
      apply: 'build',
      closeBundle() {
        writeFileSync(resolve(outDir, '.gitkeep'), '')
      },
    },
  ],
  build: { outDir, emptyOutDir: true },
  server: {
    // `npm run dev` talks to a `garimpo serve` running beside it.
    host: '127.0.0.1',
    proxy: { '/api': 'http://127.0.0.1:8080' },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['src/test/setup.ts'],
    restoreMocks: true,
  },
})
