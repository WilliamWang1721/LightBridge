import { type ReactNode } from 'react'
import { createShadcnElement as h } from './ui/createElement'
import { AppIcon } from './ui/app-icon'

export interface DashboardStatCardData {
  key: string
  label: string
  value: string | number
  hint?: string
  meta?: {
    text: string
    alert?: string
  }
  order: number
}

export interface DashboardStatsPageProps {
  cards: readonly DashboardStatCardData[]
}

type DashboardStatIcon = 'key' | 'chart' | 'userPlus' | 'cube' | 'database' | 'bolt' | 'clock'

const iconByKey: Record<string, DashboardStatIcon> = {
  apiKeys: 'key',
  accounts: 'database',
  todayRequests: 'chart',
  users: 'userPlus',
  todayTokens: 'cube',
  totalTokens: 'database',
  performance: 'bolt',
  avgResponse: 'clock',
}

function Icon({ name }: { name: DashboardStatIcon }): ReactNode {
  return h(AppIcon, { name, className: 'h-5 w-5 shrink-0' })
}

function StatCard({ card }: { card: DashboardStatCardData }): ReactNode {
  const icon = iconByKey[card.key] || 'chart'
  return h('section', {
    className: 'card min-w-0 p-4',
    style: { order: card.order },
  }, h('div', { className: 'flex items-start gap-3' }, [
    h('div', { key: 'icon', className: 'flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]' }, h(Icon, { name: icon })),
    h('div', { key: 'body', className: 'min-w-0' }, [
      h('p', { key: 'label', className: 'text-xs font-medium text-[hsl(var(--muted-foreground))]' }, card.label),
      h('p', { key: 'value', className: 'mt-1 text-xl font-semibold tracking-tight text-[hsl(var(--foreground))]' }, card.value),
      card.hint ? h('p', { key: 'hint', className: 'mt-1 text-xs text-[hsl(var(--muted-foreground))]' }, card.hint) : null,
      card.meta ? h('p', { key: 'meta', className: 'mt-1 text-xs text-[hsl(var(--muted-foreground))]' }, [
        card.meta.text,
        card.meta.alert ? h('span', { key: 'alert', className: 'ml-1 text-red-600 dark:text-red-400' }, card.meta.alert) : null,
      ]) : null,
    ]),
  ]))
}

export default function DashboardStats({ cards }: DashboardStatsPageProps) {
  return h('div', { className: 'grid min-w-0 grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4' }, cards.map((card) => h(StatCard, { key: card.key, card })))
}
