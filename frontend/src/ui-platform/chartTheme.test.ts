import { afterEach, describe, expect, it } from 'vitest'
import { Chart as ChartJS } from 'chart.js'
import { disposeChartTheme, hslVariable, hslWithAlpha, initializeChartTheme, readUIChartTheme, refreshUICharts } from './chartTheme'

afterEach(() => {
  disposeChartTheme()
  document.documentElement.removeAttribute('style')
  document.documentElement.removeAttribute('data-ui-mode')
})

describe('UI chart theme', () => {
  it('converts semantic HSL values into canvas-safe colors', () => {
    expect(hslVariable('173 80% 40%')).toBe('hsl(173 80% 40%)')
    expect(hslVariable('173 80% 40%', 0.2)).toBe('hsl(173 80% 40% / 0.2)')
    expect(hslWithAlpha('hsl(173 80% 40%)', 0.14)).toBe('hsl(173 80% 40% / 0.14)')
  })

  it('initializes safely before optional Chart.js plugins are registered', () => {
    const plugins = ChartJS.defaults.plugins as Record<string, unknown>
    const legend = plugins.legend
    const tooltip = plugins.tooltip
    delete plugins.legend
    delete plugins.tooltip
    document.documentElement.dataset.uiMode = 'modern'

    expect(() => initializeChartTheme()).not.toThrow()
    expect(() => refreshUICharts()).not.toThrow()
    disposeChartTheme()

    if (legend !== undefined) plugins.legend = legend
    if (tooltip !== undefined) plugins.tooltip = tooltip
  })

  it('restores global defaults after leaving modern mode', () => {
    const originalColor = ChartJS.defaults.color
    document.documentElement.dataset.uiMode = 'modern'
    document.documentElement.style.setProperty('--muted-foreground', '10 20% 30%')
    initializeChartTheme()
    expect(ChartJS.defaults.color).toBe('hsl(10 20% 30%)')

    document.documentElement.dataset.uiMode = 'legacy'
    refreshUICharts()
    expect(ChartJS.defaults.color).toBe(originalColor)
  })

  it('reads the active UI palette and surface tokens', () => {
    const root = document.documentElement
    root.dataset.uiMode = 'package'
    root.style.setProperty('--chart-1', '10 70% 50%')
    root.style.setProperty('--chart-2', '20 70% 50%')
    root.style.setProperty('--chart-3', '30 70% 50%')
    root.style.setProperty('--chart-4', '40 70% 50%')
    root.style.setProperty('--chart-5', '50 70% 50%')
    root.style.setProperty('--foreground', '220 20% 10%')
    root.style.setProperty('--muted-foreground', '220 10% 45%')
    root.style.setProperty('--border', '220 10% 85%')
    root.style.setProperty('--card', '0 0% 100%')
    root.style.setProperty('--popover', '0 0% 99%')
    root.style.setProperty('--popover-foreground', '220 20% 10%')
    root.style.setProperty('--ui-font-body', 'Inter Variable, Inter, sans-serif')

    const theme = readUIChartTheme(root)

    expect(theme.mode).toBe('package')
    expect(theme.palette).toEqual([
      'hsl(10 70% 50%)',
      'hsl(20 70% 50%)',
      'hsl(30 70% 50%)',
      'hsl(40 70% 50%)',
      'hsl(50 70% 50%)',
    ])
    expect(theme.mutedForeground).toBe('hsl(220 10% 45%)')
    expect(theme.fontFamily).toBe('Inter Variable, Inter, sans-serif')
  })
})
