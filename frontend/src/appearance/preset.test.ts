import { describe, expect, it } from 'vitest'
import { encodePreset } from 'shadcn/preset'
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

  it('maps current shadcn mist, medium, and non-neutral theme values safely', () => {
    const code = encodePreset({ baseColor: 'mist', chartColor: 'mist', theme: 'blue', radius: 'medium' })
    const result = decodeRuntimeAppearance(code)

    expect(result.compatibility.unavailable).toEqual([])
    expect(result.compatibility.applied).toEqual(expect.arrayContaining([
      'baseColor:mist',
      'chartColor:mist',
      'radius:medium',
    ]))
    expect(result.compatibility.deploymentControlled).toContain('theme:blue → neutral grayscale')
    expect(result.profile.baseColor).toBe('mist')
    expect(result.profile.chartColor).toBe('muted')
    expect(result.profile.radius).toBe('medium')
    expect(result.tokens.light['--ui-radius']).toBe('0.5rem')
  })
})
