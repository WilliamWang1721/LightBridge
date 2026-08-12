import { type ReactNode } from 'react'
import type { TrendDataPoint } from '@/types'
import { readUsageNumber } from '@/utils/usageDisplay'
import { createShadcnElement as h } from './ui/createElement'

const chartWidth = 640
const chartHeight = 220
const chartPadding = { top: 12, right: 12, bottom: 28, left: 38 }

export interface TokenUsageTrendProps {
  trendData?: TrendDataPoint[] | null
  loading?: boolean
  title: string
  noData: string
}

interface TrendPoint {
  date: string
  input: number
  output: number
  cacheCreation: number
  cacheRead: number
  cost: number
  actualCost: number
}

interface TrendSeries {
  key: keyof Pick<TrendPoint, 'input' | 'output' | 'cacheCreation' | 'cacheRead'>
  label: string
  color: string
  dashed?: boolean
}

const series: TrendSeries[] = [
  { key: 'input', label: 'Input', color: 'hsl(var(--foreground))' },
  { key: 'output', label: 'Output', color: 'hsl(var(--muted-foreground))' },
  { key: 'cacheCreation', label: 'Cache Creation', color: 'hsl(var(--border))' },
  { key: 'cacheRead', label: 'Cache Read', color: 'hsl(var(--muted-foreground) / 0.7)' },
]

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)}B`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(2)}K`
  return value.toLocaleString()
}

const formatCost = (value: number): string => {
  if (value >= 1000) return `${(value / 1000).toFixed(2)}K`
  if (value >= 1) return value.toFixed(2)
  if (value >= 0.01) return value.toFixed(3)
  return value.toFixed(4)
}

const normalizeTrend = (trendData?: TrendDataPoint[] | null): TrendPoint[] => (
  (trendData || []).map((point) => {
    const source = point as unknown as Record<string, unknown>
    return {
      date: String(source.date ?? source.period ?? source.timestamp ?? ''),
      input: readUsageNumber(point, ['input_tokens', 'prompt_tokens']),
      output: readUsageNumber(point, ['output_tokens', 'completion_tokens']),
      cacheCreation: readUsageNumber(point, ['cache_creation_tokens', 'cache_write_tokens']),
      cacheRead: readUsageNumber(point, ['cache_read_tokens', 'cached_tokens']),
      cost: readUsageNumber(point, ['cost', 'total_cost', 'standard_cost']),
      actualCost: readUsageNumber(point, ['actual_cost', 'user_cost']),
    }
  })
)

const pathFor = (values: number[], max: number): string => {
  const width = chartWidth - chartPadding.left - chartPadding.right
  const height = chartHeight - chartPadding.top - chartPadding.bottom
  return values.map((value, index) => {
    const x = chartPadding.left + (values.length <= 1 ? width / 2 : (index / (values.length - 1)) * width)
    const y = chartPadding.top + (1 - value / max) * height
    return `${index === 0 ? 'M' : 'L'} ${x.toFixed(2)} ${y.toFixed(2)}`
  }).join(' ')
}

function Chart({ points }: { points: TrendPoint[] }): ReactNode {
  const tokenMax = Math.max(1, ...points.flatMap((point) => series.map(({ key }) => point[key])))
  const xStep = points.length <= 1
    ? (chartWidth - chartPadding.left - chartPadding.right) / 2
    : (chartWidth - chartPadding.left - chartPadding.right) / (points.length - 1)
  const tokenHeight = chartHeight - chartPadding.top - chartPadding.bottom
  const percentPath = points.map((point) => {
    const promptTokens = point.input + point.cacheRead + point.cacheCreation
    return promptTokens > 0 ? (point.cacheRead / promptTokens) * 100 : 0
  })
  const percentPathData = percentPath.map((value, index) => {
    const x = chartPadding.left + (points.length <= 1 ? xStep : index * xStep)
    const y = chartPadding.top + (1 - value / 100) * tokenHeight
    return `${index === 0 ? 'M' : 'L'} ${x.toFixed(2)} ${y.toFixed(2)}`
  }).join(' ')
  const gridLines = [0, 0.25, 0.5, 0.75, 1]

  return h('div', { className: 'space-y-2' }, [
    h('div', { key: 'legend', className: 'flex flex-wrap gap-x-4 gap-y-1 text-[10px] text-[hsl(var(--muted-foreground))]' }, [
      ...series.map((item) => h('span', { key: item.key, className: 'inline-flex items-center gap-1' }, [
        h('span', { key: 'dot', className: 'h-2 w-2 rounded-full', style: { backgroundColor: item.color } }),
        item.label,
      ])),
      h('span', { key: 'hit-rate', className: 'inline-flex items-center gap-1' }, [
        h('span', { key: 'dot', className: 'h-2 w-2 rounded-full bg-[hsl(var(--foreground)/0.45)]' }),
        'Cache Hit Rate',
      ]),
    ]),
    h('svg', {
      key: 'svg',
      className: 'h-48 w-full overflow-visible',
      viewBox: `0 0 ${chartWidth} ${chartHeight}`,
      role: 'img',
      'aria-label': 'Token usage trend chart',
    }, [
      ...gridLines.map((ratio) => {
        const y = chartPadding.top + ratio * tokenHeight
        return h('line', {
          key: `grid-${ratio}`,
          x1: chartPadding.left,
          x2: chartWidth - chartPadding.right,
          y1: y,
          y2: y,
          stroke: 'hsl(var(--border) / 0.65)',
          strokeWidth: 1,
        })
      }),
      ...series.map((item) => h('path', {
        key: item.key,
        d: pathFor(points.map((point) => point[item.key]), tokenMax),
        fill: 'none',
        stroke: item.color,
        strokeWidth: 2,
        strokeLinecap: 'round',
        strokeLinejoin: 'round',
      })),
      h('path', {
        key: 'cache-hit-rate',
        d: percentPathData,
        fill: 'none',
        stroke: 'hsl(var(--foreground) / 0.45)',
        strokeWidth: 1.5,
        strokeDasharray: '5 5',
        strokeLinecap: 'round',
      }),
      ...series.flatMap((item) => points.map((point, index) => {
        const value = point[item.key]
        const x = chartPadding.left + (points.length <= 1 ? xStep : index * xStep)
        const y = chartPadding.top + (1 - value / tokenMax) * tokenHeight
        return h('circle', { key: `${item.key}-${index}`, cx: x, cy: y, r: 2.5, fill: item.color }, h('title', {}, `${item.label}: ${formatTokens(value)} | Actual: $${formatCost(point.actualCost)} | Standard: $${formatCost(point.cost)}`))
      })),
      ...points.map((_, index) => {
        const x = chartPadding.left + (points.length <= 1 ? xStep : index * xStep)
        const value = percentPath[index]
        const y = chartPadding.top + (1 - value / 100) * tokenHeight
        return h('circle', { key: `cache-hit-${index}`, cx: x, cy: y, r: 2, fill: 'hsl(var(--foreground) / 0.45)' }, h('title', {}, `Cache Hit Rate: ${value.toFixed(1)}%`))
      }),
    ]),
    h('div', { key: 'labels', className: 'flex justify-between gap-2 overflow-hidden text-[10px] text-[hsl(var(--muted-foreground))]' }, points.map((point) => h('span', { key: point.date, className: 'min-w-0 truncate' }, point.date))),
  ])
}

export default function TokenUsageTrend({ trendData, loading = false, title, noData }: TokenUsageTrendProps) {
  const points = normalizeTrend(trendData)
  let body: ReactNode

  if (loading) {
    body = h('div', { className: 'flex h-48 items-center justify-center', role: 'status', 'aria-label': 'Loading' }, h('div', { className: 'h-5 w-5 animate-spin rounded-full border-2 border-[hsl(var(--muted-foreground))] border-t-transparent' }))
  } else if (points.length > 0) {
    body = h(Chart, { points })
  } else {
    body = h('div', { className: 'flex h-48 items-center justify-center text-sm text-[hsl(var(--muted-foreground))]' }, noData)
  }

  return h('section', { className: 'card min-w-0 overflow-hidden p-4' }, [
    h('h3', { key: 'title', className: 'mb-4 text-sm font-semibold text-[hsl(var(--foreground))]' }, title),
    h('div', { key: 'body' }, body),
  ])
}
