import { type ReactNode } from 'react'
import { sanitizeSvg } from '@/utils/sanitize'
import { sidebarIconSvgs } from '@/console/sidebar/icons'
import { createShadcnElement as h } from './ui/createElement'

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

type DashboardStatIcon = 'key' | 'server' | 'chart' | 'userPlus' | 'cube' | 'database' | 'bolt' | 'clock'

const iconSvgs: Record<DashboardStatIcon, string> = {
  key: sidebarIconSvgs.key,
  server: sidebarIconSvgs.server,
  chart: sidebarIconSvgs.chart,
  userPlus: sidebarIconSvgs.user,
  cube: '<svg fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="m12 3 8.25 4.5v9L12 21l-8.25-4.5v-9L12 3Z"/><path d="m3.75 7.5 8.25 4.5 8.25-4.5M12 12v9"/></svg>',
  database: sidebarIconSvgs.database,
  bolt: '<svg fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="m13.5 2.25-9 11.25h6.75l-.75 8.25 9-11.25h-6.75l.75-8.25Z"/></svg>',
  clock: '<svg fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="8.75"/><path d="M12 7.5v5.25l3.75 2.25"/></svg>',
}

const iconByKey: Record<string, DashboardStatIcon> = {
  apiKeys: 'key',
  accounts: 'server',
  todayRequests: 'chart',
  users: 'userPlus',
  todayTokens: 'cube',
  totalTokens: 'database',
  performance: 'bolt',
  avgResponse: 'clock',
}

function Icon({ name }: { name: DashboardStatIcon }): ReactNode {
  return h('span', {
    className: 'h-5 w-5 shrink-0 [&>svg]:h-full [&>svg]:w-full',
    'aria-hidden': 'true',
    dangerouslySetInnerHTML: { __html: sanitizeSvg(iconSvgs[name]) },
  })
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
