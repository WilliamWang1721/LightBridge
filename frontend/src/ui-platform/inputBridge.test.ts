import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const css = readFileSync(resolve(process.cwd(), 'src/styles/ui-modern-compat-parts/part-03.css'), 'utf8')

describe('modern input compatibility bridge', () => {
  it('does not apply text-field geometry to non-text native controls', () => {
    expect(css).not.toContain(':is(input, select')
    expect(css).not.toContain(':is(input, select, textarea')
    expect(css).toContain('input:not([type])')
    expect(css).toContain("[type='text']")
    expect(css).not.toContain("[type='checkbox']")
    expect(css).not.toContain("[type='radio']")
    expect(css).not.toContain("[type='file']")
  })
})
