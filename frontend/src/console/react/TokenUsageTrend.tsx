import type { TrendDataPoint } from '@/types'
import { readUsageNumber } from '@/utils/usageDisplay'
import {
  CartesianGrid,
  Line,
  LineChart,
  XAxis,
  YAxis,
} from 'recharts'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from './ui/chart'
import { Skeleton } from './ui/skeleton'

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
  cacheHitRate: number
}

const chartConfig = {
  input: { label: 'Input', color: 'hsl(var(--chart-1))' },
  output: { label: 'Output', color: 'hsl(var(--chart-2))' },
  cacheCreation: { label: 'Cache Creation', color: 'hsl(var(--chart-3))' },
  cacheRead: { label: 'Cache Read', color: 'hsl(var(--chart-4))' },
  cacheHitRate: { label: 'Cache Hit Rate', color: 'hsl(var(--chart-5))' },
} satisfies ChartConfig

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)}B`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(2)}K`
  return value.toLocaleString()
}

const normalizeTrend = (trendData?: TrendDataPoint[] | null): TrendPoint[] => (
  (trendData || []).map((point) => {
    const source = point as unknown as Record<string, unknown>
    const input = readUsageNumber(point, ['input_tokens', 'prompt_tokens'])
    const cacheCreation = readUsageNumber(point, ['cache_creation_tokens', 'cache_write_tokens'])
    const cacheRead = readUsageNumber(point, ['cache_read_tokens', 'cached_tokens'])
    const promptTokens = input + cacheCreation + cacheRead
    return {
      date: String(source.date ?? source.period ?? source.timestamp ?? ''),
      input,
      output: readUsageNumber(point, ['output_tokens', 'completion_tokens']),
      cacheCreation,
      cacheRead,
      cacheHitRate: promptTokens > 0 ? (cacheRead / promptTokens) * 100 : 0,
    }
  })
)

function TrendChart({ points }: { points: TrendPoint[] }) {
  return (
    <ChartContainer config={chartConfig} className="h-64 w-full aspect-auto">
      <LineChart accessibilityLayer data={points} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} />
        <XAxis dataKey="date" tickLine={false} axisLine={false} tickMargin={10} minTickGap={24} />
        <YAxis yAxisId="tokens" tickLine={false} axisLine={false} tickFormatter={formatTokens} width={48} />
        <YAxis yAxisId="percent" hide domain={[0, 100]} />
        <ChartTooltip content={<ChartTooltipContent indicator="line" />} />
        <ChartLegend content={<ChartLegendContent />} />
        <Line yAxisId="tokens" dataKey="input" type="monotone" stroke="var(--color-input)" strokeWidth={2} dot={false} />
        <Line yAxisId="tokens" dataKey="output" type="monotone" stroke="var(--color-output)" strokeWidth={2} dot={false} />
        <Line yAxisId="tokens" dataKey="cacheCreation" type="monotone" stroke="var(--color-cacheCreation)" strokeWidth={2} dot={false} />
        <Line yAxisId="tokens" dataKey="cacheRead" type="monotone" stroke="var(--color-cacheRead)" strokeWidth={2} dot={false} />
        <Line yAxisId="percent" dataKey="cacheHitRate" type="monotone" stroke="var(--color-cacheHitRate)" strokeWidth={2} strokeDasharray="5 5" dot={false} />
      </LineChart>
    </ChartContainer>
  )
}

export default function TokenUsageTrend({ trendData, loading = false, title, noData }: TokenUsageTrendProps) {
  const points = normalizeTrend(trendData)

  return (
    <Card className="@container/card min-w-0 overflow-hidden">
      <CardHeader className="pb-4">
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent className="pt-0">
        {loading ? (
          <div className="space-y-3" role="status" aria-label="Loading">
            <Skeleton className="h-64 w-full" />
          </div>
        ) : points.length > 0 ? (
          <TrendChart points={points} />
        ) : (
          <div className="flex h-64 items-center justify-center text-sm text-muted-foreground">{noData}</div>
        )}
      </CardContent>
    </Card>
  )
}
