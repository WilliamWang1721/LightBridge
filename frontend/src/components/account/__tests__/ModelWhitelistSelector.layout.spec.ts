import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  fileURLToPath(new URL('../ModelWhitelistSelector.vue', import.meta.url)),
  'utf-8'
)

describe('ModelWhitelistSelector action buttons', () => {
  it('uses one equal-width responsive grid for all model actions', () => {
    expect(source).toContain('data-testid="model-action-button-group"')
    expect(source).toContain('grid-cols-1 gap-2 sm:grid-cols-3')

    for (const testId of [
      'fill-related-models-button',
      'sync-upstream-models-button',
      'clear-all-models-button'
    ]) {
      expect(source).toContain(`data-testid="${testId}"`)
    }

    expect(source.match(/min-h-10 w-full min-w-0 whitespace-nowrap/g)).toHaveLength(3)
  })

  it('uses the shared button system instead of three unrelated button styles', () => {
    expect(source).toContain('class="btn btn-secondary min-h-10')
    expect(source).toContain('class="btn btn-primary min-h-10')
    expect(source).toContain('border-red-200 text-red-600')
    expect(source).not.toContain('sm:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]')
  })
})
