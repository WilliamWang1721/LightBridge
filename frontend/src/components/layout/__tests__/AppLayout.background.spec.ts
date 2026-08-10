import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../../..')
const classicStyles = readFileSync(resolve(root, 'src/style.css'), 'utf8')
const layoutStyles = readFileSync(resolve(root, 'src/styles/ui-layout.css'), 'utf8')

describe('Luma application background', () => {
  it('uses the classic page background token for the right-hand shell', () => {
    expect(classicStyles).toMatch(/--lb-bg-page:\s*#ffffff;/)
    expect(layoutStyles).toMatch(
      /html\[data-ui-mode='modern'\] \.ui-main-shell,[\s\S]*?background:\s*var\(--lb-bg-page\);/,
    )
  })
})
