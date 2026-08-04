import { describe, expect, it } from 'vitest'
import {
  buildThemeConfigDraft,
  getThemeConfigFields,
  themeAssetURL,
  validateThemeConfig,
} from './themePackages'

const manifest = {
  config: [
    { key: 'primary_color', label: 'Primary color', type: 'color', default: '#2563eb' },
    { key: 'radius', label: 'Radius', type: 'number', default: 8, min: 0, max: 24 },
    { key: 'menu', label: 'Menu', type: 'select', default: 'default', options: ['default', 'compact'] },
    { key: 'glass', label: 'Glass', type: 'boolean', default: true },
  ],
}

describe('theme package schema', () => {
  it('builds a typed draft from manifest defaults and stored values', () => {
    expect(buildThemeConfigDraft(manifest, { radius: '12', glass: false })).toEqual({
      primary_color: '#2563eb',
      radius: 12,
      menu: 'default',
      glass: false,
    })
  })

  it('validates package-provided constraints', () => {
    const result = validateThemeConfig(manifest, {
      primary_color: 'red',
      radius: 40,
      menu: 'unknown',
      glass: true,
    })

    expect(result.errors.primary_color).toBeTruthy()
    expect(result.errors.radius).toBeTruthy()
    expect(result.value.menu).toBe('default')
  })

  it('ignores malformed fields', () => {
    expect(getThemeConfigFields({ config: [{ key: '', label: '', type: 'script' }] })).toEqual([])
  })

  it('builds an encoded package asset URL', () => {
    expect(themeAssetURL('modern light', 'images/preview image.png')).toBe(
      '/ui-themes/modern%20light/images/preview%20image.png',
    )
  })
})
