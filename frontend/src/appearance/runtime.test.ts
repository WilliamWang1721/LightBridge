import { describe, expect, it } from 'vitest'
import { applyRuntimeAppearance } from './runtime'

describe('runtime appearance application', () => {
  it('writes allowlisted tokens to the root and portal scopes', () => {
    applyRuntimeAppearance('b0')
    const style = document.getElementById('lightbridge-runtime-appearance')

    expect(document.documentElement.dataset.uiAppearance).toBe('b0')
    expect(style?.textContent).toContain(':root[data-ui-appearance] #lightbridge-ui-portal')
    expect(style?.textContent).toContain('--background: 0 0% 98%')
  })

  it('removes the runtime override when reset', () => {
    applyRuntimeAppearance(null)

    expect(document.documentElement.dataset.uiAppearance).toBeUndefined()
    expect(document.getElementById('lightbridge-runtime-appearance')).toBeNull()
  })
})
