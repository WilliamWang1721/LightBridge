import { useMemo, useState, type ReactNode } from 'react'
import type {
  ModelCatalogGroup,
  ModelCatalogModel,
  ModelCatalogPriceRange,
} from '@/api/modelCatalog'
import { createShadcnElement as h } from './ui/createElement'

export type CatalogViewMode = 'merged' | 'by_group' | 'by_channel' | 'by_account'

export interface ModelCatalogPageCopy {
  refresh: string
  searchPlaceholder: string
  allGroups: string
  modelCount: (count: number) => string
  sourceCount: (count: number) => string
  sourceDetails: string
  unknownAccount: string
  noGroups: string
  noPrice: string
  priceTokenRange: (values: { input: string; output: string }) => string
  priceRequestRange: (price: string) => string
  usageUnknown: string
  views: Record<CatalogViewMode, string>
  usageMode: (mode: string) => string
  setupMonitor: string
  monitorStatus: (status: string) => string
  empty: string
}

export interface ModelCatalogPageProps {
  models: ModelCatalogModel[]
  loading: boolean
  admin: boolean
  copy: ModelCatalogPageCopy
  onRefresh: () => void
  onQuickSetup?: (model: ModelCatalogModel) => void
}

interface CatalogSection {
  key: string
  title: string
  models: ModelCatalogModel[]
}

function Icon({ name }: { name: 'search' | 'grid' | 'chevronDown' | 'refresh' | 'plus' | 'dollar' }): ReactNode {
  const paths = {
    search: 'm21 21-4.35-4.35m1.35-5.4a6.75 6.75 0 1 1-13.5 0 6.75 6.75 0 0 1 13.5 0Z',
    grid: 'M4.5 4.5h6v6h-6v-6Zm9 0h6v6h-6v-6Zm-9 9h6v6h-6v-6Zm9 0h6v6h-6v-6Z',
    chevronDown: 'm6.75 9 5.25 5.25L17.25 9',
    refresh: 'M20.25 12a8.25 8.25 0 1 1-2.42-5.83M20.25 4.5v5.25H15',
    plus: 'M12 5.25v13.5m6.75-6.75H5.25',
    dollar: 'M12 6.75v10.5m3-8.25c-.62-.77-1.56-1.25-2.63-1.25h-.74A2.63 2.63 0 0 0 9 10.38c0 1.45 1.18 2.62 2.63 2.62h.74A2.63 2.63 0 0 1 15 15.62 2.63 2.63 0 0 1 12.37 18h-.74A3.38 3.38 0 0 1 9 16.75',
  }[name]
  return h('svg', {
    className: 'h-4 w-4 shrink-0',
    fill: 'none',
    viewBox: '0 0 24 24',
    stroke: 'currentColor',
    strokeWidth: 1.6,
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
    'aria-hidden': 'true',
  }, h('path', { d: paths }))
}

function ModelGlyph({ model }: { model: string }): ReactNode {
  const glyph = model.replace(/[^a-z0-9]/gi, '').slice(0, 1).toUpperCase() || '•'
  return h('span', {
    className: 'flex h-5 w-5 shrink-0 items-center justify-center rounded-md bg-[hsl(var(--muted))] text-[10px] font-semibold text-[hsl(var(--muted-foreground))]',
    'aria-hidden': 'true',
  }, glyph)
}

function formatMoney(value?: number | null): string {
  return value == null ? '-' : `$${Number(value).toFixed(4)}`
}

function formatMinMax(min?: number | null, max?: number | null): string {
  if (min == null && max == null) return ''
  if (min == null) return formatMoney(max)
  if (max == null || min === max) return formatMoney(min)
  return `${formatMoney(min)} - ${formatMoney(max)}`
}

function formatPriceRange(range: ModelCatalogPriceRange | null | undefined, copy: ModelCatalogPageCopy): string {
  if (!range) return copy.noPrice
  const input = formatMinMax(range.min_input_price, range.max_input_price)
  const output = formatMinMax(range.min_output_price, range.max_output_price)
  const request = formatMinMax(range.min_per_request_price, range.max_per_request_price)
  if (input || output) return copy.priceTokenRange({ input: input || '-', output: output || '-' })
  if (request) return copy.priceRequestRange(request)
  return copy.noPrice
}

function monitorClass(status: string): string {
  if (status === 'failed' || status === 'error') {
    return 'border-[hsl(var(--destructive)/.25)] bg-[hsl(var(--destructive)/.1)] text-[hsl(var(--destructive))]'
  }
  return 'border-[hsl(var(--border))] bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]'
}

function sourceChannels(source: NonNullable<ModelCatalogModel['sources']>[number]): string {
  const channels = (source.channels || []).map((channel) => channel.name).filter(Boolean)
  return channels.length > 0 ? channels.join(', ') : source.platform
}

function sectionsFor(
  models: ModelCatalogModel[],
  mode: CatalogViewMode,
  copy: ModelCatalogPageCopy,
): CatalogSection[] {
  if (mode === 'merged') return [{ key: 'merged', title: copy.views.merged, models }]

  const sections = new Map<string, CatalogSection>()
  const add = (key: string, title: string, model: ModelCatalogModel) => {
    const section = sections.get(key) || { key, title, models: [] }
    section.models.push(model)
    sections.set(key, section)
  }

  for (const model of models) {
    if (mode === 'by_group') {
      for (const group of model.groups || []) add(String(group.id), group.name, model)
    } else if (mode === 'by_channel') {
      const channels = new Map<string, string>()
      for (const source of model.sources || []) {
        for (const channel of source.channels || []) {
          const key = String(channel.id || channel.name)
          if (key) channels.set(key, channel.name || key)
        }
      }
      if (channels.size === 0) {
        for (const source of model.sources || []) {
          if (source.platform) channels.set(source.platform, source.platform)
        }
      }
      channels.forEach((title, key) => add(key, title, model))
    } else {
      for (const source of model.sources || []) {
        const key = String(source.account_id || source.account_name || source.platform)
        add(key, source.account_name || copy.unknownAccount, model)
      }
    }
  }

  return Array.from(sections.values()).sort((left, right) => left.title.localeCompare(right.title))
}

function ModelCard({
  model,
  admin,
  copy,
  onQuickSetup,
}: {
  model: ModelCatalogModel
  admin: boolean
  copy: ModelCatalogPageCopy
  onQuickSetup?: (model: ModelCatalogModel) => void
}): ReactNode {
  const groups = model.groups || []
  return h('article', { className: 'rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--background))] p-4 shadow-sm' }, [
    h('div', { key: 'heading', className: 'mb-3 flex items-start justify-between gap-3' }, [
      h('div', { key: 'name', className: 'min-w-0' }, [
        h('div', { key: 'display', className: 'flex items-center gap-2' }, [
          h(ModelGlyph, { key: 'glyph', model: model.id }),
          h('h3', { key: 'title', className: 'truncate text-sm font-semibold text-[hsl(var(--foreground))]' }, model.display_name || model.id),
        ]),
        h('p', { key: 'id', className: 'mt-1 truncate font-mono text-xs text-[hsl(var(--muted-foreground))]' }, model.id),
      ]),
      h('span', { key: 'sources', className: 'shrink-0 rounded-full border border-[hsl(var(--border))] bg-[hsl(var(--muted))] px-2 py-1 text-xs text-[hsl(var(--muted-foreground))]' }, copy.sourceCount(model.source_count)),
    ]),
    h('div', { key: 'modes', className: 'mb-3 flex flex-wrap gap-1.5' }, (model.usage_modes || []).length
      ? model.usage_modes.map((mode) => h('span', { key: mode, className: 'rounded-md bg-[hsl(var(--muted))] px-2 py-0.5 text-xs text-[hsl(var(--muted-foreground))]' }, copy.usageMode(mode)))
      : h('span', { className: 'rounded-md bg-[hsl(var(--muted))] px-2 py-0.5 text-xs text-[hsl(var(--muted-foreground))]' }, copy.usageUnknown)),
    model.monitor_status
      ? h('div', { key: 'monitor', className: 'mb-2 flex items-center gap-2' }, [
        h('span', { key: 'status', className: `inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium ${monitorClass(model.monitor_status)}` }, [
          h('span', { key: 'dot', className: 'h-1.5 w-1.5 rounded-full bg-current' }),
          copy.monitorStatus(model.monitor_status),
        ]),
        model.monitor_availability_7d != null
          ? h('span', { key: 'availability', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, `${model.monitor_availability_7d.toFixed(1)}%${model.monitor_latency_ms != null ? ` · ${model.monitor_latency_ms}ms` : ''}`)
          : null,
      ])
      : admin && onQuickSetup
        ? h('button', { key: 'setup', type: 'button', className: 'mb-2 inline-flex items-center gap-1 text-xs text-[hsl(var(--muted-foreground))] transition-colors hover:text-[hsl(var(--foreground))]', onClick: () => onQuickSetup(model) }, [h(Icon, { key: 'plus', name: 'plus' }), copy.setupMonitor])
        : null,
    h('div', { key: 'meta', className: 'space-y-2 text-xs text-[hsl(var(--muted-foreground))]' }, [
      h('div', { key: 'price', className: 'flex items-start gap-2' }, [h(Icon, { key: 'icon', name: 'dollar' }), h('span', { key: 'value' }, formatPriceRange(model.price_range, copy))]),
      h('div', { key: 'groups', className: 'flex items-start gap-2' }, [h(Icon, { key: 'icon', name: 'grid' }), h('span', { key: 'value', className: 'line-clamp-2' }, groups.length ? groups.map((group) => group.name).join(', ') : copy.noGroups)]),
    ]),
    admin && model.sources?.length
      ? h('div', { key: 'source-details', className: 'mt-4 border-t border-[hsl(var(--border))] pt-3' }, [
        h('div', { key: 'label', className: 'mb-2 text-xs font-medium text-[hsl(var(--muted-foreground))]' }, copy.sourceDetails),
        h('div', { key: 'rows', className: 'space-y-1.5' }, model.sources.map((source, index) => h('div', { key: `${source.account_id || source.account_name || source.platform}:${source.platform}:${index}`, className: 'flex items-center justify-between gap-2 rounded-lg bg-[hsl(var(--muted)/.55)] px-2 py-1.5 text-xs' }, [
          h('span', { key: 'account', className: 'min-w-0 truncate text-[hsl(var(--foreground))]' }, source.account_name || copy.unknownAccount),
          h('span', { key: 'channels', className: 'shrink-0 text-[hsl(var(--muted-foreground))]' }, sourceChannels(source)),
        ]))),
      ])
      : null,
  ])
}

export default function ModelCatalogPage({ models, loading, admin, copy, onRefresh, onQuickSetup }: ModelCatalogPageProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [activeView, setActiveView] = useState<CatalogViewMode>('merged')
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null)
  const [showGroupDropdown, setShowGroupDropdown] = useState(false)
  const visibleViewModes = admin ? (['merged', 'by_group', 'by_channel', 'by_account'] as CatalogViewMode[]) : (['merged', 'by_group'] as CatalogViewMode[])

  const allGroups = useMemo(() => {
    const groupMap = new Map<number, ModelCatalogGroup>()
    models.forEach((model) => (model.groups || []).forEach((group) => groupMap.set(group.id, group)))
    return Array.from(groupMap.values()).sort((left, right) => left.name.localeCompare(right.name))
  }, [models])
  const selectedGroupName = allGroups.find((group) => group.id === selectedGroupId)?.name || ''
  const filteredModels = useMemo(() => {
    const selected = selectedGroupId ? models.filter((model) => model.groups?.some((group) => group.id === selectedGroupId)) : models
    const query = searchQuery.trim().toLowerCase()
    if (!query) return selected
    return selected.filter((model) => {
      const groupHit = (model.groups || []).some((group) => group.name.toLowerCase().includes(query))
      const sourceHit = admin && model.sources?.some((source) => [source.account_name, source.platform, source.source].some((value) => String(value || '').toLowerCase().includes(query)))
      return model.id.toLowerCase().includes(query) || (model.display_name || '').toLowerCase().includes(query) || model.platform.toLowerCase().includes(query) || groupHit || Boolean(sourceHit)
    })
  }, [admin, models, searchQuery, selectedGroupId])
  const sections = useMemo(() => sectionsFor(filteredModels, activeView, copy), [activeView, copy, filteredModels])

  return h('div', { className: 'mx-auto w-full max-w-7xl space-y-6' }, [
    h('div', { key: 'toolbar', className: 'flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between' }, [
      h('div', { key: 'filters', className: 'flex flex-1 flex-wrap items-center gap-3' }, [
        h('label', { key: 'search', className: 'relative w-full sm:w-80' }, [
          h('span', { key: 'label', className: 'sr-only' }, copy.searchPlaceholder),
          h(Icon, { key: 'icon', name: 'search' }),
          h('input', { key: 'input', type: 'search', value: searchQuery, onChange: (event: { target: { value: string } }) => setSearchQuery(event.target.value), placeholder: copy.searchPlaceholder, className: 'input pl-10' }),
        ]),
        activeView === 'by_group' && allGroups.length > 0
          ? h('div', { key: 'group-filter', className: 'relative' }, [
            h('button', { key: 'trigger', type: 'button', className: 'inline-flex items-center gap-2 rounded-lg border border-[hsl(var(--border))] bg-[hsl(var(--background))] px-3 py-2 text-sm text-[hsl(var(--foreground))] transition-colors hover:bg-[hsl(var(--muted))]', onClick: () => setShowGroupDropdown((value) => !value), 'aria-expanded': showGroupDropdown }, [h(Icon, { key: 'grid', name: 'grid' }), h('span', { key: 'name', className: 'max-w-[120px] truncate' }, selectedGroupName || copy.allGroups), h('span', { key: 'chevron', className: showGroupDropdown ? 'rotate-180 transition-transform' : 'transition-transform' }, h(Icon, { name: 'chevronDown' }))]),
            showGroupDropdown
              ? h('div', { key: 'menu', className: 'absolute left-0 z-50 mt-2 max-h-64 w-56 overflow-y-auto rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--background))] p-1 shadow-xl' }, [
                h('button', { key: 'all', type: 'button', className: `flex w-full items-center px-3 py-2 text-left text-sm transition-colors hover:bg-[hsl(var(--muted))] ${selectedGroupId === null ? 'bg-[hsl(var(--muted))]' : ''}`, onClick: () => { setSelectedGroupId(null); setShowGroupDropdown(false) } }, copy.allGroups),
                ...allGroups.map((group) => h('button', { key: group.id, type: 'button', className: `flex w-full items-center justify-between px-3 py-2 text-left text-sm transition-colors hover:bg-[hsl(var(--muted))] ${selectedGroupId === group.id ? 'bg-[hsl(var(--muted))]' : ''}`, onClick: () => { setSelectedGroupId(group.id); setShowGroupDropdown(false) } }, [h('span', { key: 'name', className: 'truncate' }, group.name), h('span', { key: 'count', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, models.filter((model) => model.groups?.some((item) => item.id === group.id)).length)])),
              ])
              : null,
          ])
          : null,
        h('div', { key: 'views', className: 'inline-flex overflow-hidden rounded-lg border border-[hsl(var(--border))] bg-[hsl(var(--background))]' }, visibleViewModes.map((mode, index) => h('button', { key: mode, type: 'button', className: `px-3 py-2 text-sm transition-colors ${index === 0 ? 'rounded-l-lg' : ''} ${index === visibleViewModes.length - 1 ? 'rounded-r-lg' : ''} ${activeView === mode ? 'bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]' : 'text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))]'}`, onClick: () => setActiveView(mode) }, copy.views[mode]))),
      ]),
      h('button', { key: 'refresh', type: 'button', className: 'btn btn-secondary', disabled: loading, title: copy.refresh, onClick: onRefresh }, h(Icon, { name: 'refresh' })),
    ]),
    loading
      ? h('div', { key: 'loading', className: 'grid gap-3 md:grid-cols-2 xl:grid-cols-3' }, Array.from({ length: 6 }, (_, index) => h('div', { key: index, className: 'h-44 animate-pulse rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--background))]' })))
      : sections.length === 0
        ? h('div', { key: 'empty', className: 'rounded-xl border border-dashed border-[hsl(var(--border))] bg-[hsl(var(--background))] px-6 py-12 text-center' }, [h('div', { key: 'icon', className: 'mx-auto mb-3 flex h-10 w-10 items-center justify-center rounded-xl bg-[hsl(var(--muted))] text-sm font-semibold text-[hsl(var(--muted-foreground))]' }, '∅'), h('p', { key: 'text', className: 'text-sm text-[hsl(var(--muted-foreground))]' }, copy.empty)])
        : h('div', { key: 'sections', className: 'space-y-6' }, sections.map((section) => h('section', { key: section.key, className: 'space-y-3' }, [
          activeView !== 'merged' ? h('div', { key: 'heading', className: 'flex items-center gap-2' }, [h('h2', { key: 'title', className: 'text-sm font-semibold text-[hsl(var(--foreground))]' }, section.title), h('span', { key: 'count', className: 'rounded-full bg-[hsl(var(--muted))] px-2 py-0.5 text-xs text-[hsl(var(--muted-foreground))]' }, copy.modelCount(section.models.length))]) : null,
          h('div', { key: 'grid', className: 'grid gap-3 md:grid-cols-2 xl:grid-cols-3' }, section.models.map((model, index) => h(ModelCard, { key: `${section.key}:${model.id}:${index}`, model, admin, copy, onQuickSetup }))),
        ]))),
  ])
}
