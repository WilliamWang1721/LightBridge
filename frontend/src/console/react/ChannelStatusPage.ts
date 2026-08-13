import { useState, type ReactNode } from 'react'
import type { UserMonitorDetail, UserMonitorModelDetail, UserMonitorView } from '@/api/channelMonitor'
import { createShadcnElement as h } from './ui/createElement'
import { AppIcon } from './ui/app-icon'

export interface ChannelStatusCopy {
  windowTab: Record<'7d' | '15d' | '30d', string>
  overall: Record<'operational' | 'degraded', string>
  status: Record<string, string>
  providers: Record<string, string>
  refresh: string
  detailTitle: string
  closeDetail: string
  loading: string
  detailLoadError: string
  emptyTitle: string
  emptyDescription: string
  dialogLatency: string
  endpointPing: string
  availability: string
  extraModelsCount: (count: number) => string
  nextUpdateIn: (count: number) => string
  pollEvery: (seconds: number) => string
  detailColumns: {
    model: string
    latestStatus: string
    latestLatency: string
    availability7d: string
    availability15d: string
    availability30d: string
    avgLatency7d: string
  }
}

export interface ChannelStatusPageProps {
  items: UserMonitorView[]
  loading: boolean
  overallStatus: 'operational' | 'degraded'
  currentWindow: '7d' | '15d' | '30d'
  countdown: number
  detailCache: Record<number, UserMonitorDetail>
  autoRefresh: { enabled: boolean; intervalSeconds: number; intervals: readonly number[] }
  copy: ChannelStatusCopy
  onWindowChange: (value: '7d' | '15d' | '30d') => void
  onRefresh: () => void
  onAutoRefreshChange: (enabled: boolean) => void
  onIntervalChange: (seconds: number) => void
  onCardClick: (item: UserMonitorView) => void
  onLoadDetail: (id: number) => Promise<void> | void
}

const latency = (value: number | null | undefined) => value == null ? '-' : String(Math.round(value))
const percent = (value: number | null | undefined) => value == null || Number.isNaN(value) ? '-' : `${value.toFixed(2)}%`

function Icon({ name }: { name: 'refresh' | 'bolt' | 'globe' | 'clock' }): ReactNode {
  return h(AppIcon, { name })
}

function MonitorCard({ item, window, countdown, copy, onClick }: { item: UserMonitorView; window: '7d' | '15d' | '30d'; countdown: number; copy: ChannelStatusCopy; onClick: () => void }): ReactNode {
  const status = copy.status[item.primary_status] || copy.status.unknown || item.primary_status
  const availability = item.availability_7d
  return h('button', { type: 'button', className: 'card group flex min-h-[250px] w-full flex-col p-5 text-left transition-transform hover:-translate-y-0.5 hover:bg-[hsl(var(--muted)/.25)]', onClick }, [
    h('div', { key: 'header', className: 'flex items-start gap-3' }, [
      h('span', { key: 'provider', className: 'flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--muted))] text-xs font-semibold uppercase text-[hsl(var(--foreground))]' }, item.provider.slice(0, 1)),
      h('div', { key: 'name', className: 'min-w-0 flex-1' }, [h('div', { key: 'title', className: 'truncate text-base font-semibold text-[hsl(var(--foreground))]' }, item.name), h('div', { key: 'meta', className: 'mt-1 flex min-w-0 items-center gap-1.5 text-xs text-[hsl(var(--muted-foreground))]' }, [h('span', { key: 'provider', className: 'rounded-md bg-[hsl(var(--muted))] px-1.5 py-0.5 font-medium uppercase' }, copy.providers[item.provider] || item.provider), h('span', { key: 'model', className: 'truncate font-mono' }, item.primary_model), item.group_name ? h('span', { key: 'group', className: 'shrink-0 rounded-md bg-[hsl(var(--muted))] px-1.5 py-0.5' }, item.group_name) : null])]),
      h('span', { key: 'status', className: 'shrink-0 rounded-full border border-[hsl(var(--border))] bg-[hsl(var(--muted)/.65)] px-2.5 py-1 text-xs font-semibold text-[hsl(var(--foreground))]' }, status),
    ]),
    h('div', { key: 'metrics', className: 'mt-5 grid grid-cols-2 gap-2' }, [
      h('div', { key: 'latency', className: 'rounded-xl bg-[hsl(var(--muted)/.55)] p-3' }, [h('div', { key: 'label', className: 'flex items-center gap-1 text-xs text-[hsl(var(--muted-foreground))]' }, [h(Icon, { key: 'icon', name: 'bolt' }), copy.dialogLatency]), h('div', { key: 'value', className: 'mt-1 font-mono text-lg text-[hsl(var(--foreground))]' }, latency(item.primary_latency_ms)), h('span', { key: 'unit', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, 'ms')]),
      h('div', { key: 'ping', className: 'rounded-xl bg-[hsl(var(--muted)/.55)] p-3' }, [h('div', { key: 'label', className: 'flex items-center gap-1 text-xs text-[hsl(var(--muted-foreground))]' }, [h(Icon, { key: 'icon', name: 'globe' }), copy.endpointPing]), h('div', { key: 'value', className: 'mt-1 font-mono text-lg text-[hsl(var(--foreground))]' }, latency(item.primary_ping_latency_ms)), h('span', { key: 'unit', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, 'ms')]),
    ]),
    h('div', { key: 'availability', className: 'mt-4 flex items-center justify-between border-t border-[hsl(var(--border))] pt-4' }, [h('div', { key: 'label', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, `${copy.availability} · ${copy.windowTab[window]}`), h('span', { key: 'value', className: 'font-mono text-sm font-semibold text-[hsl(var(--foreground))]' }, percent(availability))]),
    h('div', { key: 'timeline', className: 'mt-4 flex h-4 items-end gap-0.5' }, (item.timeline || []).slice(-48).map((point, index) => h('span', { key: `${point.checked_at}-${index}`, className: `min-w-0 flex-1 rounded-sm ${point.status === 'operational' ? 'bg-[hsl(var(--foreground))]' : 'bg-[hsl(var(--muted-foreground)/.45)]'}`, style: { height: point.status === 'operational' ? '100%' : '65%' }, title: point.checked_at }))),
    h('div', { key: 'footer', className: 'mt-2 flex items-center justify-between text-[10px] text-[hsl(var(--muted-foreground))]' }, [item.extra_models?.length ? copy.extraModelsCount(item.extra_models.length) : '', h('span', { key: 'countdown', className: 'inline-flex items-center gap-1' }, [h(Icon, { key: 'icon', name: 'clock' }), copy.nextUpdateIn(countdown)])]),
  ])
}

function DetailDialog({ detail, title, loading, copy, onClose }: { detail: UserMonitorDetail | null; title: string; loading: boolean; copy: ChannelStatusCopy; onClose: () => void }): ReactNode {
  return h('div', { className: 'fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4', role: 'presentation', onMouseDown: (event: { target: unknown; currentTarget: unknown }) => { if (event.target === event.currentTarget) onClose() } }, h('section', { className: 'max-h-[calc(100vh-2rem)] w-full max-w-6xl overflow-hidden rounded-2xl border border-[hsl(var(--border))] bg-[hsl(var(--background))] shadow-2xl', role: 'dialog', 'aria-modal': 'true', 'aria-label': title }, [
    h('header', { key: 'header', className: 'flex items-center justify-between gap-4 border-b border-[hsl(var(--border))] px-6 py-4' }, [h('h2', { key: 'title', className: 'text-base font-semibold text-[hsl(var(--foreground))]' }, title), h('button', { key: 'close', type: 'button', className: 'rounded-xl px-2 py-1 text-xl text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))]', 'aria-label': copy.closeDetail, onClick: onClose }, '×')]),
    loading ? h('div', { key: 'loading', className: 'p-10 text-center text-sm text-[hsl(var(--muted-foreground))]', role: 'status' }, copy.loading) : detail ? h('div', { key: 'table', className: 'max-h-[65vh] overflow-auto p-6' }, h('table', { className: 'w-full min-w-[760px] border-collapse text-left text-sm' }, [
      h('thead', { key: 'head' }, h('tr', { className: 'border-b border-[hsl(var(--border))] text-xs uppercase tracking-wide text-[hsl(var(--muted-foreground))]' }, [copy.detailColumns.model, copy.detailColumns.latestStatus, copy.detailColumns.latestLatency, copy.detailColumns.availability7d, copy.detailColumns.availability15d, copy.detailColumns.availability30d, copy.detailColumns.avgLatency7d].map((label) => h('th', { key: label, className: 'px-3 py-2' }, label)))),
      h('tbody', { key: 'body' }, detail.models.map((model: UserMonitorModelDetail) => h('tr', { key: model.model, className: 'border-b border-[hsl(var(--border))]' }, [h('td', { key: 'model', className: 'px-3 py-3 font-medium text-[hsl(var(--foreground))]' }, model.model), h('td', { key: 'status', className: 'px-3 py-3' }, h('span', { className: 'rounded-full border border-[hsl(var(--border))] bg-[hsl(var(--muted))] px-2 py-1 text-xs' }, copy.status[model.latest_status] || model.latest_status)), h('td', { key: 'latency', className: 'px-3 py-3 font-mono' }, latency(model.latest_latency_ms)), h('td', { key: '7d', className: 'px-3 py-3 font-mono' }, percent(model.availability_7d)), h('td', { key: '15d', className: 'px-3 py-3 font-mono' }, percent(model.availability_15d)), h('td', { key: '30d', className: 'px-3 py-3 font-mono' }, percent(model.availability_30d)), h('td', { key: 'avg', className: 'px-3 py-3 font-mono' }, latency(model.avg_latency_7d_ms))]))),
    ])) : h('div', { key: 'error', className: 'p-10 text-center text-sm text-[hsl(var(--muted-foreground))]' }, copy.detailLoadError),
    h('footer', { key: 'footer', className: 'flex justify-end border-t border-[hsl(var(--border))] px-6 py-4' }, h('button', { type: 'button', className: 'btn btn-secondary', onClick: onClose }, copy.closeDetail)),
  ]))
}

export default function ChannelStatusPage({ items, loading, overallStatus, currentWindow, countdown, detailCache, autoRefresh, copy, onWindowChange, onRefresh, onAutoRefreshChange, onIntervalChange, onCardClick, onLoadDetail }: ChannelStatusPageProps) {
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const selected = selectedId == null ? null : items.find((item) => item.id === selectedId) || null
  const detail = selectedId == null ? null : detailCache[selectedId] || null
  const openDetail = (item: UserMonitorView) => {
    setSelectedId(item.id)
    setDetailLoading(true)
    onCardClick(item)
    Promise.resolve(onLoadDetail(item.id)).finally(() => setDetailLoading(false))
  }
  return h('div', { className: 'space-y-3' }, [
    h('section', { key: 'hero', className: 'flex flex-wrap items-center justify-end gap-3 py-3' }, [
      h('div', { key: 'tabs', className: 'inline-flex rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--muted)/.55)] p-0.5' }, (['7d', '15d', '30d'] as const).map((value) => h('button', { key: value, type: 'button', role: 'tab', 'aria-selected': currentWindow === value, className: `rounded-lg px-3 py-1.5 text-xs ${currentWindow === value ? 'bg-[hsl(var(--background))] font-semibold text-[hsl(var(--foreground))] shadow-sm' : 'text-[hsl(var(--muted-foreground))]'}`, onClick: () => onWindowChange(value) }, copy.windowTab[value]))),
      h('span', { key: 'status', className: 'rounded-full border border-[hsl(var(--border))] bg-[hsl(var(--muted))] px-2.5 py-1 text-xs font-semibold uppercase tracking-wide text-[hsl(var(--foreground))]' }, copy.overall[overallStatus]),
      h('button', { key: 'refresh', type: 'button', className: 'btn btn-secondary h-9', disabled: loading, title: copy.refresh, onClick: onRefresh }, h(Icon, { name: 'refresh' })),
      h('label', { key: 'auto', className: 'inline-flex items-center gap-2 text-xs text-[hsl(var(--muted-foreground))]' }, [h('input', { type: 'checkbox', checked: autoRefresh.enabled, onChange: (event: { target: { checked: boolean } }) => onAutoRefreshChange(event.target.checked) }), `${copy.pollEvery(autoRefresh.intervalSeconds)} · ${countdown}s`]),
      h('select', { key: 'interval', value: autoRefresh.intervalSeconds, className: 'input h-9 w-20', disabled: !autoRefresh.enabled, 'aria-label': 'Polling interval', onChange: (event: { target: { value: string } }) => onIntervalChange(Number(event.target.value)) }, autoRefresh.intervals.map((seconds) => h('option', { key: seconds, value: seconds }, `${seconds}s`))),
    ]),
    loading && !items.length ? h('div', { key: 'loading', className: 'grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4' }, Array.from({ length: 6 }, (_, index) => h('div', { key: index, className: 'card min-h-[250px] animate-pulse p-5' }))) : !items.length ? h('section', { key: 'empty', className: 'card flex min-h-64 flex-col items-center justify-center gap-2 p-8 text-center' }, [h('p', { key: 'title', className: 'text-sm font-medium text-[hsl(var(--foreground))]' }, copy.emptyTitle), h('p', { key: 'description', className: 'text-sm text-[hsl(var(--muted-foreground))]' }, copy.emptyDescription)]) : h('div', { key: 'cards', className: 'grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4' }, items.map((item) => h(MonitorCard, { key: item.id, item, window: currentWindow, countdown, copy, onClick: () => openDetail(item) }))),
    selected ? h(DetailDialog, { key: 'dialog', title: selected.name || copy.detailTitle, detail, loading: detailLoading, copy, onClose: () => setSelectedId(null) }) : null,
  ])
}
