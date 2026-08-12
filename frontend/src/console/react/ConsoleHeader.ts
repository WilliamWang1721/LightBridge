import { useEffect, useRef, useState, type ReactNode } from 'react'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import type { User, UserAnnouncement, UserSubscription } from '@/types'
import { formatRelativeTime, formatRelativeWithDateTime } from '@/utils/format'
import { sanitizeSvg } from '@/utils/sanitize'
import { sidebarIconSvgs } from '@/console/sidebar/icons'
import { createShadcnElement as h } from './ui/createElement'

type Granularity = 'hour' | 'day'

export interface ConsoleHeaderLabels {
  refresh: string
  customize: string
  timeRange: string
  granularity: string
  hour: string
  day: string
  reset: string
  close: string
  apply: string
  startDate: string
  endDate: string
  selectDateRange: string
  presetToday: string
  presetYesterday: string
  presetLast24Hours: string
  preset7Days: string
  preset14Days: string
  preset30Days: string
  announcementsTitle: string
  announcementsUnread: string
  announcementsMarkAllRead: string
  announcementsEmpty: string
  announcementsEmptyDescription: string
  announcementsRead: string
  announcementsMarkRead: string
  announcementsReadStatus: string
  announcementsMarkReadHint: string
  docs: string
  balance: string
  profile: string
  apiKeys: string
  github: string
  contactSupport: string
  restartTour: string
  logout: string
  subscriptionTitle: string
  subscriptionActiveCount: (count: number) => string
  subscriptionUnlimited: string
  subscriptionDaily: string
  subscriptionWeekly: string
  subscriptionMonthly: string
  subscriptionExpired: string
  subscriptionExpiresToday: string
  subscriptionExpiresTomorrow: string
  subscriptionDaysRemaining: (days: number) => string
  subscriptionViewAll: string
  formatDateRange: (start: string, end: string, locale: 'zh' | 'en') => string
}

export interface ConsoleHeaderProps {
  pageTitle: string
  pageDescription: string
  user?: User | null
  isAdmin: boolean
  contactInfo?: string | null
  showOnboardingButton: boolean
  showTimeRangeButton: boolean
  showDashboardCustomizeButton: boolean
  startDate: string
  endDate: string
  granularity: Granularity
  locale: 'zh' | 'en'
  announcements?: UserAnnouncement[]
  announcementLoading?: boolean
  subscriptions?: UserSubscription[]
  labels: ConsoleHeaderLabels
  onRefresh?: () => void
  onCustomizeDashboard?: () => void
  onTimeRangeApply?: (start: string, end: string, granularity: Granularity) => void
  onTimeRangeReset?: () => void
  onMarkAnnouncementRead?: (id: number) => void | Promise<void>
  onMarkAllAnnouncementsRead?: () => void | Promise<void>
  onNavigate: (path: string) => void
  onLogout: () => void | Promise<void>
  onReplayGuide?: () => void
}

const icon = (body: string) => `<svg fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">${body}</svg>`

const iconSvgs = {
  ...sidebarIconSvgs,
  refresh: icon('<path d="M20.25 12a8.25 8.25 0 0 0-14.1-5.83L3.75 8.57m0 0V3.75m0 4.82h4.82M3.75 12a8.25 8.25 0 0 0 14.1 5.83l2.4-2.4m0 0v4.82m0-4.82h-4.82"/>'),
  clock: icon('<circle cx="12" cy="12" r="8.75"/><path d="M12 7.5v5.25l3.75 2.25"/>'),
  grid: sidebarIconSvgs.dashboard,
  chevronDown: icon('<path d="m6.75 9 5.25 5.25L17.25 9"/>'),
  x: icon('<path d="M6 6l12 12M18 6 6 18"/>'),
  logout: icon('<path d="M15.75 9V5.25A2.25 2.25 0 0 0 13.5 3h-6a2.25 2.25 0 0 0-2.25 2.25v13.5A2.25 2.25 0 0 0 7.5 21h6a2.25 2.25 0 0 0 2.25-2.25V15m-3-3h9m0 0-3-3m3 3-3 3"/>'),
  check: icon('<path d="m5 12 4 4L19 6"/>'),
  dollar: icon('<circle cx="12" cy="12" r="8.75"/><path d="M12 6v12m-3-2.25.75.56c1.24.93 3.26.93 4.5 0 1.24-.93 1.24-2.44 0-3.37-.47-.35-1.08-.54-1.69-.54-.61 0-1.22-.19-1.69-.54-1.18-.93-1.18-2.44 0-3.37 1.18-.93 3.09-.93 4.27 0l.44.35"/>'),
  lightbulb: icon('<path d="M9.75 18h4.5m-3.75 3h3m-5.7-7.02A6.75 6.75 0 1 1 16.2 13.98c-.84.5-1.45 1.39-1.45 2.37V18h-5.5v-1.65c0-.98-.61-1.87-1.45-2.37Z"/>'),
  book: icon('<path d="M12 6.04A8.97 8.97 0 0 0 6 3.75c-1.05 0-2.06.18-3 .51v14.25A8.99 8.99 0 0 1 6 18c2.3 0 4.41.87 6 2.29m0-14.25a8.97 8.97 0 0 1 6-2.29c1.05 0 2.06.18 3 .51v14.25A8.99 8.99 0 0 0 18 18a8.97 8.97 0 0 0-6 2.29m0-14.25v14.25"/>'),
  inbox: icon('<path d="M20 13V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v7m16 0v5a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-5m16 0h-2.59a1 1 0 0 0-.71.29l-2.41 2.42a1 1 0 0 1-.71.29h-3.17a1 1 0 0 1-.71-.29l-2.41-2.42A1 1 0 0 0 6.59 13H4"/>'),
  github: '<svg fill="currentColor" viewBox="0 0 24 24"><path fill-rule="evenodd" clip-rule="evenodd" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0 1 12 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z"/></svg>',
} as const

function Icon({ name, className = '' }: { name: keyof typeof iconSvgs; className?: string }): ReactNode {
  return h('span', {
    className: `inline-flex h-5 w-5 shrink-0 [&>svg]:h-full [&>svg]:w-full ${className}`,
    'aria-hidden': 'true',
    dangerouslySetInnerHTML: { __html: sanitizeSvg(iconSvgs[name]) },
  })
}

const iconButtonClass = 'relative flex h-9 w-9 items-center justify-center rounded-xl text-[hsl(var(--muted-foreground))] transition-colors hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))]'
const panelClass = 'rounded-2xl border border-[hsl(var(--border))] bg-[hsl(var(--background))] shadow-2xl'

function dateString(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function shiftDate(days: number): string {
  const date = new Date()
  date.setDate(date.getDate() + days)
  return dateString(date)
}

function TimeRangeControl({ props }: { props: ConsoleHeaderProps }): ReactNode {
  const [open, setOpen] = useState(false)
  const [draftStart, setDraftStart] = useState(props.startDate)
  const [draftEnd, setDraftEnd] = useState(props.endDate)
  const [draftGranularity, setDraftGranularity] = useState<Granularity>(props.granularity)
  const ref = useRef<HTMLDivElement | null>(null)
  const tomorrow = shiftDate(1)
  const labels = props.labels

  useEffect(() => {
    if (!open) return
    setDraftStart(props.startDate)
    setDraftEnd(props.endDate)
    setDraftGranularity(props.granularity)
  }, [open, props.startDate, props.endDate, props.granularity])

  useEffect(() => {
    if (!open) return
    const onDocumentClick = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDocumentClick)
    return () => document.removeEventListener('mousedown', onDocumentClick)
  }, [open])

  const presets = [
    { value: 'today', label: labels.presetToday, range: () => ({ start: dateString(new Date()), end: dateString(new Date()) }) },
    { value: 'yesterday', label: labels.presetYesterday, range: () => ({ start: shiftDate(-1), end: shiftDate(-1) }) },
    { value: 'last24Hours', label: labels.presetLast24Hours, range: () => ({ start: shiftDate(-1), end: dateString(new Date()) }) },
    { value: '7days', label: labels.preset7Days, range: () => ({ start: shiftDate(-6), end: dateString(new Date()) }) },
    { value: '14days', label: labels.preset14Days, range: () => ({ start: shiftDate(-13), end: dateString(new Date()) }) },
    { value: '30days', label: labels.preset30Days, range: () => ({ start: shiftDate(-29), end: dateString(new Date()) }) },
  ]

  const activePreset = presets.find((preset) => {
    const range = preset.range()
    return range.start === draftStart && range.end === draftEnd
  })
  const valid = Boolean(draftStart && draftEnd && draftStart <= draftEnd)

  const selectPreset = (preset: typeof presets[number]) => {
    const range = preset.range()
    setDraftStart(range.start)
    setDraftEnd(range.end)
  }

  return h('div', { ref }, [
    h('button', {
      key: 'trigger',
      type: 'button',
      className: iconButtonClass,
      title: labels.timeRange,
      'aria-label': labels.timeRange,
      'aria-expanded': open,
      onClick: () => setOpen((value) => !value),
    }, h(Icon, { name: 'clock' })),
    open ? h('div', {
      key: 'panel',
      className: `absolute right-0 top-11 z-50 w-[min(24rem,calc(100vw-1rem))] p-4 ${panelClass}`,
      role: 'dialog',
      'aria-label': labels.timeRange,
    }, [
      h('h2', { key: 'title', className: 'text-sm font-semibold text-[hsl(var(--foreground))]' }, labels.timeRange),
      h('div', { key: 'presets', className: 'mt-3 grid grid-cols-2 gap-1.5' }, presets.map((preset) => h('button', {
        key: preset.value,
        type: 'button',
        className: `rounded-lg px-2.5 py-2 text-left text-xs transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] ${activePreset?.value === preset.value ? 'bg-[hsl(var(--foreground))] text-[hsl(var(--background))]' : 'text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]'}`,
        'aria-pressed': activePreset?.value === preset.value,
        onClick: () => selectPreset(preset),
      }, preset.label))),
      h('div', { key: 'range', className: 'mt-4 grid grid-cols-2 gap-2 border-t border-[hsl(var(--border))] pt-4' }, [
        h('label', { key: 'start', className: 'space-y-1.5 text-xs text-[hsl(var(--muted-foreground))]' }, [
          h('span', { key: 'label' }, labels.startDate),
          h('input', { key: 'input', className: 'input h-9 w-full px-2 text-xs', type: 'date', value: draftStart, max: draftEnd || tomorrow, onChange: (event: Event) => setDraftStart((event.target as HTMLInputElement).value) }),
        ]),
        h('label', { key: 'end', className: 'space-y-1.5 text-xs text-[hsl(var(--muted-foreground))]' }, [
          h('span', { key: 'label' }, labels.endDate),
          h('input', { key: 'input', className: 'input h-9 w-full px-2 text-xs', type: 'date', value: draftEnd, min: draftStart, max: tomorrow, onChange: (event: Event) => setDraftEnd((event.target as HTMLInputElement).value) }),
        ]),
      ]),
      h('div', { key: 'granularity', className: 'mt-4' }, [
        h('span', { key: 'label', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, labels.granularity),
        h('div', { key: 'options', className: 'mt-2 grid grid-cols-2 gap-2', role: 'group' }, [
          h('button', { key: 'hour', type: 'button', className: `rounded-lg border px-2 py-2 text-xs transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] ${draftGranularity === 'hour' ? 'border-[hsl(var(--foreground))] bg-[hsl(var(--foreground))] text-[hsl(var(--background))]' : 'border-[hsl(var(--border))] text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))]'}`, 'aria-pressed': draftGranularity === 'hour', onClick: () => setDraftGranularity('hour') }, labels.hour),
          h('button', { key: 'day', type: 'button', className: `rounded-lg border px-2 py-2 text-xs transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] ${draftGranularity === 'day' ? 'border-[hsl(var(--foreground))] bg-[hsl(var(--foreground))] text-[hsl(var(--background))]' : 'border-[hsl(var(--border))] text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))]'}`, 'aria-pressed': draftGranularity === 'day', onClick: () => setDraftGranularity('day') }, labels.day),
        ]),
      ]),
      h('div', { key: 'footer', className: 'mt-4 flex justify-between gap-3 border-t border-[hsl(var(--border))] pt-3' }, [
        h('button', { key: 'reset', type: 'button', className: 'rounded-lg px-2.5 py-2 text-xs text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))]', onClick: () => { props.onTimeRangeReset?.() } }, labels.reset),
        h('div', { key: 'actions', className: 'flex gap-2' }, [
          h('button', { key: 'close', type: 'button', className: 'rounded-lg px-2.5 py-2 text-xs text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))]', onClick: () => setOpen(false) }, labels.close),
          h('button', { key: 'apply', type: 'button', className: 'rounded-lg bg-[hsl(var(--foreground))] px-3 py-2 text-xs font-medium text-[hsl(var(--background))] disabled:cursor-not-allowed disabled:opacity-40', disabled: !valid, onClick: () => { if (valid) { props.onTimeRangeApply?.(draftStart, draftEnd, draftGranularity); setOpen(false) } } }, labels.apply),
        ]),
      ]),
    ]) : null,
  ])
}

function AnnouncementCenter({ props }: { props: ConsoleHeaderProps }): ReactNode {
  const [open, setOpen] = useState(false)
  const [selected, setSelected] = useState<UserAnnouncement | null>(null)
  const announcements = props.announcements || []
  const unreadCount = announcements.filter((item) => !item.read_at).length
  const labels = props.labels

  useEffect(() => {
    if (!open && !selected) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setSelected(null)
        setOpen(false)
      }
    }
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.body.style.overflow = previousOverflow
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open, selected])

  const openDetail = (item: UserAnnouncement) => {
    setSelected(item)
    if (!item.read_at) void props.onMarkAnnouncementRead?.(item.id)
  }

  const modal = selected
    ? h('div', { className: 'fixed inset-0 z-[110] flex items-start justify-center overflow-y-auto bg-black/60 p-4 pt-[6vh] backdrop-blur-sm', onClick: () => setSelected(null) }, h('article', { className: `w-full max-w-[760px] overflow-hidden ${panelClass}`, onClick: (event: MouseEvent) => event.stopPropagation() }, [
      h('header', { key: 'header', className: 'flex items-start justify-between gap-4 border-b border-[hsl(var(--border))] p-6' }, [
        h('div', { key: 'title', className: 'min-w-0' }, [
          h('div', { key: 'eyebrow', className: 'mb-3 inline-flex items-center gap-2 text-xs text-[hsl(var(--muted-foreground))]' }, [h(Icon, { key: 'icon', name: 'bell', className: 'h-4 w-4' }), labels.announcementsTitle]),
          h('h2', { key: 'heading', className: 'text-xl font-semibold text-[hsl(var(--foreground))]' }, selected.title),
          h('div', { key: 'meta', className: 'mt-2 text-xs text-[hsl(var(--muted-foreground))]' }, `${formatRelativeWithDateTime(selected.created_at)} · ${selected.read_at ? labels.announcementsRead : labels.announcementsUnread}`),
        ]),
        h('button', { key: 'close', type: 'button', className: iconButtonClass, 'aria-label': labels.close, onClick: () => setSelected(null) }, h(Icon, { name: 'x' })),
      ]),
      h('div', { key: 'body', className: 'markdown-body max-h-[60vh] overflow-y-auto p-6 text-sm leading-7 text-[hsl(var(--foreground))]', dangerouslySetInnerHTML: { __html: DOMPurify.sanitize(marked.parse(selected.content || '') as string) } }),
      h('footer', { key: 'footer', className: 'flex items-center justify-end gap-2 border-t border-[hsl(var(--border))] p-4' }, [
        h('button', { key: 'close', type: 'button', className: 'rounded-lg px-3 py-2 text-xs text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))]', onClick: () => setSelected(null) }, labels.close),
        !selected.read_at ? h('button', { key: 'read', type: 'button', className: 'rounded-lg bg-[hsl(var(--foreground))] px-3 py-2 text-xs font-medium text-[hsl(var(--background))]', onClick: () => { void props.onMarkAnnouncementRead?.(selected.id); setSelected(null) } }, labels.announcementsMarkRead) : null,
      ]),
    ]))
    : h('div', { className: 'fixed inset-0 z-[100] flex items-start justify-center overflow-y-auto bg-black/60 p-4 pt-[8vh] backdrop-blur-sm', onClick: () => setOpen(false) }, h('section', { className: `w-full max-w-[620px] overflow-hidden ${panelClass}`, onClick: (event: MouseEvent) => event.stopPropagation() }, [
      h('header', { key: 'header', className: 'flex items-start justify-between gap-4 border-b border-[hsl(var(--border))] p-5' }, [
        h('div', { key: 'title' }, [
          h('div', { key: 'heading', className: 'flex items-center gap-2 text-base font-semibold text-[hsl(var(--foreground))]' }, [h(Icon, { key: 'icon', name: 'bell', className: 'h-4 w-4' }), labels.announcementsTitle]),
          unreadCount > 0 ? h('p', { key: 'unread', className: 'mt-1 text-xs text-[hsl(var(--muted-foreground))]' }, `${unreadCount} ${labels.announcementsUnread}`) : null,
        ]),
        h('div', { key: 'actions', className: 'flex items-center gap-2' }, [
          unreadCount > 0 ? h('button', { key: 'all', type: 'button', className: 'rounded-lg bg-[hsl(var(--foreground))] px-3 py-2 text-xs font-medium text-[hsl(var(--background))] disabled:opacity-50', disabled: props.announcementLoading, onClick: () => { void props.onMarkAllAnnouncementsRead?.() } }, labels.announcementsMarkAllRead) : null,
          h('button', { key: 'close', type: 'button', className: iconButtonClass, 'aria-label': labels.close, onClick: () => setOpen(false) }, h(Icon, { name: 'x' })),
        ]),
      ]),
      props.announcementLoading ? h('div', { key: 'loading', className: 'flex h-48 items-center justify-center text-sm text-[hsl(var(--muted-foreground))]', role: 'status' }, '…') : announcements.length > 0 ? h('div', { key: 'list', className: 'max-h-[65vh] overflow-y-auto' }, announcements.map((item) => h('button', { key: item.id, type: 'button', className: `group relative flex w-full items-center gap-3 border-b border-[hsl(var(--border))] px-5 py-4 text-left transition-colors hover:bg-[hsl(var(--muted)/0.55)] ${!item.read_at ? 'bg-[hsl(var(--muted)/0.35)]' : ''}`, onClick: () => openDetail(item) }, [
        h('span', { key: 'status', className: `flex h-9 w-9 shrink-0 items-center justify-center rounded-xl ${item.read_at ? 'bg-[hsl(var(--muted))] text-[hsl(var(--muted-foreground))]' : 'bg-[hsl(var(--foreground))] text-[hsl(var(--background))]'}` }, h(Icon, { name: item.read_at ? 'check' : 'bell', className: 'h-4 w-4' })),
        h('span', { key: 'content', className: 'min-w-0 flex-1' }, [h('span', { key: 'title', className: 'block truncate text-sm font-medium text-[hsl(var(--foreground))]' }, item.title), h('span', { key: 'meta', className: 'mt-1 block text-xs text-[hsl(var(--muted-foreground))]' }, `${formatRelativeTime(item.created_at)}${item.read_at ? '' : ` · ${labels.announcementsUnread}`}`)]),
        h(Icon, { key: 'arrow', name: 'chevronDown', className: 'rotate-[-90deg] text-[hsl(var(--muted-foreground))]' }),
      ]))) : h('div', { key: 'empty', className: 'flex flex-col items-center justify-center px-5 py-16 text-center' }, [h(Icon, { key: 'icon', name: 'inbox', className: 'mb-3 h-10 w-10 text-[hsl(var(--muted-foreground))]' }), h('p', { key: 'title', className: 'text-sm font-medium text-[hsl(var(--foreground))]' }, labels.announcementsEmpty), h('p', { key: 'description', className: 'mt-1 text-xs text-[hsl(var(--muted-foreground))]' }, labels.announcementsEmptyDescription)]),
    ]))

  return h('div', {}, [
    h('button', { key: 'trigger', type: 'button', className: iconButtonClass, title: labels.announcementsTitle, 'aria-label': labels.announcementsTitle, 'aria-expanded': open || Boolean(selected), onClick: () => { setOpen(true); setSelected(null) } }, [h(Icon, { key: 'icon', name: 'bell' }), unreadCount > 0 ? h('span', { key: 'badge', className: 'absolute right-1 top-1 h-2 w-2 rounded-full bg-[hsl(var(--foreground))]', 'aria-label': `${unreadCount} ${labels.announcementsUnread}` }) : null]),
    open || selected ? modal : null,
  ])
}

function getMaxUsagePercentage(subscription: UserSubscription): number {
  const percentages = [
    subscription.group?.daily_limit_usd ? ((subscription.daily_usage_usd || 0) / subscription.group.daily_limit_usd) * 100 : 0,
    subscription.group?.weekly_limit_usd ? ((subscription.weekly_usage_usd || 0) / subscription.group.weekly_limit_usd) * 100 : 0,
    subscription.group?.monthly_limit_usd ? ((subscription.monthly_usage_usd || 0) / subscription.group.monthly_limit_usd) * 100 : 0,
  ]
  return Math.max(...percentages)
}

function SubscriptionProgress({ props }: { props: ConsoleHeaderProps }): ReactNode {
  const [open, setOpen] = useState(false)
  const subscriptions = [...(props.subscriptions || [])].sort((a, b) => getMaxUsagePercentage(b) - getMaxUsagePercentage(a))
  if (subscriptions.length === 0) return null
  const labels = props.labels

  const isUnlimited = (subscription: UserSubscription) => !subscription.group?.daily_limit_usd && !subscription.group?.weekly_limit_usd && !subscription.group?.monthly_limit_usd
  const progressClass = (value: number | undefined, limit: number | null | undefined) => {
    if (!limit) return 'bg-[hsl(var(--muted-foreground))]'
    const percentage = ((value || 0) / limit) * 100
    if (percentage >= 90) return 'bg-red-500'
    if (percentage >= 70) return 'bg-amber-500'
    return 'bg-[hsl(var(--foreground))]'
  }
  const progress = (value: number | undefined, limit: number | null | undefined) => `${Math.min(((value || 0) / (limit || 1)) * 100, 100)}%`
  const daysRemaining = (expiresAt: string | null) => {
    if (!expiresAt) return ''
    const expiry = new Date(expiresAt)
    const diff = expiry.getTime() - Date.now()
    if (Number.isNaN(expiry.getTime()) || diff <= 0) return labels.subscriptionExpired
    const days = Math.round((new Date(expiry.getFullYear(), expiry.getMonth(), expiry.getDate()).getTime() - new Date(new Date().getFullYear(), new Date().getMonth(), new Date().getDate()).getTime()) / 86400000)
    if (days <= 0) return labels.subscriptionExpiresToday
    if (days === 1) return labels.subscriptionExpiresTomorrow
    return labels.subscriptionDaysRemaining(days)
  }
  const metric = (label: string, value: number | undefined, limit: number | null | undefined) => limit ? h('div', { key: label, className: 'flex items-center gap-2' }, [h('span', { key: 'label', className: 'w-10 shrink-0 text-[10px] text-[hsl(var(--muted-foreground))]' }, label), h('div', { key: 'bar', className: 'h-1.5 min-w-0 flex-1 rounded-full bg-[hsl(var(--muted))]' }, h('div', { className: `h-1.5 rounded-full ${progressClass(value, limit)}`, style: { width: progress(value, limit) } })), h('span', { key: 'value', className: 'w-24 shrink-0 text-right text-[10px] text-[hsl(var(--muted-foreground))]' }, `$${(value || 0).toFixed(2)}/$${limit.toFixed(2)}`)]) : null

  return h('div', { className: 'relative' }, [
    h('button', { key: 'trigger', type: 'button', className: 'flex items-center gap-2 rounded-xl bg-[hsl(var(--muted))] px-3 py-1.5 text-[hsl(var(--foreground))] transition-colors hover:bg-[hsl(var(--border))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))]', title: labels.subscriptionTitle, 'aria-expanded': open, onClick: () => setOpen((value) => !value) }, [h(Icon, { key: 'icon', name: 'creditCard', className: 'h-4 w-4' }), h('span', { key: 'dots', className: 'flex items-center gap-0.5' }, subscriptions.slice(0, 3).map((subscription) => h('span', { key: subscription.id, className: `h-2 w-2 rounded-full ${isUnlimited(subscription) ? 'bg-[hsl(var(--foreground))]' : progressClass(getMaxUsagePercentage(subscription), 100)}` }))), h('span', { key: 'count', className: 'text-xs font-medium' }, subscriptions.length)]),
    open ? h('div', { key: 'panel', className: `absolute right-0 top-11 z-50 w-[min(340px,calc(100vw-1rem))] overflow-hidden ${panelClass}` }, [
      h('header', { key: 'header', className: 'border-b border-[hsl(var(--border))] p-3' }, [h('h3', { key: 'title', className: 'text-sm font-semibold text-[hsl(var(--foreground))]' }, labels.subscriptionTitle), h('p', { key: 'count', className: 'mt-0.5 text-xs text-[hsl(var(--muted-foreground))]' }, labels.subscriptionActiveCount(subscriptions.length))]),
      h('div', { key: 'list', className: 'max-h-64 overflow-y-auto' }, subscriptions.map((subscription) => h('div', { key: subscription.id, className: 'border-b border-[hsl(var(--border))] p-3 last:border-b-0' }, [
        h('div', { key: 'name', className: 'mb-2 flex items-center justify-between gap-2' }, [h('span', { key: 'label', className: 'truncate text-sm font-medium text-[hsl(var(--foreground))]' }, subscription.group?.name || `Group #${subscription.group_id}`), subscription.expires_at ? h('span', { key: 'expiry', className: 'shrink-0 text-xs text-[hsl(var(--muted-foreground))]' }, daysRemaining(subscription.expires_at)) : null]),
        isUnlimited(subscription) ? h('div', { key: 'unlimited', className: 'rounded-lg bg-[hsl(var(--muted))] px-2.5 py-1.5 text-xs font-medium text-[hsl(var(--foreground))]' }, `∞ ${labels.subscriptionUnlimited}`) : h('div', { key: 'metrics', className: 'space-y-1.5' }, [metric(labels.subscriptionDaily, subscription.daily_usage_usd, subscription.group?.daily_limit_usd), metric(labels.subscriptionWeekly, subscription.weekly_usage_usd, subscription.group?.weekly_limit_usd), metric(labels.subscriptionMonthly, subscription.monthly_usage_usd, subscription.group?.monthly_limit_usd)]),
      ]))),
      h('button', { key: 'viewAll', type: 'button', className: 'w-full border-t border-[hsl(var(--border))] px-3 py-2 text-center text-xs text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]', onClick: () => { setOpen(false); props.onNavigate('/subscriptions') } }, labels.subscriptionViewAll),
    ]) : null,
  ])
}

function UserMenu({ props }: { props: ConsoleHeaderProps }): ReactNode {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement | null>(null)
  const user = props.user
  const close = () => setOpen(false)
  useEffect(() => {
    if (!open) return
    const onDocumentClick = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) close()
    }
    document.addEventListener('mousedown', onDocumentClick)
    return () => document.removeEventListener('mousedown', onDocumentClick)
  }, [open])
  if (!user) return null
  const displayName = user.username || user.email?.split('@')[0] || ''
  const initials = (user.username || user.email?.split('@')[0] || '').slice(0, 2).toUpperCase()
  const navigate = (path: string) => { close(); props.onNavigate(path) }
  const menuItem = (key: string, iconName: keyof typeof iconSvgs, label: string, action: () => void) => h('button', { key, type: 'button', className: 'flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-[hsl(var(--foreground))] transition-colors hover:bg-[hsl(var(--muted))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[hsl(var(--ring))]', onClick: action }, [h(Icon, { key: 'icon', name: iconName, className: 'h-4 w-4 text-[hsl(var(--muted-foreground))]' }), label])

  return h('div', { ref, className: 'relative' }, [
    h('button', { key: 'trigger', type: 'button', className: 'flex items-center gap-2 rounded-xl p-1.5 transition-colors hover:bg-[hsl(var(--muted))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))]', 'aria-label': 'User menu', 'aria-expanded': open, 'aria-controls': 'lightbridge-user-menu', onClick: () => setOpen((value) => !value) }, [
      h('span', { key: 'avatar', className: 'flex h-8 w-8 items-center justify-center overflow-hidden rounded-xl bg-[hsl(var(--foreground))] text-sm font-medium text-[hsl(var(--background))]' }, user.avatar_url ? h('img', { src: user.avatar_url, alt: displayName, className: 'h-full w-full object-cover' }) : initials),
      h('span', { key: 'name', className: 'hidden text-left md:block' }, [h('span', { key: 'display', className: 'block text-sm font-medium text-[hsl(var(--foreground))]' }, displayName), h('span', { key: 'role', className: 'block text-xs capitalize text-[hsl(var(--muted-foreground))]' }, user.role)]),
      h(Icon, { key: 'chevron', name: 'chevronDown', className: 'hidden h-4 w-4 text-[hsl(var(--muted-foreground))] md:block' }),
    ]),
    open ? h('div', { key: 'menu', id: 'lightbridge-user-menu', className: `absolute right-0 top-11 z-50 w-56 overflow-hidden ${panelClass}`, role: 'menu' }, [
      h('div', { key: 'info', className: 'border-b border-[hsl(var(--border))] px-4 py-3' }, [h('div', { key: 'name', className: 'text-sm font-medium text-[hsl(var(--foreground))]' }, displayName), h('div', { key: 'email', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, user.email)]),
      h('div', { key: 'mobileBalance', className: 'border-b border-[hsl(var(--border))] px-4 py-2 sm:hidden' }, [h('div', { key: 'label', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, props.labels.balance), h('div', { key: 'value', className: 'text-sm font-semibold text-[hsl(var(--foreground))]' }, `$${(user.balance || 0).toFixed(2)}`)]),
      h('div', { key: 'links', className: 'py-1' }, [menuItem('profile', 'user', props.labels.profile, () => navigate('/profile')), menuItem('keys', 'key', props.labels.apiKeys, () => navigate('/keys')), props.isAdmin ? h('a', { key: 'github', href: 'https://github.com/WilliamWang1721/LightBridge', target: '_blank', rel: 'noopener noreferrer', className: 'flex items-center gap-2 px-4 py-2 text-sm text-[hsl(var(--foreground))] hover:bg-[hsl(var(--muted))]', onClick: close }, [h(Icon, { key: 'icon', name: 'github', className: 'h-4 w-4 text-[hsl(var(--muted-foreground))]' }), props.labels.github]) : null]),
      props.contactInfo ? h('div', { key: 'contact', className: 'border-t border-[hsl(var(--border))] px-4 py-2.5 text-xs text-[hsl(var(--muted-foreground))]' }, `${props.labels.contactSupport}: ${props.contactInfo}`) : null,
      props.showOnboardingButton ? h('div', { key: 'guide', className: 'border-t border-[hsl(var(--border))] py-1' }, menuItem('guide', 'lightbulb', props.labels.restartTour, () => { close(); props.onReplayGuide?.() })) : null,
      h('div', { key: 'logout', className: 'border-t border-[hsl(var(--border))] py-1' }, menuItem('logout', 'logout', props.labels.logout, () => { close(); void props.onLogout() })),
    ]) : null,
  ])
}

export default function ConsoleHeader(props: ConsoleHeaderProps): ReactNode {
  const user = props.user
  const handleInternalNavigation = (event: MouseEvent, path: string) => {
    event.preventDefault()
    props.onNavigate(path)
  }

  return h('header', { className: 'app-topbar sticky top-0 z-30' }, h('div', { className: 'flex min-h-20 items-center justify-between gap-4 px-4 md:px-6' }, [
    h('div', { key: 'heading', className: 'min-w-0' }, [h('h1', { key: 'title', className: 'app-topbar-title truncate text-[hsl(var(--foreground))]' }, props.pageTitle), props.pageDescription ? h('p', { key: 'description', className: 'app-topbar-description truncate text-[hsl(var(--muted-foreground))]' }, props.pageDescription) : null]),
    h('div', { key: 'actions', className: 'flex shrink-0 items-center gap-1.5 sm:gap-2.5' }, [
      props.showTimeRangeButton && props.onRefresh ? h('button', { key: 'refresh', type: 'button', className: iconButtonClass, title: props.labels.refresh, 'aria-label': props.labels.refresh, onClick: props.onRefresh }, h(Icon, { name: 'refresh' })) : null,
      props.showDashboardCustomizeButton && props.onCustomizeDashboard ? h('button', { key: 'customize', type: 'button', className: iconButtonClass, title: props.labels.customize, 'aria-label': props.labels.customize, onClick: props.onCustomizeDashboard }, h(Icon, { name: 'grid' })) : null,
      props.showTimeRangeButton ? h(TimeRangeControl, { key: 'timeRange', props }) : null,
      user ? h(AnnouncementCenter, { key: 'announcements', props }) : null,
      h('a', { key: 'docs', href: '/docs', className: iconButtonClass, title: props.labels.docs, 'aria-label': props.labels.docs, onClick: (event: MouseEvent) => handleInternalNavigation(event, '/docs') }, h(Icon, { name: 'book' })),
      user ? h(SubscriptionProgress, { key: 'subscriptions', props }) : null,
      user ? h('div', { key: 'balance', className: 'hidden items-center gap-2 rounded-xl bg-[hsl(var(--secondary))] px-3 py-1.5 sm:flex' }, [h(Icon, { key: 'icon', name: 'dollar', className: 'h-4 w-4 text-[hsl(var(--muted-foreground))]' }), h('span', { key: 'value', className: 'text-sm font-semibold text-[hsl(var(--foreground))]' }, `$${(user.balance || 0).toFixed(2)}`)]) : null,
      user ? h(UserMenu, { key: 'menu', props }) : null,
    ]),
  ]))
}
