import * as React from 'react'
import {
  Activity,
  ChartNoAxesColumn,
  Database,
  Gauge,
  KeyRound,
  Server,
  UserPlus,
  Zap,
  type LucideIcon,
} from 'lucide-react'
import type { UserUsageTrendPoint } from '@/types'
import type { DashboardStatCardData } from './DashboardStats'
import ModelDistribution, { type ModelDistributionPageProps } from './ModelDistribution'
import TokenUsageTrend, { type TokenUsageTrendProps } from './TokenUsageTrend'
import { Badge } from './ui/badge'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from './ui/card'
import { Skeleton } from './ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from './ui/table'

const metricIcons: Record<string, LucideIcon> = {
  apiKeys: KeyRound,
  accounts: Server,
  todayRequests: ChartNoAxesColumn,
  users: UserPlus,
  todayTokens: Database,
  totalTokens: Database,
  performance: Zap,
  avgResponse: Gauge,
}

export interface DashboardPageLabels {
  recentUsage: string
  userUsageTrend: string
  requests: string
  tokens: string
  noData: string
  userPrefix: (id: number) => string
}

export interface DashboardPageProps {
  loading: boolean
  cards: readonly DashboardStatCardData[]
  modelDistribution: ModelDistributionPageProps
  tokenUsageTrend: TokenUsageTrendProps
  userTrend: readonly UserUsageTrendPoint[]
  userTrendLoading: boolean
  labels: DashboardPageLabels
}

function SectionCards({ cards, loading }: Pick<DashboardPageProps, 'cards' | 'loading'>) {
  const visibleCards = loading && cards.length === 0 ? Array.from({ length: 8 }, (_, index) => ({ key: `loading-${index}` })) : cards

  return (
    <div className="*:data-[slot=card]:shadow-xs grid grid-cols-1 gap-3 px-3 *:data-[slot=card]:bg-gradient-to-t *:data-[slot=card]:from-primary/5 *:data-[slot=card]:to-card sm:grid-cols-2 dark:*:data-[slot=card]:bg-card lg:grid-cols-4">
      {visibleCards.map((card) => {
        if (!('label' in card)) {
          return (
            <Card key={card.key} className="@container/card gap-4 py-4">
              <CardHeader className="gap-3 px-4">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-8 w-32" />
              </CardHeader>
              <CardFooter className="flex-col items-start gap-2 px-4">
                <Skeleton className="h-4 w-36" />
                <Skeleton className="h-4 w-24" />
              </CardFooter>
            </Card>
          )
        }

        const MetricIcon = metricIcons[card.key] || Activity
        return (
          <Card key={card.key} className="@container/card gap-4 py-4">
            <CardHeader className="relative px-4">
              <CardDescription className="text-base font-medium">{card.label}</CardDescription>
              <CardTitle className="text-3xl font-semibold tabular-nums">
                {card.value}
              </CardTitle>
              <div className="absolute right-4 top-4">
                <Badge variant="outline" className="flex gap-1 rounded-lg text-xs">
                  <MetricIcon className="size-3" aria-hidden="true" />
                  <span className="sr-only">{card.label}</span>
                </Badge>
              </div>
            </CardHeader>
            <CardFooter className="flex-col items-start gap-1 px-4 text-sm">
              <div className="line-clamp-1 flex gap-2 font-medium">
                {card.hint || card.meta?.text || '\u00a0'}
                <Activity className="size-4" aria-hidden="true" />
              </div>
              <div className="text-muted-foreground">{card.meta?.alert || '\u00a0'}</div>
            </CardFooter>
          </Card>
        )
      })}
    </div>
  )
}

function UserTrendCard({
  points,
  loading,
  labels,
}: {
  points: readonly UserUsageTrendPoint[]
  loading: boolean
  labels: DashboardPageLabels
}) {
  const totals = new Map<number, { name: string; requests: number; tokens: number }>()
  for (const point of points) {
    const name = point.username?.trim() || point.email?.trim() || labels.userPrefix(point.user_id)
    const current = totals.get(point.user_id) || { name, requests: 0, tokens: 0 }
    current.requests += point.requests
    current.tokens += point.tokens
    totals.set(point.user_id, current)
  }
  const rows = [...totals.entries()].sort(([, a], [, b]) => b.tokens - a.tokens).slice(0, 12)

  return (
    <Card className="@container/card min-w-0">
      <CardHeader>
        <CardTitle>{labels.userUsageTrend}</CardTitle>
        <CardDescription>{labels.recentUsage}</CardDescription>
      </CardHeader>
      <CardContent className="min-h-56">
        {loading ? (
          <div className="space-y-3" role="status" aria-label={labels.userUsageTrend}>
            {Array.from({ length: 5 }, (_, index) => <Skeleton key={index} className="h-8 w-full" />)}
          </div>
        ) : rows.length === 0 ? (
          <div className="flex h-48 items-center justify-center text-sm text-muted-foreground">{labels.noData}</div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{labels.userUsageTrend}</TableHead>
                <TableHead className="text-right">{labels.requests}</TableHead>
                <TableHead className="text-right">{labels.tokens}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map(([userId, row]) => (
                <TableRow key={userId}>
                  <TableCell className="max-w-[14rem] truncate font-medium">{row.name}</TableCell>
                  <TableCell className="text-right tabular-nums text-muted-foreground">{row.requests.toLocaleString()}</TableCell>
                  <TableCell className="text-right tabular-nums">{row.tokens.toLocaleString()}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}

export default function DashboardPage({
  loading,
  cards,
  modelDistribution,
  tokenUsageTrend,
  userTrend,
  userTrendLoading,
  labels,
}: DashboardPageProps) {
  return (
    <div className="@container/main flex flex-1 flex-col gap-2">
      <div className="flex flex-col gap-4 py-3 md:gap-4">
        <SectionCards cards={cards} loading={loading} />
        <div className="px-3">
          <TokenUsageTrend {...tokenUsageTrend} />
        </div>
        <div className="grid min-w-0 grid-cols-1 gap-4 px-3 lg:grid-cols-2">
          <ModelDistribution {...modelDistribution} />
          <UserTrendCard points={userTrend} loading={userTrendLoading} labels={labels} />
        </div>
      </div>
    </div>
  )
}
