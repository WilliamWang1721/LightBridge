import { type ReactNode } from 'react'
import type { UserAvailableChannel, UserAvailableGroup, UserSupportedModel } from '@/api/channels'
import { createShadcnElement as h } from './ui/createElement'
import { AppIcon } from './ui/app-icon'

export interface AvailableChannelsPageCopy {
  searchPlaceholder: string
  refresh: string
  name: string
  description: string
  platform: string
  groups: string
  supportedModels: string
  empty: string
  noPricing: string
  noModels: string
  exclusive: string
  public: string
}

export interface AvailableChannelsPageProps {
  rows: UserAvailableChannel[]
  searchQuery: string
  loading: boolean
  userGroupRates: Record<number, number>
  copy: AvailableChannelsPageCopy
  onSearch: (value: string) => void
  onRefresh: () => void
}

function Icon({ name }: { name: 'search' | 'refresh' | 'shield' | 'globe' | 'inbox' }): ReactNode {
  return h(AppIcon, { name })
}

function channelGroups(channel: UserAvailableChannel): UserAvailableGroup[] {
  const groups = new Map<number, UserAvailableGroup>()
  channel.platforms.forEach((section) => section.groups.forEach((group) => groups.set(group.id, group)))
  return [...groups.values()]
}

function GroupList({ channel, rates, copy }: { channel: UserAvailableChannel; rates: Record<number, number>; copy: AvailableChannelsPageCopy }): ReactNode {
  const groups = channelGroups(channel)
  if (!groups.length) return h('span', { className: 'text-xs text-[hsl(var(--muted-foreground))]' }, '-')
  const renderGroup = (group: UserAvailableGroup) => {
    const customRate = rates[group.id]
    const rateChanged = customRate != null && customRate !== group.rate_multiplier
    return h('span', { key: group.id, className: 'inline-flex max-w-full items-center gap-1 rounded-lg border border-[hsl(var(--border))] bg-[hsl(var(--muted)/.55)] px-2 py-1 text-xs text-[hsl(var(--foreground))]', title: group.available_ingress_protocols?.join(', ') || undefined }, [
      h('span', { key: 'name', className: 'truncate' }, group.name),
      h('span', { key: 'rate', className: 'shrink-0 font-mono text-[10px] text-[hsl(var(--muted-foreground))]' }, rateChanged ? `${group.rate_multiplier}x → ${customRate}x` : `${group.rate_multiplier}x`),
    ])
  }
  return h('div', { className: 'space-y-2' }, [
    groups.filter((group) => group.is_exclusive).length ? h('div', { key: 'exclusive', className: 'flex flex-wrap items-center gap-1.5' }, [h('span', { key: 'label', className: 'inline-flex items-center gap-1 text-[10px] font-medium uppercase text-[hsl(var(--muted-foreground))]' }, [h(Icon, { key: 'icon', name: 'shield' }), copy.exclusive]), ...groups.filter((group) => group.is_exclusive).map(renderGroup)]) : null,
    groups.filter((group) => !group.is_exclusive).length ? h('div', { key: 'public', className: 'flex flex-wrap items-center gap-1.5' }, [h('span', { key: 'label', className: 'inline-flex items-center gap-1 text-[10px] font-medium uppercase text-[hsl(var(--muted-foreground))]' }, [h(Icon, { key: 'icon', name: 'globe' }), copy.public]), ...groups.filter((group) => !group.is_exclusive).map(renderGroup)]) : null,
  ])
}

function pricingLabel(model: UserSupportedModel, copy: AvailableChannelsPageCopy): string {
  if (!model.pricing) return copy.noPricing
  if (model.pricing.billing_mode === 'per_request') return model.pricing.per_request_price == null ? 'per request' : `$${model.pricing.per_request_price}/request`
  if (model.pricing.billing_mode === 'image') return model.pricing.image_output_price == null ? 'image' : `$${model.pricing.image_output_price}/image`
  if (model.pricing.input_price == null && model.pricing.output_price == null) return 'token'
  return `$${model.pricing.input_price ?? 0}/$${model.pricing.output_price ?? 0} / 1M`
}

function Models({ models, copy, platform }: { models: UserSupportedModel[]; copy: AvailableChannelsPageCopy; platform: string }): ReactNode {
  if (!models.length) return h('span', { className: 'text-xs text-[hsl(var(--muted-foreground))]' }, copy.noModels)
  return h('div', { className: 'flex flex-wrap gap-1.5' }, models.map((model) => h('span', { key: `${platform}-${model.name}`, className: 'inline-flex max-w-full items-center gap-1 rounded-lg border border-[hsl(var(--border))] px-2 py-1 text-xs text-[hsl(var(--foreground))]', title: `${model.platform || platform} · ${pricingLabel(model, copy)}` }, [h('span', { key: 'name', className: 'truncate' }, model.name), h('span', { key: 'price', className: 'shrink-0 text-[10px] text-[hsl(var(--muted-foreground))]' }, pricingLabel(model, copy))])))
}

function Table({ rows, rates, copy }: { rows: UserAvailableChannel[]; rates: Record<number, number>; copy: AvailableChannelsPageCopy }): ReactNode {
  if (!rows.length) return h('div', { className: 'card flex min-h-64 flex-col items-center justify-center gap-3 p-8 text-center' }, [h(Icon, { key: 'icon', name: 'inbox' }), h('p', { key: 'text', className: 'text-sm text-[hsl(var(--muted-foreground))]' }, copy.empty)])
  return h('div', { className: 'card overflow-hidden' }, h('div', { className: 'overflow-x-auto' }, h('table', { className: 'w-full min-w-[920px] border-collapse text-sm' }, [
    h('thead', { key: 'head' }, h('tr', { className: 'border-b border-[hsl(var(--border))] bg-[hsl(var(--muted)/.45)] text-left text-xs font-medium uppercase tracking-wide text-[hsl(var(--muted-foreground))]' }, [copy.name, copy.description, copy.platform, copy.groups, copy.supportedModels].map((label) => h('th', { key: label, className: 'px-4 py-3' }, label)))),
    h('tbody', { key: 'body' }, rows.flatMap((channel, channelIndex) => channel.platforms.map((section, sectionIndex) => h('tr', { key: `${channel.name}-${section.platform}-${channelIndex}`, className: 'border-b border-[hsl(var(--border))] align-top last:border-0 hover:bg-[hsl(var(--muted)/.25)]' }, [
      sectionIndex === 0 ? h('td', { key: 'name', rowSpan: channel.platforms.length, className: 'w-44 px-4 py-4 align-middle font-medium text-[hsl(var(--foreground))]' }, channel.name) : null,
      sectionIndex === 0 ? h('td', { key: 'description', rowSpan: channel.platforms.length, className: 'w-52 px-4 py-4 align-middle text-xs text-[hsl(var(--muted-foreground))]' }, channel.description || '-') : null,
      h('td', { key: 'platform', className: 'px-4 py-4' }, h('span', { className: 'inline-flex items-center rounded-lg border border-[hsl(var(--border))] bg-[hsl(var(--muted)/.55)] px-2 py-1 text-[11px] font-medium uppercase text-[hsl(var(--foreground))]' }, section.platform)),
      sectionIndex === 0 ? h('td', { key: 'groups', rowSpan: channel.platforms.length, className: 'px-4 py-4' }, h(GroupList, { channel, rates, copy })) : null,
      h('td', { key: 'models', className: 'px-4 py-4' }, h(Models, { models: section.supported_models, copy, platform: section.platform })),
    ])))),
  ])))
}

export default function AvailableChannelsPage({ rows, searchQuery, loading, userGroupRates, copy, onSearch, onRefresh }: AvailableChannelsPageProps) {
  return h('div', { className: 'table-page-layout flex min-h-0 flex-col gap-5' }, [
    h('div', { key: 'filters', className: 'flex flex-col justify-between gap-4 lg:flex-row lg:items-start' }, [
      h('label', { key: 'search', className: 'relative block w-full sm:w-80' }, [h('span', { className: 'sr-only' }, copy.searchPlaceholder), h('span', { className: 'pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[hsl(var(--muted-foreground))]' }, h(Icon, { name: 'search' })), h('input', { value: searchQuery, type: 'search', placeholder: copy.searchPlaceholder, className: 'input h-10 pl-10', onChange: (event: { target: { value: string } }) => onSearch(event.target.value) })]),
      h('button', { key: 'refresh', type: 'button', className: 'btn btn-secondary h-10 self-end', disabled: loading, title: copy.refresh, onClick: onRefresh }, [h(Icon, { key: 'icon', name: 'refresh' }), h('span', { key: 'label', className: 'sr-only' }, copy.refresh)]),
    ]),
    loading ? h('div', { key: 'loading', className: 'card flex min-h-64 items-center justify-center' }, h(Icon, { name: 'refresh' })) : h(Table, { key: 'table', rows, rates: userGroupRates, copy }),
  ])
}
