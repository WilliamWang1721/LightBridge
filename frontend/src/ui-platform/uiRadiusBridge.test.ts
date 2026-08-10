import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const bridge = readFileSync(
  resolve(process.cwd(), 'src/styles/ui-modern-tailwind-bridge-parts/part-01.css'),
  'utf8',
)

describe('Luma radius bridge', () => {
  it('maps legacy square controls and rounded-none utilities to the shared radius token', () => {
    expect(bridge).toMatch(
      /:is\(input, select, textarea, button, \.btn, \.dropdown-item\)[\s\S]*?border-radius: var\(--ui-radius\) !important;/,
    )
    expect(bridge).toMatch(
      /\[class~='rounded-none'\][\s\S]*?border-radius: var\(--ui-radius\) !important;/,
    )
    expect(bridge).not.toMatch(
      /\[class~='rounded-none'\]\s*\{\s*border-radius:\s*0\s*!important;/,
    )
  })
})
