import { describe, expect, it } from 'vitest'
import { resolveAppIcon } from './icons'

describe('icon providers', () => {
  it('supports Lucide and LightBridge Outline providers', () => {
    const lucide = resolveAppIcon('settings', 'lucide')
    const classic = resolveAppIcon('settings', 'classic')

    expect(lucide).toBeTruthy()
    expect(classic).toBeTruthy()
    expect(classic).not.toBe(lucide)
  })

  it('falls back to Lucide for unknown provider ids', () => {
    expect(resolveAppIcon('home', 'missing')).toBe(resolveAppIcon('home', 'lucide'))
  })
})
