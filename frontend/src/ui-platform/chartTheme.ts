import {
  Chart as ChartJS,
  type Chart,
  type ChartDataset,
  type ChartType,
  type Plugin,
} from 'chart.js'
import { UI_PLATFORM_EVENT } from './registry'

export interface UIChartTheme {
  mode: string
  palette: string[]
  foreground: string
  mutedForeground: string
  border: string
  background: string
  surface: string
  popover: string
  popoverForeground: string
  destructive: string
  fontFamily: string
}

type MutableRecord = Record<string, unknown>
type MutableDataset = ChartDataset<ChartType> & MutableRecord
type PropertySnapshot = { exists: boolean; value: unknown }
type ObjectSnapshot = Record<string, PropertySnapshot>

type ChartSnapshot = {
  options: ObjectSnapshot
  scales: Array<{ target: MutableRecord; snapshot: ObjectSnapshot }>
  pluginContainer?: {
    target: MutableRecord
    snapshot: ObjectSnapshot
  }
}

const datasetProperties = [
  'backgroundColor',
  'borderColor',
  'pointBackgroundColor',
  'pointBorderColor',
  'hoverBackgroundColor',
  'hoverBorderColor',
] as const

const datasetSnapshots = new WeakMap<object, ObjectSnapshot>()
const chartSnapshots = new WeakMap<Chart, ChartSnapshot>()
let globalDefaultsSnapshot: ChartSnapshot | undefined
let initialized = false
let darkModeObserver: MutationObserver | undefined

function cloneValue<T>(value: T): T {
  if (Array.isArray(value)) return value.map((item) => cloneValue(item)) as T
  if (value && typeof value === 'object') {
    const prototype = Object.getPrototypeOf(value)
    if (prototype === Object.prototype || prototype === null) {
      return Object.fromEntries(
        Object.entries(value as MutableRecord).map(([key, item]) => [key, cloneValue(item)]),
      ) as T
    }
  }
  return value
}

function captureProperty(target: MutableRecord, key: string): PropertySnapshot {
  return {
    exists: Object.prototype.hasOwnProperty.call(target, key),
    value: cloneValue(target[key]),
  }
}

function captureProperties(target: MutableRecord, keys: readonly string[]): ObjectSnapshot {
  return Object.fromEntries(keys.map((key) => [key, captureProperty(target, key)]))
}

function restoreProperties(target: MutableRecord, snapshot: ObjectSnapshot): void {
  for (const [key, property] of Object.entries(snapshot)) {
    if (property.exists) target[key] = cloneValue(property.value)
    else delete target[key]
  }
}

function cssVariable(style: CSSStyleDeclaration, name: string, fallback: string): string {
  const value = style.getPropertyValue(name).trim()
  return value || fallback
}

export function hslVariable(value: string, alpha?: number): string {
  const normalized = value.trim()
  if (!normalized) return 'transparent'
  if (normalized.startsWith('hsl(') || normalized.startsWith('#') || normalized.startsWith('rgb')) {
    return normalized
  }
  return alpha === undefined ? `hsl(${normalized})` : `hsl(${normalized} / ${alpha})`
}

export function readUIChartTheme(root: HTMLElement = document.documentElement): UIChartTheme {
  const style = getComputedStyle(root)
  const palette = Array.from({ length: 5 }, (_, index) =>
    hslVariable(cssVariable(style, `--chart-${index + 1}`, '173 80% 40%')),
  )

  return {
    mode: root.dataset.uiMode || 'modern',
    palette,
    foreground: hslVariable(cssVariable(style, '--foreground', '222 47% 11%')),
    mutedForeground: hslVariable(cssVariable(style, '--muted-foreground', '215 16% 47%')),
    border: hslVariable(cssVariable(style, '--border', '214 32% 91%'), 0.8),
    background: hslVariable(cssVariable(style, '--background', '210 20% 98%')),
    surface: hslVariable(cssVariable(style, '--card', '0 0% 100%')),
    popover: hslVariable(cssVariable(style, '--popover', '0 0% 100%')),
    popoverForeground: hslVariable(cssVariable(style, '--popover-foreground', '222 47% 11%')),
    destructive: hslVariable(cssVariable(style, '--destructive', '0 72% 51%')),
    fontFamily: cssVariable(style, '--ui-font-body', 'Inter Variable, Inter, system-ui, sans-serif'),
  }
}

function rememberDataset(dataset: MutableDataset): void {
  if (datasetSnapshots.has(dataset)) return
  datasetSnapshots.set(dataset, captureProperties(dataset, datasetProperties))
}

function restoreDataset(dataset: MutableDataset): void {
  const snapshot = datasetSnapshots.get(dataset)
  if (!snapshot) return
  restoreProperties(dataset, snapshot)
  datasetSnapshots.delete(dataset)
}

function rememberChart(chart: Chart): void {
  if (chartSnapshots.has(chart)) return
  const options = chart.options as unknown as MutableRecord
  const snapshot: ChartSnapshot = {
    options: captureProperties(options, ['color', 'font', 'plugins']),
    scales: [],
  }

  const scales = options.scales
  if (scales && typeof scales === 'object') {
    for (const scale of Object.values(scales as MutableRecord)) {
      if (!scale || typeof scale !== 'object') continue
      const target = scale as MutableRecord
      snapshot.scales.push({ target, snapshot: captureProperties(target, ['ticks', 'grid', 'border', 'title']) })
    }
  }

  const plugins = options.plugins
  if (plugins && typeof plugins === 'object') {
    const target = plugins as MutableRecord
    snapshot.pluginContainer = { target, snapshot: captureProperties(target, ['legend', 'tooltip']) }
  }
  chartSnapshots.set(chart, snapshot)
}

function restoreChart(chart: Chart): void {
  const snapshot = chartSnapshots.get(chart)
  chart.data.datasets.forEach((dataset) => restoreDataset(dataset as MutableDataset))
  if (!snapshot) return

  for (const scale of snapshot.scales) restoreProperties(scale.target, scale.snapshot)
  if (snapshot.pluginContainer) {
    restoreProperties(snapshot.pluginContainer.target, snapshot.pluginContainer.snapshot)
  }
  restoreProperties(chart.options as unknown as MutableRecord, snapshot.options)
  chartSnapshots.delete(chart)
}

function paletteForLength(theme: UIChartTheme, length: number): string[] {
  return Array.from({ length }, (_, index) => theme.palette[index % theme.palette.length])
}

function applyDatasetTheme(
  dataset: MutableDataset,
  type: ChartType,
  datasetIndex: number,
  dataLength: number,
  theme: UIChartTheme,
): void {
  rememberDataset(dataset)
  const color = theme.palette[datasetIndex % theme.palette.length]

  if (type === 'doughnut' || type === 'pie' || type === 'polarArea') {
    const colors = paletteForLength(theme, dataLength)
    dataset.backgroundColor = colors
    dataset.borderColor = theme.surface
    dataset.hoverBackgroundColor = colors
    dataset.hoverBorderColor = theme.surface
    return
  }
  if (type === 'line' || type === 'scatter' || type === 'radar') {
    dataset.borderColor = color
    dataset.backgroundColor = hslWithAlpha(color, 0.14)
    dataset.pointBackgroundColor = color
    dataset.pointBorderColor = theme.surface
    dataset.hoverBackgroundColor = color
    dataset.hoverBorderColor = theme.surface
    return
  }
  if (type === 'bubble') {
    dataset.backgroundColor = hslWithAlpha(color, 0.58)
    dataset.borderColor = color
    dataset.hoverBackgroundColor = hslWithAlpha(color, 0.78)
    dataset.hoverBorderColor = color
    return
  }
  dataset.backgroundColor = hslWithAlpha(color, 0.78)
  dataset.borderColor = color
  dataset.hoverBackgroundColor = color
  dataset.hoverBorderColor = color
}

export function hslWithAlpha(color: string, alpha: number): string {
  const match = color.match(/^hsl\((.+)\)$/)
  if (!match) return color
  const body = match[1].split('/')[0].trim()
  return `hsl(${body} / ${alpha})`
}

function applyChartTheme(chart: Chart, theme: UIChartTheme): void {
  const modern = theme.mode === 'modern' || theme.mode === 'package'
  if (!modern) {
    restoreChart(chart)
    return
  }

  rememberChart(chart)
  const chartType = (chart.config as { type?: ChartType }).type || 'line'
  chart.data.datasets.forEach((dataset, datasetIndex) => {
    const mutable = dataset as MutableDataset
    const metaType = chart.getDatasetMeta(datasetIndex).type as ChartType | undefined
    applyDatasetTheme(mutable, metaType || chartType, datasetIndex, dataset.data?.length ?? 0, theme)
  })

  const options = chart.options
  options.color = theme.mutedForeground
  options.font = { ...options.font, family: theme.fontFamily }

  if (options.scales) {
    Object.values(options.scales).forEach((scale) => {
      if (!scale) return
      const mutable = scale as unknown as MutableRecord
      mutable.ticks = { ...(mutable.ticks as MutableRecord | undefined), color: theme.mutedForeground }
      mutable.grid = { ...(mutable.grid as MutableRecord | undefined), color: theme.border }
      mutable.border = { ...(mutable.border as MutableRecord | undefined), color: theme.border }
      if (mutable.title) mutable.title = { ...(mutable.title as MutableRecord), color: theme.foreground }
    })
  }

  const plugins = (options.plugins ||= {})
  plugins.legend = {
    ...plugins.legend,
    labels: {
      ...plugins.legend?.labels,
      color: theme.mutedForeground,
      font: { ...plugins.legend?.labels?.font, family: theme.fontFamily },
    },
  }
  plugins.tooltip = {
    ...plugins.tooltip,
    backgroundColor: theme.popover,
    titleColor: theme.popoverForeground,
    bodyColor: theme.popoverForeground,
    borderColor: theme.border,
    borderWidth: 1,
    titleFont: { ...plugins.tooltip?.titleFont, family: theme.fontFamily },
    bodyFont: { ...plugins.tooltip?.bodyFont, family: theme.fontFamily },
  }
}

function rememberGlobalDefaults(): void {
  if (globalDefaultsSnapshot) return
  const defaults = ChartJS.defaults as unknown as MutableRecord
  const snapshot: ChartSnapshot = {
    options: captureProperties(defaults, ['color', 'borderColor', 'font', 'plugins']),
    scales: [],
  }
  const plugins = defaults.plugins
  if (plugins && typeof plugins === 'object') {
    const target = plugins as MutableRecord
    snapshot.pluginContainer = { target, snapshot: captureProperties(target, ['legend', 'tooltip']) }
  }
  globalDefaultsSnapshot = snapshot
}

function restoreGlobalDefaults(): void {
  const snapshot = globalDefaultsSnapshot
  if (!snapshot) return
  if (snapshot.pluginContainer) restoreProperties(snapshot.pluginContainer.target, snapshot.pluginContainer.snapshot)
  restoreProperties(ChartJS.defaults as unknown as MutableRecord, snapshot.options)
  globalDefaultsSnapshot = undefined
}

function applyChartDefaults(theme: UIChartTheme): void {
  if (theme.mode !== 'modern' && theme.mode !== 'package') {
    restoreGlobalDefaults()
    return
  }
  rememberGlobalDefaults()
  ChartJS.defaults.color = theme.mutedForeground
  ChartJS.defaults.borderColor = theme.border
  ChartJS.defaults.font.family = theme.fontFamily

  const defaults = ChartJS.defaults as unknown as MutableRecord
  const plugins = (defaults.plugins && typeof defaults.plugins === 'object'
    ? defaults.plugins
    : (defaults.plugins = {})) as MutableRecord
  const legend = (plugins.legend && typeof plugins.legend === 'object'
    ? plugins.legend
    : (plugins.legend = {})) as MutableRecord
  const labels = (legend.labels && typeof legend.labels === 'object'
    ? legend.labels
    : (legend.labels = {})) as MutableRecord
  labels.color = theme.mutedForeground

  const tooltip = (plugins.tooltip && typeof plugins.tooltip === 'object'
    ? plugins.tooltip
    : (plugins.tooltip = {})) as MutableRecord
  tooltip.backgroundColor = theme.popover
  tooltip.titleColor = theme.popoverForeground
  tooltip.bodyColor = theme.popoverForeground
  tooltip.borderColor = theme.border
  tooltip.borderWidth = 1
}

const uiThemePlugin: Plugin = {
  id: 'lightbridge-ui-theme',
  beforeUpdate(chart) {
    if (typeof document === 'undefined') return
    applyChartTheme(chart, readUIChartTheme())
  },
}

export function refreshUICharts(): void {
  if (typeof document === 'undefined') return
  const theme = readUIChartTheme()
  applyChartDefaults(theme)
  Object.values(ChartJS.instances).forEach((chart) => {
    applyChartTheme(chart, theme)
    chart.update('none')
  })
}

export function initializeChartTheme(): void {
  if (initialized || typeof window === 'undefined' || typeof document === 'undefined') return
  initialized = true
  ChartJS.register(uiThemePlugin)
  refreshUICharts()
  window.addEventListener(UI_PLATFORM_EVENT, refreshUICharts)
  darkModeObserver = new MutationObserver((mutations) => {
    if (mutations.some((mutation) => mutation.attributeName === 'class')) refreshUICharts()
  })
  darkModeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
}

export function disposeChartTheme(): void {
  if (!initialized || typeof window === 'undefined') return
  initialized = false
  window.removeEventListener(UI_PLATFORM_EVENT, refreshUICharts)
  darkModeObserver?.disconnect()
  darkModeObserver = undefined
  ChartJS.unregister(uiThemePlugin)
  restoreGlobalDefaults()
  Object.values(ChartJS.instances).forEach((chart) => {
    restoreChart(chart)
    chart.update('none')
  })
}
