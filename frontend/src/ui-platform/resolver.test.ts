import { describe, expect, it } from 'vitest'
import { DEFAULT_UI_PROFILE } from './registry'
import { parseStoredUIProfile, resolveUIProfile, sanitizeUIProfileOverrides } from './resolver'

describe('resolveUIProfile', () => {
  it('uses LightBridge Luma as the built-in default while preserving Legacy as an explicit mode', () => {
    const { profile } = resolveUIProfile()

    expect(profile).toEqual(DEFAULT_UI_PROFILE)
    expect(profile.mode).toBe('modern')
    expect(profile.componentStyle).toBe('luma')

    const legacy = resolveUIProfile({ userPreferences: { mode: 'legacy' } }).profile
    expect(legacy.mode).toBe('legacy')
    expect(legacy.componentStyle).toBe('luma')
  })

  it('keeps Luma fixed while applying independent user preferences', () => {
    const { profile, warnings } = resolveUIProfile({
      userPreferences: {
        mode: 'modern',
        baseColor: 'zinc',
        chartColor: 'vivid',
        font: 'system',
        radius: 'large',
        menuSize: 'wide',
      },
    })

    expect(profile.componentStyle).toBe('luma')
    expect(profile.mode).toBe('modern')
    expect(profile.baseColor).toBe('zinc')
    expect(profile.chartColor).toBe('vivid')
    expect(profile.font).toBe('system')
    expect(profile.radius).toBe('large')
    expect(profile.menuSize).toBe('wide')
    expect(warnings).toEqual([])
  })

  it('applies configuration layers in precedence order', () => {
    const { profile } = resolveUIProfile({
      adminDefaults: { radius: 'small', menu: 'subtle' },
      packageDefaults: { radius: 'large' },
      localPreferences: { menu: 'outlined' },
      userPreferences: { radius: 'default' },
      previewOverrides: { menu: 'translucent' },
    })

    expect(profile.radius).toBe('default')
    expect(profile.menu).toBe('translucent')
  })

  it('falls back for unsupported values', () => {
    const { profile, warnings } = resolveUIProfile({
      userPreferences: {
        baseColor: 'unknown',
        iconLibrary: 'unknown',
      },
    })

    expect(profile.baseColor).toBe(DEFAULT_UI_PROFILE.baseColor)
    expect(profile.iconLibrary).toBe(DEFAULT_UI_PROFILE.iconLibrary)
    expect(warnings).toHaveLength(2)
  })

  it('requires an active package for package mode', () => {
    const { profile, warnings } = resolveUIProfile({
      userPreferences: { mode: 'package' },
    })

    expect(profile.mode).toBe('modern')
    expect(warnings).toContain('Package mode requires an active package; using modern mode.')
  })
})

describe('parseStoredUIProfile', () => {
  it('ignores attempts to override the component style', () => {
    const value = parseStoredUIProfile(
      JSON.stringify({ componentStyle: 'other', mode: 'modern', baseColor: 'stone' }),
    )

    expect(value).toEqual({ mode: 'modern', baseColor: 'stone' })
  })

  it('returns undefined for invalid JSON', () => {
    expect(parseStoredUIProfile('{')).toBeUndefined()
  })

  it('keeps only known string fields and rejects dangerous package ids', () => {
    const value = sanitizeUIProfileOverrides({
      mode: ' modern ',
      baseColor: 'stone',
      activePackageId: '__proto__',
      unknown: 'value',
      radius: 42,
      __proto__: { polluted: true },
    })

    expect(value).toEqual({ mode: 'modern', baseColor: 'stone' })
    expect(({} as Record<string, unknown>).polluted).toBeUndefined()
  })
})
