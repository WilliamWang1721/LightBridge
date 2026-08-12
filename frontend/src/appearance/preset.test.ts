import { describe, expect, it } from 'vitest'
import { decodeRuntimeAppearance, runtimeAppearanceFromSettings } from './preset'

describe('runtime appearance preset', () => {
  it('decodes the official default preset into project-owned tokens', () => {
    const result = decodeRuntimeAppearance(' b0 ')
    expect(result.preset.style).toBe('nova')
    expect(result.profile.baseColor).toBe('neutral')
    expect(result.tokens.light['--background']).toBe('0 0% 98%')
    expect(result.tokens.dark['--background']).toBe('0 0% 8%')
  })

  it('fails closed for malformed or unsupported codes', () => {
    expect(() => decodeRuntimeAppearance('not-a-preset')).toThrow()
    expect(runtimeAppearanceFromSettings('not-a-preset')).toBeNull()
  })
})
