import { useEffect, useRef, useState, type MouseEvent as ReactMouseEvent, type ReactNode } from 'react'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import type { User, UserAnnouncement, UserSubscription } from '@/types'
import { formatRelativeTime, formatRelativeWithDateTime } from '@/utils/format'
import { createShadcnElement as h } from './ui/createElement'
import { AppIcon, type AppIconName } from './ui/app-icon'
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from './ui/breadcrumb'

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
  breadcrumbs: readonly ConsoleHeaderBreadcrumb[]
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

export interface ConsoleHeaderBreadcrumb {
  label: string
  path?: string
}

function Icon({ name, className = 'h-5 w-5' }: { name: AppIconName; className?: string }): ReactNode {
  return h(AppIcon, { name, className })
}

const iconButtonClass = 'relative flex h-9 w-9 items-center justify-center rounded-xl bg-transparent text-[hsl(var(--muted-foreground))] transition-colors hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))]'
const panelClass = 'rounded-2xl border border-[hsl(var(--border))] bg-[hsl(var(--popover))] shadow-lg'

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

function HeaderBreadcrumbs({ props }: { props: ConsoleHeaderProps }): ReactNode {
  const items = props.breadcrumbs.length > 0 ? props.breadcrumbs : [{ label: props.pageTitle }]
  const navigate = (event: ReactMouseEvent<HTMLAnchorElement>, path?: string) => {
    if (!path) return
    event.preventDefault()
    props.onNavigate(path)
  }

  return h(Breadcrumb, { 'aria-label': props.pageTitle, className: 'min-w-0' }, h(BreadcrumbList, { className: 'text-sm' }, items.flatMap((item, index) => {
    const last = index === items.length - 1
    return [
      h(BreadcrumbItem, { key: `item-${index}`, className: 'min-w-0' }, last
        ? h(BreadcrumbPage, { className: 'truncate font-semibold' }, item.label)
        : h(BreadcrumbLink as unknown as React.JSXElementConstructor<Record<string, unknown>>, { render: h('a', { href: item.path || '#', onClick: (event: ReactMouseEvent<HTMLAnchorElement>) => navigate(event, item.path) }), className: 'truncate' }, item.label)),
      last ? null : h(BreadcrumbSeparator, { key: `separator-${index}` }),
    ]
  })))
}

export default function ConsoleHeader(props: ConsoleHeaderProps): ReactNode {
  const user = props.user
  const handleInternalNavigation = (event: MouseEvent, path: string) => {
    event.preventDefault()
    props.onNavigate(path)
  }

  return h('header', { className: 'app-topbar sticky top-0 z-30' }, h('div', { className: 'flex min-h-20 items-center justify-between gap-4 px-4 md:px-6' }, [
    h('div', { key: 'heading', className: 'min-w-0 flex-1' }, h(HeaderBreadcrumbs, { props })),
    h('div', { key: 'actions', className: 'flex shrink-0 items-center gap-1.5 sm:gap-2.5' }, [
      props.showTimeRangeButton && props.onRefresh ? h('button', { key: 'refresh', type: 'button', className: iconButtonClass, title: props.labels.refresh, 'aria-label': props.labels.refresh, onClick: props.onRefresh }, h(Icon, { name: 'refresh' })) : null,
      props.showDashboardCustomizeButton && props.onCustomizeDashboard ? h('button', { key: 'customize', type: 'button', className: iconButtonClass, title: props.labels.customize, 'aria-label': props.labels.customize, onClick: props.onCustomizeDashboard }, h(Icon, { name: 'grid' })) : null,
      props.showTimeRangeButton ? h(TimeRangeControl, { key: 'timeRange', props }) : null,
      user ? h(AnnouncementCenter, { key: 'announcements', props }) : null,
      h('a', { key: 'docs', href: '/docs', className: iconButtonClass, title: props.labels.docs, 'aria-label': props.labels.docs, onClick: (event: MouseEvent) => handleInternalNavigation(event, '/docs') }, h(Icon, { name: 'book' })),
      user ? h(SubscriptionProgress, { key: 'subscriptions', props }) : null,
      user ? h('div', { key: 'balance', className: 'hidden items-center gap-2 rounded-xl bg-[hsl(var(--secondary))] px-3 py-1.5 sm:flex' }, [h(Icon, { key: 'icon', name: 'dollar', className: 'h-4 w-4 text-[hsl(var(--muted-foreground))]' }), h('span', { key: 'value', className: 'text-sm font-semibold text-[hsl(var(--foreground))]' }, `$${(user.balance || 0).toFixed(2)}`)]) : null,
    ]),
  ]))
}
