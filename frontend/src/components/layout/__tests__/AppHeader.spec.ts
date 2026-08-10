import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const testDirectory = dirname(fileURLToPath(import.meta.url))
const headerSource = readFileSync(resolve(testDirectory, '../AppHeader.vue'), 'utf8')
const layoutSource = readFileSync(resolve(testDirectory, '../AppLayout.vue'), 'utf8')
const modernCompatSource = readFileSync(
  resolve(testDirectory, '../../../styles/ui-modern-compat-parts/part-05.css'),
  'utf8'
)

describe('AppHeader shell integration', () => {
  it('keeps the top bar full-width, square and fixed inside the page shell', () => {
    const topbarStyle = modernCompatSource.match(/\.app-topbar\s*\{[\s\S]*?\n\}/)?.[0]

    expect(layoutSource).toMatch(/class="ui-main-shell[^"]*"[\s\S]*?<AppHeader/)
    expect(headerSource).toContain('<header class="app-topbar sticky top-0 z-30">')
    expect(headerSource).not.toContain('app-topbar glass')
    expect(topbarStyle).toContain('position: sticky !important;')
    expect(topbarStyle).toContain('width: 100%;')
    expect(topbarStyle).toContain('margin: 0 !important;')
    expect(topbarStyle).toContain('border-radius: 0 !important;')
    expect(topbarStyle).toContain('box-shadow: none !important;')
    expect(topbarStyle).toContain('backdrop-filter: none !important;')
  })
})
