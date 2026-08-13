import { type ReactNode } from 'react'
import type { DashboardStats } from '@/types/payment'
import { createShadcnElement as h } from './ui/createElement'
import { AppIcon } from './ui/app-icon'

const DAYS = [7, 30, 90] as const

export interface PaymentDashboardPageCopy {
  daySuffix: string
  refresh: string
  todayRevenue: string
  totalRevenue: string
  todayOrders: string
  avgAmount: string
  orders: string
  dailyRevenue: string
  revenue: string
  orderCount: string
  paymentDistribution: string
  topUsers: string
  noData: string
  paymentMethod: (type: string) => string
}

export interface PaymentDashboardPageProps {
  days: number
  loading: boolean
  stats: DashboardStats | null
  copy: PaymentDashboardPageCopy
  onDaysChange: (days: number) => void
  onRefresh: () => void
}

function Icon({ name, className = 'h-5 w-5' }: { name: 'money' | 'card' | 'chart' | 'refresh'; className?: string }): ReactNode {
  return h(AppIcon, { name, className })
}

function formatMoney(value: number): string {
  return Number.isFinite(value) ? value.toFixed(2) : '0.00'
}

function StatCard({
  icon,
  label,
  value,
  hint,
}: {
  icon: 'money' | 'card' | 'chart'
  label: string
  value: string
  hint?: string
}): ReactNode {
  return h('section', { className: 'card min-w-0 p-4' }, h('div', { className: 'flex items-center gap-3' }, [
    h('div', { key: 'icon', className: 'flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]' }, h(Icon, { name: icon })),
    h('div', { key: 'content', className: 'min-w-0' }, [
      h('p', { key: 'label', className: 'text-xs font-medium text-[hsl(var(--muted-foreground))]' }, label),
      h('p', { key: 'value', className: 'text-xl font-bold text-[hsl(var(--foreground))]' }, value),
      hint ? h('p', { key: 'hint', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, hint) : null,
    ]),
  ]))
}

function LineChart({
  data,
  copy,
}: {
  data: DashboardStats['daily_series']
  copy: Pick<PaymentDashboardPageCopy, 'dailyRevenue' | 'revenue' | 'orderCount' | 'noData'>
}): ReactNode {
  if (!data.length) return h('div', { className: 'flex h-64 items-center justify-center text-sm text-[hsl(var(--muted-foreground))]' }, copy.noData)

  const width = 800
  const height = 240
  const padding = { top: 18, right: 18, bottom: 34, left: 44 }
  const innerWidth = width - padding.left - padding.right
  const innerHeight = height - padding.top - padding.bottom
  const amountMax = Math.max(...data.map((item) => item.amount), 1)
  const countMax = Math.max(...data.map((item) => item.count), 1)
  const x = (index: number) => padding.left + (data.length === 1 ? innerWidth / 2 : index * innerWidth / (data.length - 1))
  const points = (read: (item: DashboardStats['daily_series'][number]) => number, max: number) => data.map((item, index) => `${x(index)},${padding.top + innerHeight - read(item) / max * innerHeight}`).join(' ')
  const labels = data.filter((_, index) => data.length <= 6 || index === 0 || index === data.length - 1 || index % Math.ceil(data.length / 5) === 0)

  return h('div', { className: 'space-y-3' }, [
    h('div', { key: 'legend', className: 'flex flex-wrap items-center gap-4 text-xs text-[hsl(var(--muted-foreground))]' }, [
      h('span', { key: 'revenue', className: 'inline-flex items-center gap-1.5' }, [h('i', { key: 'dot', className: 'h-2 w-2 rounded-full bg-[hsl(var(--foreground))]' }), copy.revenue]),
      h('span', { key: 'orders', className: 'inline-flex items-center gap-1.5' }, [h('i', { key: 'dot', className: 'h-2 w-2 rounded-full border border-[hsl(var(--muted-foreground))]' }), copy.orderCount]),
    ]),
    h('svg', { key: 'chart', className: 'h-64 w-full overflow-visible', viewBox: `0 0 ${width} ${height}`, role: 'img', 'aria-label': copy.dailyRevenue }, [
      ...[0, 0.5, 1].map((ratio) => h('line', { key: `grid-${ratio}`, x1: padding.left, x2: width - padding.right, y1: padding.top + ratio * innerHeight, y2: padding.top + ratio * innerHeight, stroke: 'hsl(var(--border))', strokeWidth: 1, opacity: 0.7 })),
      h('polyline', { key: 'amount', points: points((item) => item.amount, amountMax), fill: 'none', stroke: 'hsl(var(--foreground))', strokeWidth: 2.5, strokeLinecap: 'round', strokeLinejoin: 'round' }),
      h('polyline', { key: 'count', points: points((item) => item.count, countMax), fill: 'none', stroke: 'hsl(var(--muted-foreground))', strokeWidth: 2, strokeDasharray: '5 4', strokeLinecap: 'round', strokeLinejoin: 'round' }),
      ...data.map((item, index) => h('circle', { key: `amount-${item.date}-${index}`, cx: x(index), cy: padding.top + innerHeight - item.amount / amountMax * innerHeight, r: 2.5, fill: 'hsl(var(--foreground))' })),
      ...labels.map((item) => {
        const index = data.indexOf(item)
        return h('text', { key: `label-${item.date}-${index}`, x: x(index), y: height - 8, textAnchor: 'middle', className: 'fill-[hsl(var(--muted-foreground))] text-[11px]' }, item.date)
      }),
    ]),
  ])
}

function PaymentDistribution({
  methods,
  copy,
}: {
  methods: DashboardStats['payment_methods']
  copy: Pick<PaymentDashboardPageCopy, 'paymentDistribution' | 'paymentMethod' | 'noData'>
}): ReactNode {
  return h('section', { className: 'card p-4' }, [
    h('h2', { key: 'title', className: 'mb-4 text-sm font-semibold text-[hsl(var(--foreground))]' }, copy.paymentDistribution),
    methods.length
      ? h('div', { key: 'rows', className: 'space-y-3' }, methods.map((method) => h('div', { key: method.type, className: 'flex items-center justify-between' }, [
        h('div', { key: 'label', className: 'flex items-center gap-2' }, [
          h('span', { key: 'dot', className: 'inline-block h-3 w-3 rounded-full bg-[hsl(var(--foreground))]' }),
          h('span', { key: 'name', className: 'text-sm text-[hsl(var(--muted-foreground))]' }, copy.paymentMethod(method.type)),
        ]),
        h('div', { key: 'value', className: 'text-right' }, [
          h('span', { key: 'amount', className: 'text-sm font-medium text-[hsl(var(--foreground))]' }, `¥${formatMoney(method.amount)}`),
          h('span', { key: 'count', className: 'ml-2 text-xs text-[hsl(var(--muted-foreground))]' }, `(${method.count})`),
        ]),
      ])))
      : h('div', { key: 'empty', className: 'flex h-32 items-center justify-center text-sm text-[hsl(var(--muted-foreground))]' }, copy.noData),
  ])
}

function TopUsers({
  users,
  copy,
}: {
  users: DashboardStats['top_users']
  copy: Pick<PaymentDashboardPageCopy, 'topUsers' | 'noData'>
}): ReactNode {
  return h('section', { className: 'card p-4' }, [
    h('h2', { key: 'title', className: 'mb-4 text-sm font-semibold text-[hsl(var(--foreground))]' }, copy.topUsers),
    users.length
      ? h('div', { key: 'rows', className: 'space-y-2' }, users.map((user, index) => h('div', { key: user.user_id, className: 'flex items-center justify-between rounded-lg px-3 py-2 hover:bg-[hsl(var(--muted))]' }, [
        h('div', { key: 'user', className: 'flex min-w-0 items-center gap-3' }, [
          h('span', { key: 'rank', className: 'flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[hsl(var(--muted))] text-xs font-bold text-[hsl(var(--foreground))]' }, index + 1),
          h('span', { key: 'email', className: 'truncate text-sm text-[hsl(var(--muted-foreground))]' }, user.email),
        ]),
        h('span', { key: 'amount', className: 'ml-3 shrink-0 text-sm font-medium text-[hsl(var(--foreground))]' }, `¥${formatMoney(user.amount)}`),
      ])))
      : h('div', { key: 'empty', className: 'flex h-32 items-center justify-center text-sm text-[hsl(var(--muted-foreground))]' }, copy.noData),
  ])
}

export default function PaymentDashboardPage({ days, loading, stats, copy, onDaysChange, onRefresh }: PaymentDashboardPageProps) {
  return h('div', { className: 'space-y-6' }, [
    h('div', { key: 'controls', className: 'flex items-center justify-end' }, h('div', { className: 'flex items-center gap-2' }, [
      h('div', { key: 'days', className: 'flex rounded-lg border border-[hsl(var(--border))]' }, DAYS.map((value, index) => h('button', {
        key: value,
        type: 'button',
        className: [
          'px-3 py-1.5 text-xs font-medium transition-colors',
          index === 0 ? 'rounded-l-lg' : '',
          index === DAYS.length - 1 ? 'rounded-r-lg' : '',
          days === value ? 'bg-[hsl(var(--foreground))] text-[hsl(var(--background))]' : 'text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))]',
        ].join(' '),
        onClick: () => onDaysChange(value),
      }, `${value}${copy.daySuffix}`))),
      h('button', { key: 'refresh', type: 'button', className: 'btn btn-secondary', title: copy.refresh, disabled: loading, onClick: onRefresh }, h(Icon, { name: 'refresh', className: loading ? 'h-5 w-5 animate-spin' : 'h-5 w-5' })),
    ])),
    loading
      ? h('div', { key: 'loading', className: 'flex items-center justify-center py-12' }, h('span', { className: 'h-8 w-8 animate-spin rounded-full border-2 border-[hsl(var(--border))] border-t-[hsl(var(--foreground))]' }))
      : stats
        ? h('div', { key: 'content', className: 'space-y-6' }, [
          h('div', { key: 'stats', className: 'grid grid-cols-2 gap-4 lg:grid-cols-4' }, [
            h(StatCard, { key: 'today-revenue', icon: 'money', label: copy.todayRevenue, value: `$${formatMoney(stats.today_amount)}`, hint: `${stats.today_count} ${copy.orders}` }),
            h(StatCard, { key: 'total-revenue', icon: 'card', label: copy.totalRevenue, value: `$${formatMoney(stats.total_amount)}`, hint: `${stats.total_count} ${copy.orders}` }),
            h(StatCard, { key: 'today-orders', icon: 'chart', label: copy.todayOrders, value: String(stats.today_count) }),
            h(StatCard, { key: 'avg-amount', icon: 'chart', label: copy.avgAmount, value: `$${formatMoney(stats.avg_amount)}` }),
          ]),
          h('section', { key: 'chart', className: 'card p-4' }, [
            h('h2', { key: 'title', className: 'mb-4 text-sm font-semibold text-[hsl(var(--foreground))]' }, copy.dailyRevenue),
            h(LineChart, { key: 'line', data: stats.daily_series || [], copy }),
          ]),
          h('div', { key: 'breakdown', className: 'grid grid-cols-1 gap-6 lg:grid-cols-2' }, [
            h(PaymentDistribution, { key: 'methods', methods: stats.payment_methods || [], copy }),
            h(TopUsers, { key: 'users', users: stats.top_users || [], copy }),
          ]),
        ])
        : null,
  ])
}
