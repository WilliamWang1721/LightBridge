import { useRef, useState, type ReactNode } from 'react'
import { getUserBreakdown } from '@/api/admin/dashboard'
import type { ModelStat, UserBreakdownItem, UserSpendingRankingItem } from '@/types'
import { createShadcnElement as h } from './ui/createElement'
import { Button } from './ui/button'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { AppIcon } from './ui/app-icon'

export interface ModelDistributionLabels {
  modelDistribution: string
  spendingRankingTitle: string
  viewModelDistribution: string
  viewSpendingRanking: string
  spendingRankingUser: string
  spendingRankingRequests: string
  spendingRankingTokens: string
  spendingRankingSpend: string
  spendingRankingOther: string
  model: string
  requests: string
  tokens: string
  actual: string
  accountCost: string
  standard: string
  noData: string
  failedToLoad: string
  userPrefix: (id: number) => string
}

export interface ModelDistributionPageProps {
  modelStats?: ModelStat[] | null
  rankingItems?: UserSpendingRankingItem[] | null
  rankingTotalActualCost?: number
  rankingTotalRequests?: number
  rankingTotalTokens?: number
  loading?: boolean
  rankingLoading?: boolean
  rankingError?: boolean
  startDate?: string
  endDate?: string
  labels: ModelDistributionLabels
  onRankingClick?: (item: UserSpendingRankingItem) => void
}

type RankingDisplayItem = UserSpendingRankingItem & { isOther?: boolean }

const chartColors = [
  'hsl(var(--foreground))',
  'hsl(var(--muted-foreground))',
  'hsl(var(--border))',
  'hsl(var(--muted-foreground) / 0.65)',
  'hsl(var(--foreground) / 0.75)',
  'hsl(var(--border) / 0.65)',
  'hsl(var(--muted-foreground) / 0.45)',
  'hsl(var(--foreground) / 0.55)',
]

const formatTokens = (value: number): string => {
  const normalized = Number.isFinite(value) ? value : 0
  if (normalized >= 1_000_000_000) return `${(normalized / 1_000_000_000).toFixed(2)}B`
  if (normalized >= 1_000_000) return `${(normalized / 1_000_000).toFixed(2)}M`
  if (normalized >= 1_000) return `${(normalized / 1_000).toFixed(2)}K`
  return normalized.toLocaleString()
}

const formatCost = (value: number | null | undefined): string => {
  const normalized = typeof value === 'number' && Number.isFinite(value) ? value : 0
  if (normalized >= 1000) return `${(normalized / 1000).toFixed(2)}K`
  if (normalized >= 1) return normalized.toFixed(2)
  if (normalized >= 0.01) return normalized.toFixed(3)
  return normalized.toFixed(4)
}

const formatNumber = (value: number): string => (
  (Number.isFinite(value) ? value : 0).toLocaleString()
)

function Spinner(): ReactNode {
  return h('div', { className: 'flex h-48 items-center justify-center', role: 'status', 'aria-label': 'Loading' },
    h('div', {
      className: 'h-5 w-5 animate-spin rounded-full border-2 border-[hsl(var(--muted-foreground))] border-t-transparent',
    }),
  )
}

function Chevron({ expanded }: { expanded: boolean }): ReactNode {
  return h(AppIcon, { name: expanded ? 'chevronDown' : 'chevronRight', className: 'h-3 w-3 shrink-0' })
}

interface DonutItem {
  label: string
  value: number
}

function Donut({ items, valueLabel }: { items: DonutItem[]; valueLabel: (value: number) => string }): ReactNode {
  const radius = 42
  const circumference = 2 * Math.PI * radius
  const total = items.reduce((sum, item) => sum + Math.max(item.value, 0), 0)
  let offset = 0

  return h('div', { className: 'h-48 w-48 shrink-0', role: 'img', 'aria-label': 'Distribution chart' }, h('svg', {
    className: 'h-full w-full',
    viewBox: '0 0 112 112',
  }, [
    h('circle', {
      key: 'track',
      cx: 56,
      cy: 56,
      r: radius,
      fill: 'none',
      stroke: 'hsl(var(--muted) / 0.6)',
      strokeWidth: 24,
    }),
    ...items.map((item, index) => {
      const value = Math.max(item.value, 0)
      const length = total > 0 ? (value / total) * circumference : 0
      const circle = h('circle', {
        key: `${item.label}-${index}`,
        cx: 56,
        cy: 56,
        r: radius,
        fill: 'none',
        stroke: chartColors[index % chartColors.length],
        strokeDasharray: `${length} ${circumference - length}`,
        strokeDashoffset: -offset,
        strokeWidth: 24,
        transform: 'rotate(-90 56 56)',
        strokeLinecap: 'butt',
      }, h('title', {}, `${item.label}: ${valueLabel(value)}`))
      offset += length
      return circle
    }),
  ]))
}

function UserBreakdown({ items, loading, labels }: {
  items: UserBreakdownItem[]
  loading: boolean
  labels: ModelDistributionLabels
}): ReactNode {
  if (loading) {
    return h('div', { className: 'flex items-center justify-center py-3' }, h(Spinner))
  }
  if (items.length === 0) {
    return h('div', { className: 'py-2 text-center text-xs text-[hsl(var(--muted-foreground))]' }, labels.noData)
  }

  return h('div', { className: 'bg-[hsl(var(--muted)/0.35)]' }, h('table', { className: 'w-full text-xs' }, h('tbody', {}, items.map((user) => h('tr', {
    key: user.user_id,
    className: 'border-t border-[hsl(var(--border)/0.5)]',
  }, [
    h('td', { key: 'email', className: 'max-w-[120px] truncate py-1 pl-6 text-[hsl(var(--muted-foreground))]', title: user.email }, user.email || labels.userPrefix(user.user_id)),
    h('td', { key: 'requests', className: 'py-1 text-right text-[hsl(var(--muted-foreground))]' }, formatNumber(user.requests)),
    h('td', { key: 'tokens', className: 'py-1 text-right text-[hsl(var(--muted-foreground))]' }, formatTokens(user.total_tokens)),
    h('td', { key: 'actual', className: 'py-1 text-right text-[hsl(var(--foreground))]' }, `$${formatCost(user.actual_cost)}`),
    h('td', { key: 'account', className: 'py-1 text-right text-[hsl(var(--muted-foreground))]' }, `$${formatCost(user.account_cost)}`),
    h('td', { key: 'standard', className: 'py-1 pr-1 text-right text-[hsl(var(--muted-foreground)/0.7)]' }, `$${formatCost(user.cost)}`),
  ])))))
}

export default function ModelDistribution({
  modelStats = [],
  rankingItems = [],
  rankingTotalActualCost = 0,
  rankingTotalRequests = 0,
  rankingTotalTokens = 0,
  loading = false,
  rankingLoading = false,
  rankingError = false,
  startDate,
  endDate,
  labels,
  onRankingClick,
}: ModelDistributionPageProps): ReactNode {
  const [activeView, setActiveView] = useState<'model_distribution' | 'spending_ranking'>('model_distribution')
  const [expandedModel, setExpandedModel] = useState<string | null>(null)
  const [breakdownItems, setBreakdownItems] = useState<UserBreakdownItem[]>([])
  const [breakdownLoading, setBreakdownLoading] = useState(false)
  const breakdownSequence = useRef(0)
  const displayModelStats = [...(modelStats || [])].sort((a, b) => b.total_tokens - a.total_tokens)
  const rankings = rankingItems || []
  const rankedActualCost = rankings.reduce((sum, item) => sum + item.actual_cost, 0)
  const rankedRequests = rankings.reduce((sum, item) => sum + item.requests, 0)
  const rankedTokens = rankings.reduce((sum, item) => sum + item.tokens, 0)
  const otherRankingItem: RankingDisplayItem | null = rankings.length > 0
    ? {
        user_id: 0,
        email: '',
        actual_cost: Math.max(rankingTotalActualCost - rankedActualCost, 0),
        requests: Math.max(rankingTotalRequests - rankedRequests, 0),
        tokens: Math.max(rankingTotalTokens - rankedTokens, 0),
        isOther: true,
      }
    : null
  const hasOther = otherRankingItem !== null && (
    otherRankingItem.actual_cost > 0.000001 || otherRankingItem.requests > 0 || otherRankingItem.tokens > 0
  )
  const rankingDisplayItems: RankingDisplayItem[] = hasOther
    ? [...rankings, otherRankingItem!]
    : [...rankings]

  const toggleBreakdown = (model: string) => {
    if (expandedModel === model) {
      setExpandedModel(null)
      return
    }
    const sequence = ++breakdownSequence.current
    setExpandedModel(model)
    setBreakdownItems([])
    setBreakdownLoading(true)
    void getUserBreakdown({ start_date: startDate, end_date: endDate, model, model_source: 'requested' })
      .then((response) => {
        if (sequence === breakdownSequence.current) setBreakdownItems(response.users || [])
      })
      .catch(() => {
        if (sequence === breakdownSequence.current) setBreakdownItems([])
      })
      .finally(() => {
        if (sequence === breakdownSequence.current) setBreakdownLoading(false)
      })
  }

  const rankingLabel = (item: RankingDisplayItem) => (
    item.isOther ? labels.spendingRankingOther : item.email || labels.userPrefix(item.user_id)
  )
  const title = activeView === 'model_distribution' ? labels.modelDistribution : labels.spendingRankingTitle
  const modelContent = loading
    ? h(Spinner)
    : displayModelStats.length === 0
      ? h('div', { className: 'flex h-48 items-center justify-center text-sm text-[hsl(var(--muted-foreground))]' }, labels.noData)
      : h('div', { className: 'flex min-w-0 items-center gap-6' }, [
        h(Donut, {
          key: 'chart',
          items: displayModelStats.map((model) => ({ label: model.model, value: model.total_tokens })),
          valueLabel: formatTokens,
        }),
        h('div', { key: 'table', className: 'max-h-48 min-w-0 flex-1 overflow-y-auto' }, h('table', { className: 'w-full text-xs' }, [
          h('thead', { key: 'head' }, h('tr', { className: 'text-[hsl(var(--muted-foreground))]' }, [
            h('th', { key: 'model', className: 'pb-2 text-left' }, labels.model),
            h('th', { key: 'requests', className: 'pb-2 text-right' }, labels.requests),
            h('th', { key: 'tokens', className: 'pb-2 text-right' }, labels.tokens),
            h('th', { key: 'actual', className: 'pb-2 text-right' }, labels.actual),
            h('th', { key: 'account', className: 'pb-2 text-right' }, labels.accountCost),
            h('th', { key: 'standard', className: 'pb-2 text-right' }, labels.standard),
          ])),
          h('tbody', { key: 'body' }, displayModelStats.flatMap((model) => {
            const expanded = expandedModel === model.model
            return [
              h('tr', { key: model.model, className: 'border-t border-[hsl(var(--border)/0.45)]' }, [
                h('td', { key: 'model', className: 'max-w-[140px] truncate py-1.5 font-medium text-[hsl(var(--foreground))]' }, h('button', {
                  type: 'button',
                  className: 'inline-flex max-w-full items-center gap-1 truncate text-left hover:text-[hsl(var(--muted-foreground))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))]',
                  title: model.model,
                  'aria-expanded': expanded,
                  onClick: () => toggleBreakdown(model.model),
                }, [h(Chevron, { key: 'chevron', expanded }), h('span', { key: 'name', className: 'truncate' }, model.model)])),
                h('td', { key: 'requests', className: 'py-1.5 text-right text-[hsl(var(--muted-foreground))]' }, formatNumber(model.requests)),
                h('td', { key: 'tokens', className: 'py-1.5 text-right text-[hsl(var(--muted-foreground))]' }, formatTokens(model.total_tokens)),
                h('td', { key: 'actual', className: 'py-1.5 text-right text-[hsl(var(--foreground))]' }, `$${formatCost(model.actual_cost)}`),
                h('td', { key: 'account', className: 'py-1.5 text-right text-[hsl(var(--muted-foreground))]' }, `$${formatCost(model.account_cost)}`),
                h('td', { key: 'standard', className: 'py-1.5 text-right text-[hsl(var(--muted-foreground)/0.7)]' }, `$${formatCost(model.cost)}`),
              ]),
              expanded ? h('tr', { key: `${model.model}-breakdown` }, h('td', { colSpan: 6, className: 'p-0' }, h(UserBreakdown, { items: breakdownItems, loading: breakdownLoading, labels }))) : null,
            ]
          })),
        ])),
      ])

  let rankingContent: ReactNode
  if (rankingLoading) {
    rankingContent = h(Spinner)
  } else if (rankingError) {
    rankingContent = h('div', { className: 'flex h-48 items-center justify-center text-sm text-[hsl(var(--muted-foreground))]' }, labels.failedToLoad)
  } else if (rankingDisplayItems.length === 0) {
    rankingContent = h('div', { className: 'flex h-48 items-center justify-center text-sm text-[hsl(var(--muted-foreground))]' }, labels.noData)
  } else {
    rankingContent = h('div', { className: 'flex min-w-0 items-center gap-6' }, [
      h(Donut, {
        key: 'chart',
        items: rankingDisplayItems.map((item) => ({ label: rankingLabel(item), value: item.actual_cost })),
        valueLabel: (value) => `$${formatCost(value)}`,
      }),
      h('div', { key: 'table', className: 'max-h-48 min-w-0 flex-1 overflow-y-auto' }, h('table', { className: 'w-full text-xs' }, [
        h('thead', { key: 'head' }, h('tr', { className: 'text-[hsl(var(--muted-foreground))]' }, [
          h('th', { key: 'user', className: 'pb-2 text-left' }, labels.spendingRankingUser),
          h('th', { key: 'requests', className: 'pb-2 text-right' }, labels.spendingRankingRequests),
          h('th', { key: 'tokens', className: 'pb-2 text-right' }, labels.spendingRankingTokens),
          h('th', { key: 'spend', className: 'pb-2 text-right' }, labels.spendingRankingSpend),
        ])),
        h('tbody', { key: 'body' }, rankingDisplayItems.map((item, index) => {
          const clickable = !item.isOther && Boolean(onRankingClick)
          const handleClick = () => {
            if (clickable) onRankingClick?.(item)
          }
          return h('tr', {
            key: item.isOther ? 'others' : `${item.user_id}-${index}`,
            className: item.isOther
              ? 'border-t border-[hsl(var(--border)/0.45)] bg-[hsl(var(--muted)/0.35)]'
              : 'cursor-pointer border-t border-[hsl(var(--border)/0.45)] transition-colors hover:bg-[hsl(var(--muted)/0.45)]',
            role: clickable ? 'button' : undefined,
            tabIndex: clickable ? 0 : undefined,
            'aria-label': clickable ? rankingLabel(item) : undefined,
            onClick: handleClick,
            onKeyDown: clickable ? (event: KeyboardEvent) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                handleClick()
              }
            } : undefined,
          }, [
            h('td', { key: 'user', className: 'py-1.5' }, h('div', { className: 'flex min-w-0 items-center gap-2' }, [
              h('span', { key: 'rank', className: 'shrink-0 text-[11px] font-semibold text-[hsl(var(--muted-foreground))]' }, item.isOther ? 'Σ' : `#${index + 1}`),
              h('span', { key: 'label', className: 'block max-w-[140px] truncate font-medium text-[hsl(var(--foreground))]', title: rankingLabel(item) }, rankingLabel(item)),
            ])),
            h('td', { key: 'requests', className: 'py-1.5 text-right text-[hsl(var(--muted-foreground))]' }, formatNumber(item.requests)),
            h('td', { key: 'tokens', className: 'py-1.5 text-right text-[hsl(var(--muted-foreground))]' }, formatTokens(item.tokens)),
            h('td', { key: 'actual', className: 'py-1.5 text-right text-[hsl(var(--foreground))]' }, `$${formatCost(item.actual_cost)}`),
          ])
        })),
      ])),
    ])
  }

  return h(Card, { className: '@container/card min-w-0 overflow-hidden' }, [
    h(CardHeader, { key: 'header', className: 'flex-row items-center justify-between gap-3 space-y-0 pb-4' }, [
      h(CardTitle, { key: 'title', className: 'text-base' }, title),
      h('div', { key: 'switches', className: 'inline-flex rounded-lg bg-[hsl(var(--muted))] p-1' }, [
        h(Button, {
          key: 'model',
          variant: activeView === 'model_distribution' ? 'default' : 'ghost',
          size: 'sm',
          className: 'h-7 px-2.5 text-xs',
          'aria-pressed': activeView === 'model_distribution',
          disabled: loading || rankingLoading,
          onClick: () => setActiveView('model_distribution'),
        }, labels.viewModelDistribution),
        h(Button, {
          key: 'ranking',
          variant: activeView === 'spending_ranking' ? 'default' : 'ghost',
          size: 'sm',
          className: 'h-7 px-2.5 text-xs',
          'aria-pressed': activeView === 'spending_ranking',
          disabled: loading || rankingLoading,
          onClick: () => setActiveView('spending_ranking'),
        }, labels.viewSpendingRanking),
      ]),
    ]),
    h(CardContent, { key: 'content', className: 'pt-0' }, activeView === 'model_distribution' ? modelContent : rankingContent),
  ])
}
