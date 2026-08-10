import { beforeEach, describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({
  getUIProfile: vi.fn(),
  updateUIProfile: vi.fn(),
  resetUIProfile: vi.fn(),
}))

vi.mock('@/api/uiProfile', () => ({ default: api }))

describe('useUIPlatform account hydration', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    localStorage.clear()
    document.getElementById('lightbridge-ui-portal')?.remove()
    for (const attribute of Array.from(document.documentElement.attributes)) {
      if (attribute.name.startsWith('data-ui-')) document.documentElement.removeAttribute(attribute.name)
    }
  })

  it('keeps edits live when stale account hydration finishes and then synchronizes them', async () => {
    let finishHydration!: (value: Record<string, string>) => void
    api.getUIProfile.mockReturnValue(new Promise((resolve) => {
      finishHydration = resolve
    }))
    api.updateUIProfile.mockImplementation(async (preferences) => preferences)

    const { useUIPlatform } = await import('./useUIPlatform')
    const ui = useUIPlatform()
    const hydration = ui.hydrateAccountPreferences()
    await vi.waitFor(() => expect(api.getUIProfile).toHaveBeenCalledOnce())

    await ui.updatePreferences({
      radius: 'large',
      density: 'compact',
      motion: 'reduced',
      iconLibrary: 'classic',
      chartColor: 'vivid',
    })

    expect(document.documentElement.dataset.uiRadius).toBe('large')
    expect(document.documentElement.dataset.uiStyle).toBe('luma')
    expect(document.documentElement.dataset.uiDensity).toBe('compact')
    expect(document.documentElement.dataset.uiMotion).toBe('reduced')
    expect(document.documentElement.dataset.uiIcons).toBe('classic')
    expect(document.documentElement.dataset.uiChartColor).toBe('vivid')

    finishHydration({
      radius: 'small',
      density: 'comfortable',
      motion: 'expressive',
      iconLibrary: 'lucide',
      chartColor: 'muted',
    })
    await hydration

    expect(api.updateUIProfile).toHaveBeenCalledWith(expect.objectContaining({
      radius: 'large',
      density: 'compact',
      motion: 'reduced',
      iconLibrary: 'classic',
      chartColor: 'vivid',
    }))
    expect(document.documentElement.dataset.uiRadius).toBe('large')
    expect(document.documentElement.dataset.uiStyle).toBe('luma')
    expect(document.documentElement.dataset.uiDensity).toBe('compact')
    expect(document.documentElement.dataset.uiMotion).toBe('reduced')
    expect(document.documentElement.dataset.uiIcons).toBe('classic')
    expect(document.documentElement.dataset.uiChartColor).toBe('vivid')
  })

  it('keeps a reset live when stale account hydration finishes', async () => {
    let finishHydration!: (value: Record<string, string>) => void
    api.getUIProfile.mockReturnValue(new Promise((resolve) => {
      finishHydration = resolve
    }))
    api.updateUIProfile.mockImplementation(async (preferences) => preferences)
    localStorage.setItem('lightbridge.ui.profile.v1', JSON.stringify({ radius: 'large' }))

    const { useUIPlatform } = await import('./useUIPlatform')
    const ui = useUIPlatform()
    const hydration = ui.hydrateAccountPreferences()
    await vi.waitFor(() => expect(api.getUIProfile).toHaveBeenCalledOnce())

    await ui.resetPreferences()
    expect(document.documentElement.dataset.uiRadius).toBe('default')

    finishHydration({ radius: 'small' })
    await hydration

    expect(api.updateUIProfile).toHaveBeenCalledWith({})
    expect(document.documentElement.dataset.uiRadius).toBe('default')
    expect(localStorage.getItem('lightbridge.ui.profile.v1')).toBeNull()
  })
})
