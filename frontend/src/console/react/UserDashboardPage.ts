import { type ReactNode } from 'react'
import type { UserDashboardStats } from '@/api/usage'
import type { PlatformQuotaItem, ModelStat, TrendDataPoint, UsageLog } from '@/types'
import TokenUsageTrend from './TokenUsageTrend'
import { readUsageNumber } from '@/utils/usageDisplay'
import { createShadcnElement as h } from './ui/createElement'
import { AppIcon, type AppIconName } from './ui/app-icon'

export interface UserDashboardCopy {
  refresh: string
  balance: string
  available: string
  apiKeys: string
  active: string
  todayRequests: string
  total: string
  todayCost: string
  totalTokens: string
  todayTokens: string
  input: string
  output: string
  performance: string
  avgResponse: string
  modelDistribution: string
  tokenUsageTrend: string
  model: string
  requests: string
  tokens: string
  actual: string
  standard: string
  noData: string
  timeRange: string
  granularity: string
  day: string
  hour: string
  recentUsage: string
  last7Days: string
  noUsageRecords: string
  startUsingApi: string
  viewAllUsage: string
  quickActions: string
  createApiKey: string
  generateNewKey: string
  viewUsage: string
  checkDetailedLogs: string
  redeemCode: string
  addBalanceWithCode: string
  platformBreakdown: string
  platformBreakdownEmpty: string
  platformCount: (count: number) => string
  platformOther: string
  platformQuota: { title: string; daily: string; weekly: string; monthly: string; resetsAt: (time: string) => string; disabled: string }
}

export interface UserDashboardPageProps {
  stats: UserDashboardStats | null
  balance: number
  isSimple: boolean
  loading: boolean
  loadingUsage: boolean
  loadingCharts: boolean
  trend: TrendDataPoint[]
  models: ModelStat[]
  recentUsage: UsageLog[]
  platformQuotas: PlatformQuotaItem[] | null
  startDate: string
  endDate: string
  granularity: string
  copy: UserDashboardCopy
  onDateChange: (key: 'startDate' | 'endDate', value: string) => void
  onGranularityChange: (value: string) => void
  onRefresh: () => void
  onNavigate: (path: string) => void
}

function Icon({ name }: { name: AppIconName }): ReactNode {
  return h(AppIcon, { name, className: 'h-5 w-5' })
}

const money = (value: number) => Number.isFinite(value) ? value.toFixed(4) : '0.0000'
const tokens = (value: number) => value >= 1_000_000 ? `${(value / 1_000_000).toFixed(1)}M` : value >= 1000 ? `${(value / 1000).toFixed(1)}K` : value.toLocaleString()
const duration = (value: number) => value >= 1000 ? `${(value / 1000).toFixed(2)}s` : `${value.toFixed(0)}ms`

function StatCard({ icon, label, value, hint }: { icon: Parameters<typeof Icon>[0]['name']; label: string; value: string; hint: string }): ReactNode {
  return h('section', { className: 'card min-w-0 p-4' }, h('div', { className: 'flex items-center gap-3' }, [
    h('div', { key: 'icon', className: 'flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]' }, h(Icon, { name: icon })),
    h('div', { key: 'text', className: 'min-w-0' }, [
      h('p', { key: 'label', className: 'text-xs font-medium text-[hsl(var(--muted-foreground))]' }, label),
      h('p', { key: 'value', className: 'text-xl font-bold text-[hsl(var(--foreground))]' }, value),
      h('p', { key: 'hint', className: 'truncate text-xs text-[hsl(var(--muted-foreground))]' }, hint),
    ]),
  ]))
}

function PlatformBreakdown({ stats, quotas, copy }: { stats: UserDashboardStats; quotas: PlatformQuotaItem[] | null; copy: UserDashboardCopy }): ReactNode {
  const labels: Record<string, string> = { anthropic: 'Claude', openai: 'OpenAI', gemini: 'Gemini', grok: 'Grok', antigravity: 'Gemini', custom: 'Custom' }
  const order: Record<string, number> = { anthropic: 0, openai: 1, gemini: 2, grok: 3, antigravity: 4, custom: 5 }
  const byStats = new Map((stats.by_platform || []).map((item) => [item.platform, item]))
  const byQuota = new Map((quotas || []).map((item) => [item.platform, item]))
  const cards = Array.from(new Set([...byStats.keys(), ...byQuota.keys()])).sort((a, b) => (order[a] ?? 99) - (order[b] ?? 99))
  const total = stats.total_actual_cost
  const sum = cards.reduce((value, platform) => value + (byStats.get(platform)?.total_actual_cost || 0), 0)
  if (total - sum > 0.0001) cards.push('__other__')
  if (!cards.length) return h('section', { className: 'card p-4' }, [h('h2', { key: 'title', className: 'mb-4 text-sm font-semibold text-[hsl(var(--foreground))]' }, copy.platformBreakdown), h('p', { key: 'empty', className: 'py-8 text-center text-sm text-[hsl(var(--muted-foreground))]' }, copy.platformBreakdownEmpty)])
  const platformCards = cards.map((platform) => {
    const stat = byStats.get(platform)
    const quota = byQuota.get(platform)
    const title = platform === '__other__' ? copy.platformOther : labels[platform] || platform
    const cost = stat?.total_actual_cost ?? (platform === '__other__' ? Math.max(0, total - sum) : 0)
    const windows: Array<['daily' | 'weekly' | 'monthly', number | null, number]> = [['daily', quota?.daily_limit_usd ?? null, quota?.daily_usage_usd ?? 0], ['weekly', quota?.weekly_limit_usd ?? null, quota?.weekly_usage_usd ?? 0], ['monthly', quota?.monthly_limit_usd ?? null, quota?.monthly_usage_usd ?? 0]]
    const quotaContent = quota && windows.some(([, limit]) => limit != null) ? h('div', { key: 'quota', className: 'mt-3 space-y-2 border-t border-[hsl(var(--border))] pt-2' }, [
      h('p', { key: 'label', className: 'text-[10px] uppercase tracking-wide text-[hsl(var(--muted-foreground))]' }, copy.platformQuota.title),
      ...windows.flatMap(([window, limit, usage]) => limit == null ? [] : [h('div', { key: window, className: 'space-y-1' }, [
        h('div', { key: 'line', className: 'flex justify-between gap-2 text-xs text-[hsl(var(--muted-foreground))]' }, [copy.platformQuota[window], h('span', { className: 'font-mono' }, limit === 0 ? copy.platformQuota.disabled : `$${usage.toFixed(2)} / $${limit.toFixed(2)}`)]),
        h('div', { key: 'bar', className: 'h-1.5 overflow-hidden rounded-full bg-[hsl(var(--muted))]' }, h('div', { className: `h-full rounded-full ${limit === 0 ? 'bg-[hsl(var(--destructive))]' : 'bg-[hsl(var(--foreground))]'}`, style: { width: `${limit === 0 ? 100 : Math.min(100, Math.round((usage / limit) * 100))}%` } })),
      ])]),
    ]) : null
    return h('article', { key: platform, className: `rounded-xl border p-3 ${platform === '__other__' ? 'border-dashed border-[hsl(var(--border))] bg-[hsl(var(--muted)/.4)]' : 'border-[hsl(var(--border))]'}` }, [
      h('div', { key: 'top', className: 'flex items-center justify-between gap-2' }, [h('span', { key: 'name', className: 'text-sm font-semibold text-[hsl(var(--foreground))]' }, title), h('span', { key: 'cost', className: 'font-mono text-sm text-[hsl(var(--foreground))]' }, `$${money(cost)}`)]),
      h('div', { key: 'metrics', className: 'mt-2 space-y-1 text-xs text-[hsl(var(--muted-foreground))]' }, [
        h('div', { key: 'today', className: 'flex justify-between gap-2' }, [copy.todayCost, h('span', { className: 'font-mono text-[hsl(var(--foreground))]' }, `$${money(stat?.today_actual_cost || 0)}`)]),
        h('div', { key: 'requests', className: 'flex justify-between gap-2' }, [copy.requests, h('span', { className: 'font-mono' }, stat?.total_requests ? stat.total_requests.toLocaleString() : '-')]),
        h('div', { key: 'tokens', className: 'flex justify-between gap-2' }, [copy.tokens, h('span', { className: 'font-mono' }, stat?.total_tokens ? tokens(stat.total_tokens) : '-')]),
      ]),
      quotaContent,
    ])
  })
  return h('section', { className: 'card p-4' }, [
    h('div', { key: 'heading', className: 'mb-4 flex items-center justify-between gap-3' }, [h('h2', { key: 'title', className: 'text-sm font-semibold text-[hsl(var(--foreground))]' }, copy.platformBreakdown), h('span', { key: 'count', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, copy.platformCount(cards.length))]),
    h('div', { key: 'grid', className: 'grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4' }, platformCards),
  ])
}

function ModelDistribution({ models, copy }: { models: ModelStat[]; copy: UserDashboardCopy }): ReactNode {
  const max = Math.max(1, ...models.map((model) => model.total_tokens))
  return h('section', { className: 'card min-w-0 overflow-hidden p-4' }, [
    h('h2', { key: 'title', className: 'mb-4 text-sm font-semibold text-[hsl(var(--foreground))]' }, copy.modelDistribution),
    models.length ? h('div', { key: 'rows', className: 'space-y-3' }, models.slice(0, 8).map((model) => h('div', { key: model.model, className: 'space-y-1.5' }, [
      h('div', { key: 'heading', className: 'flex items-center justify-between gap-3 text-xs' }, [
        h('span', { key: 'model', className: 'min-w-0 truncate font-medium text-[hsl(var(--foreground))]' }, model.model),
        h('span', { key: 'tokens', className: 'shrink-0 text-[hsl(var(--muted-foreground))]' }, `${tokens(model.total_tokens)} ${copy.tokens}`),
      ]),
      h('div', { key: 'bar', className: 'h-2 overflow-hidden rounded-full bg-[hsl(var(--muted))]' }, h('div', { className: 'h-full rounded-full bg-[hsl(var(--foreground))]', style: { width: `${Math.max(2, model.total_tokens / max * 100)}%` } })),
      h('div', { key: 'meta', className: 'flex justify-between text-[10px] text-[hsl(var(--muted-foreground))]' }, [`${model.requests.toLocaleString()} ${copy.requests}`, `$${money(model.actual_cost)} ${copy.actual}`]),
    ]))) : h('p', { key: 'empty', className: 'flex h-48 items-center justify-center text-sm text-[hsl(var(--muted-foreground))]' }, copy.noData),
  ])
}

function RecentUsage({ data, loading, copy, onNavigate }: { data: UsageLog[]; loading: boolean; copy: UserDashboardCopy; onNavigate: (path: string) => void }): ReactNode {
  if (loading) return h('section', { className: 'card p-6' }, h('div', { className: 'flex h-48 items-center justify-center' }, h('span', { className: 'h-6 w-6 animate-spin rounded-full border-2 border-[hsl(var(--border))] border-t-[hsl(var(--foreground))]' })))
  const rows = data.slice(0, 5).map((log) => ({ model: String(log.model || '-'), created: new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(log.created_at)), tokens: readUsageNumber(log, ['total_tokens', 'tokens'], readUsageNumber(log, ['input_tokens']) + readUsageNumber(log, ['output_tokens'])), actual: readUsageNumber(log, ['actual_cost', 'user_cost']), cost: readUsageNumber(log, ['total_cost', 'cost']) }))
  const rowCards = rows.map((row, index) => h('div', { key: `${row.model}-${index}`, className: 'flex min-w-0 flex-col gap-3 rounded-xl bg-[hsl(var(--muted)/.55)] p-4 sm:flex-row sm:items-center sm:justify-between' }, [
    h('div', { key: 'name', className: 'min-w-0' }, [
      h('p', { key: 'model', className: 'truncate text-sm font-medium text-[hsl(var(--foreground))]' }, row.model),
      h('p', { key: 'date', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, row.created),
    ]),
    h('div', { key: 'numbers', className: 'text-left sm:text-right' }, [
      h('p', { key: 'cost', className: 'text-sm font-semibold text-[hsl(var(--foreground))]' }, `$${money(row.actual)} / $${money(row.cost)}`),
      h('p', { key: 'tokens', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, `${tokens(row.tokens)} ${copy.tokens}`),
    ]),
  ]))
  return h('section', { className: 'card overflow-hidden' }, [
    h('header', { key: 'header', className: 'flex items-center justify-between border-b border-[hsl(var(--border))] px-6 py-4' }, [h('h2', { key: 'title', className: 'text-sm font-semibold text-[hsl(var(--foreground))]' }, copy.recentUsage), h('span', { key: 'badge', className: 'rounded-full bg-[hsl(var(--muted))] px-2.5 py-1 text-xs text-[hsl(var(--muted-foreground))]' }, copy.last7Days)]),
    rows.length ? h('div', { key: 'body', className: 'space-y-3 p-6' }, [...rowCards, h('button', { key: 'all', type: 'button', className: 'flex w-full items-center justify-center gap-2 py-3 text-sm font-medium text-[hsl(var(--foreground))] transition-colors hover:text-[hsl(var(--muted-foreground))]', onClick: () => onNavigate('/usage') }, [copy.viewAllUsage, h(Icon, { key: 'icon', name: 'arrow' })])]) : h('div', { key: 'empty', className: 'p-10 text-center' }, [h('p', { key: 'title', className: 'text-sm font-medium text-[hsl(var(--foreground))]' }, copy.noUsageRecords), h('p', { key: 'hint', className: 'mt-1 text-sm text-[hsl(var(--muted-foreground))]' }, copy.startUsingApi)]),
  ])
}

function QuickActions({ copy, onNavigate }: { copy: UserDashboardCopy; onNavigate: (path: string) => void }): ReactNode {
  const actions = [{ path: '/keys', icon: 'key' as const, title: copy.createApiKey, description: copy.generateNewKey }, { path: '/usage', icon: 'chart' as const, title: copy.viewUsage, description: copy.checkDetailedLogs }, { path: '/redeem', icon: 'gift' as const, title: copy.redeemCode, description: copy.addBalanceWithCode }]
  return h('section', { className: 'card overflow-hidden' }, [h('header', { key: 'header', className: 'border-b border-[hsl(var(--border))] px-6 py-4' }, h('h2', { className: 'text-sm font-semibold text-[hsl(var(--foreground))]' }, copy.quickActions)), h('div', { key: 'actions', className: 'space-y-3 p-4' }, actions.map((action) => h('button', { key: action.path, type: 'button', className: 'group flex w-full items-center gap-4 rounded-xl bg-[hsl(var(--muted)/.55)] p-4 text-left transition-colors hover:bg-[hsl(var(--muted))]', onClick: () => onNavigate(action.path) }, [h('span', { key: 'icon', className: 'flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-[hsl(var(--background))] text-[hsl(var(--foreground))]' }, h(Icon, { name: action.icon })), h('span', { key: 'text', className: 'min-w-0 flex-1' }, [h('span', { key: 'title', className: 'block text-sm font-medium text-[hsl(var(--foreground))]' }, action.title), h('span', { key: 'description', className: 'block text-xs text-[hsl(var(--muted-foreground))]' }, action.description)]), h(Icon, { key: 'arrow', name: 'arrow' })])))])
}

export default function UserDashboardPage({ stats, balance, isSimple, loading, loadingUsage, loadingCharts, trend, models, recentUsage, platformQuotas, startDate, endDate, granularity, copy, onDateChange, onGranularityChange, onRefresh, onNavigate }: UserDashboardPageProps) {
  if (loading || !stats) return h('div', { className: 'flex items-center justify-center py-12', role: 'status' }, h('span', { className: 'h-8 w-8 animate-spin rounded-full border-2 border-[hsl(var(--border))] border-t-[hsl(var(--foreground))]' }))
  const costHint = `$${money(stats.total_actual_cost)} / $${money(stats.total_cost)}`
  return h('div', { className: 'min-w-0 space-y-5 pb-4 md:space-y-6' }, [
    h('div', { key: 'stats', className: 'space-y-4' }, [h('div', { key: 'row1', className: 'grid min-w-0 grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4' }, [!isSimple ? h(StatCard, { key: 'balance', icon: 'wallet', label: copy.balance, value: `$${balance.toFixed(2)}`, hint: copy.available }) : null, h(StatCard, { key: 'keys', icon: 'key', label: copy.apiKeys, value: String(stats.total_api_keys || 0), hint: `${stats.active_api_keys || 0} ${copy.active}` }), h(StatCard, { key: 'requests', icon: 'chart', label: copy.todayRequests, value: String(stats.today_requests || 0), hint: `${copy.total}: ${stats.total_requests.toLocaleString()}` }), h(StatCard, { key: 'cost', icon: 'dollar', label: copy.todayCost, value: `$${money(stats.today_actual_cost || 0)}`, hint: `${costHint} ${copy.total}` })]), h('div', { key: 'row2', className: 'grid min-w-0 grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4' }, [h(StatCard, { key: 'todayTokens', icon: 'cube', label: copy.todayTokens, value: tokens(stats.today_tokens || 0), hint: `${copy.input}: ${tokens(stats.today_input_tokens || 0)} / ${copy.output}: ${tokens(stats.today_output_tokens || 0)}` }), h(StatCard, { key: 'totalTokens', icon: 'database', label: copy.totalTokens, value: tokens(stats.total_tokens || 0), hint: `${copy.input}: ${tokens(stats.total_input_tokens || 0)} / ${copy.output}: ${tokens(stats.total_output_tokens || 0)}` }), h(StatCard, { key: 'performance', icon: 'bolt', label: copy.performance, value: `${tokens(stats.rpm || 0)} RPM`, hint: `${tokens(stats.tpm || 0)} TPM` }), h(StatCard, { key: 'duration', icon: 'clock', label: copy.avgResponse, value: duration(stats.average_duration_ms || 0), hint: copy.available })])]),
    h('div', { key: 'filters', className: 'card flex flex-wrap items-end gap-3 p-4' }, [h('label', { key: 'start', className: 'block' }, [h('span', { className: 'mb-1 block text-xs font-medium text-[hsl(var(--muted-foreground))]' }, `${copy.timeRange} · ${copy.day}`), h('input', { type: 'date', value: startDate, className: 'input h-10', onChange: (event: { target: { value: string } }) => onDateChange('startDate', event.target.value) })]), h('label', { key: 'end', className: 'block' }, [h('span', { className: 'mb-1 block text-xs font-medium text-[hsl(var(--muted-foreground))]' }, copy.timeRange), h('input', { type: 'date', value: endDate, className: 'input h-10', onChange: (event: { target: { value: string } }) => onDateChange('endDate', event.target.value) })]), h('button', { key: 'refresh', type: 'button', className: 'btn btn-secondary h-10', disabled: loadingCharts, onClick: onRefresh }, copy.refresh), h('label', { key: 'granularity', className: 'ml-auto block' }, [h('span', { className: 'mb-1 block text-xs font-medium text-[hsl(var(--muted-foreground))]' }, copy.granularity), h('select', { value: granularity, className: 'input h-10 min-w-28', onChange: (event: { target: { value: string } }) => onGranularityChange(event.target.value) }, [h('option', { key: 'day', value: 'day' }, copy.day), h('option', { key: 'hour', value: 'hour' }, copy.hour)])])]),
    h('div', { key: 'charts', className: 'grid min-w-0 grid-cols-1 gap-5 xl:grid-cols-2' }, [h(ModelDistribution, { key: 'models', models, copy }), h(TokenUsageTrend, { key: 'trend', trendData: trend, loading: loadingCharts, title: copy.tokenUsageTrend, noData: copy.noData })]),
    h(PlatformBreakdown, { key: 'platforms', stats, quotas: platformQuotas, copy }),
    h('div', { key: 'lower', className: 'grid min-w-0 grid-cols-1 gap-5 xl:grid-cols-3' }, [h('div', { key: 'recent', className: 'min-w-0 xl:col-span-2' }, h(RecentUsage, { data: recentUsage, loading: loadingUsage, copy, onNavigate })), h('div', { key: 'actions', className: 'min-w-0' }, h(QuickActions, { copy, onNavigate }))]),
  ])
}
