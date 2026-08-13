import { type ReactNode } from 'react'
import type { FeatureControlState, FeatureRuntimeComponent } from '@/api/admin/features'
import { createShadcnElement as h } from './ui/createElement'
import { AppIcon } from './ui/app-icon'
import { Switch as ShadcnSwitch } from './ui'

export type FeatureGroupID = 'core' | 'optional' | 'extension' | 'other'

export interface FeatureRegistryFeatureGroup {
  id: FeatureGroupID
  features: readonly FeatureControlState[]
}

export interface FeatureRegistryPageCopy {
  refresh: string
  profile: (profile: string) => string
  featureCount: (count: number) => string
  enabled: string
  disabled: string
  restartRequired: string
  configurable: string
  notConfigurable: string
  unavailable: string
  overrideEnabled: string
  overrideDisabled: string
  minimumProfile: (profile: string) => string
  configuredState: (state: string) => string
  runtimeState: (state: string) => string
  reason: (reason: string) => string
  dependencies: string
  noDependencies: string
  surfaces: string
  runtimeComponents: string
  noRuntimeComponents: string
  restoreDefault: string
  noFeatures: string
  noFeaturesHint: string
  tierLabel: (tier: FeatureGroupID) => string
  activationLabel: (activation: string) => string
  reasonLabel: (reason: string) => string
}

export interface FeatureRegistryPageProps {
  loading: boolean
  error: string
  profile?: string
  groups: readonly FeatureRegistryFeatureGroup[]
  busyFeatureID: string
  copy: FeatureRegistryPageCopy
  onRefresh: () => void
  onSetEnabled: (feature: FeatureControlState, enabled: boolean) => void
  onRestoreDefault: (feature: FeatureControlState) => void
}

function Icon({ name, className = 'h-5 w-5' }: { name: 'refresh' | 'error'; className?: string }): ReactNode {
  return h(AppIcon, { name, className })
}

function Switch({
  checked,
  disabled,
  label,
  onChange,
}: {
  checked: boolean
  disabled: boolean
  label: string
  onChange: () => void
}): ReactNode {
  return h(ShadcnSwitch, { checked, disabled, 'aria-label': label, onCheckedChange: () => onChange() })
}

function runtimeComponentLabel(component: FeatureRuntimeComponent): string {
  const status = component.status || (component.running ? 'running' : component.started ? 'started' : '')
  return [component.name || component.feature || JSON.stringify(component), status].filter(Boolean).join(' · ')
}

function runtimeComponentKey(component: FeatureRuntimeComponent): string {
  return `${component.name || ''}:${component.feature || ''}:${component.status || ''}:${component.updatedAt || ''}`
}

function FeatureCard({
  feature,
  copy,
  busy,
  onSetEnabled,
  onRestoreDefault,
}: {
  feature: FeatureControlState
  copy: FeatureRegistryPageCopy
  busy: boolean
  onSetEnabled: (feature: FeatureControlState, enabled: boolean) => void
  onRestoreDefault: (feature: FeatureControlState) => void
}): ReactNode {
  const enabledText = feature.enabled ? copy.enabled : copy.disabled
  return h('article', { className: 'card p-5', 'data-test': `feature-${feature.id}` }, [
    h('div', { key: 'header', className: 'flex items-start justify-between gap-4' }, [
      h('div', { key: 'identity', className: 'min-w-0' }, [
        h('div', { key: 'title', className: 'flex flex-wrap items-center gap-2' }, [
          h('h3', { key: 'label', className: 'truncate font-semibold text-[hsl(var(--foreground))]' }, feature.label || feature.id),
          h('span', { key: 'status', className: 'rounded-full bg-[hsl(var(--muted))] px-2 py-0.5 text-xs font-medium text-[hsl(var(--foreground))]' }, enabledText),
          feature.requiresRestart
            ? h('span', { key: 'restart', className: 'rounded-full bg-[hsl(var(--muted))] px-2 py-0.5 text-xs font-medium text-[hsl(var(--muted-foreground))]' }, copy.restartRequired)
            : null,
        ]),
        h('p', { key: 'id', className: 'mt-1 font-mono text-xs text-[hsl(var(--muted-foreground))]' }, feature.id),
      ]),
      h(Switch, {
        key: 'switch',
        checked: feature.configuredEnabled,
        disabled: busy || feature.controllable !== true,
        label: feature.label || feature.id,
        onChange: () => onSetEnabled(feature, !feature.configuredEnabled),
      }),
    ]),
    h('div', { key: 'badges', className: 'mt-4 flex flex-wrap gap-2 text-xs' }, [
      h('span', { key: 'activation', className: 'rounded bg-[hsl(var(--muted))] px-2 py-1 text-[hsl(var(--muted-foreground))]' }, copy.activationLabel(feature.activation)),
      h('span', { key: 'profile', className: 'rounded bg-[hsl(var(--muted))] px-2 py-1 text-[hsl(var(--muted-foreground))]' }, copy.minimumProfile(feature.minimumProfile || '—')),
      h('span', { key: 'controllable', className: 'rounded bg-[hsl(var(--muted))] px-2 py-1 text-[hsl(var(--muted-foreground))]' }, feature.controllable === true ? copy.configurable : copy.notConfigurable),
      feature.available === false ? h('span', { key: 'available', className: 'rounded bg-[hsl(var(--muted))] px-2 py-1 text-[hsl(var(--muted-foreground))]' }, copy.unavailable) : null,
      feature.override !== null && feature.override !== undefined
        ? h('span', { key: 'override', className: 'rounded bg-[hsl(var(--muted))] px-2 py-1 text-[hsl(var(--muted-foreground))]' }, feature.override ? copy.overrideEnabled : copy.overrideDisabled)
        : null,
    ]),
    h('dl', { key: 'details', className: 'mt-4 grid gap-3 border-t border-[hsl(var(--border))] pt-4 text-sm' }, [
      h('div', { key: 'states', className: 'flex flex-wrap gap-x-2 gap-y-1 text-[hsl(var(--muted-foreground))]' }, [
        copy.configuredState(feature.configuredEnabled ? copy.enabled : copy.disabled),
        h('span', { key: 'runtime', className: 'text-[hsl(var(--foreground))]' }, `· ${copy.runtimeState(enabledText)}`),
      ]),
      feature.reason ? h('div', { key: 'reason', className: 'text-[hsl(var(--muted-foreground))]' }, copy.reason(copy.reasonLabel(feature.reason))) : null,
      h('div', { key: 'dependencies' }, [
        h('dt', { key: 'label', className: 'mb-1 text-xs font-medium uppercase tracking-wide text-[hsl(var(--muted-foreground))]' }, copy.dependencies),
        h('dd', { key: 'values', className: 'flex flex-wrap gap-1.5' }, feature.dependencies?.length
          ? feature.dependencies.map((dependency) => h('span', { key: dependency, className: 'rounded bg-[hsl(var(--muted))] px-2 py-1 font-mono text-xs text-[hsl(var(--foreground))]' }, dependency))
          : h('span', { className: 'text-sm text-[hsl(var(--muted-foreground))]' }, copy.noDependencies)),
      ]),
      feature.surfaces?.length ? h('div', { key: 'surfaces' }, [
        h('dt', { key: 'label', className: 'mb-1 text-xs font-medium uppercase tracking-wide text-[hsl(var(--muted-foreground))]' }, copy.surfaces),
        h('dd', { key: 'values', className: 'flex flex-wrap gap-1.5' }, feature.surfaces.map((surface) => h('span', { key: surface, className: 'rounded bg-[hsl(var(--muted))] px-2 py-1 font-mono text-xs text-[hsl(var(--foreground))]' }, surface))),
      ]) : null,
      h('div', { key: 'runtime-components' }, [
        h('dt', { key: 'label', className: 'mb-1 text-xs font-medium uppercase tracking-wide text-[hsl(var(--muted-foreground))]' }, copy.runtimeComponents),
        h('dd', { key: 'values' }, feature.runtimeComponents?.length
          ? h('div', { className: 'space-y-1.5' }, feature.runtimeComponents.map((component) => h('div', { key: runtimeComponentKey(component), className: 'rounded bg-[hsl(var(--muted))] px-2 py-1.5' }, [
            h('span', { key: 'label', className: component.status === 'error' || component.lastError ? 'font-mono text-xs text-red-700 dark:text-red-300' : 'font-mono text-xs text-[hsl(var(--foreground))]' }, runtimeComponentLabel(component)),
            component.lastError ? h('p', { key: 'error', className: 'mt-1 break-words text-xs text-red-700 dark:text-red-300', 'data-test': `runtime-error-${feature.id}` }, component.lastError) : null,
          ])))
          : h('span', { className: 'text-sm text-[hsl(var(--muted-foreground))]' }, copy.noRuntimeComponents)),
      ]),
    ]),
    feature.override !== null && feature.override !== undefined
      ? h('div', { key: 'restore', className: 'mt-4 flex justify-end border-t border-[hsl(var(--border))] pt-4' }, h('button', {
        type: 'button',
        className: 'btn btn-secondary btn-sm',
        disabled: busy,
        onClick: () => onRestoreDefault(feature),
      }, copy.restoreDefault))
      : null,
  ])
}

export default function FeatureRegistryPage({
  loading,
  error,
  profile,
  groups,
  busyFeatureID,
  copy,
  onRefresh,
  onSetEnabled,
  onRestoreDefault,
}: FeatureRegistryPageProps) {
  const featureCount = groups.reduce((total, group) => total + group.features.length, 0)
  return h('div', { className: 'space-y-6' }, [
    h('div', { key: 'header', className: 'flex justify-end' }, [
      h('button', { key: 'refresh', type: 'button', className: 'btn btn-secondary w-fit', disabled: loading, onClick: onRefresh }, [
        h(Icon, { key: 'icon', name: 'refresh', className: loading ? 'h-4 w-4 animate-spin' : 'h-4 w-4' }),
        copy.refresh,
      ]),
    ]),
    h('div', { key: 'summary', className: 'flex flex-wrap items-center gap-2 text-sm' }, [
      profile ? h('span', { key: 'profile', className: 'rounded-full bg-[hsl(var(--muted))] px-3 py-1 text-[hsl(var(--foreground))]' }, copy.profile(profile)) : null,
      h('span', { key: 'count', className: 'rounded-full bg-[hsl(var(--muted))] px-3 py-1 text-[hsl(var(--muted-foreground))]' }, copy.featureCount(featureCount)),
    ]),
    error ? h('div', { key: 'error', className: 'flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-200', role: 'alert' }, [
      h(Icon, { key: 'icon', name: 'error', className: 'mt-0.5 h-5 w-5 shrink-0' }),
      h('p', { key: 'message', className: 'whitespace-pre-line break-words' }, error),
    ]) : null,
    !loading && groups.length === 0 && !error ? h('div', { key: 'empty', className: 'card p-6 text-center' }, [
      h('p', { key: 'title', className: 'text-sm font-medium text-[hsl(var(--foreground))]' }, copy.noFeatures),
      h('p', { key: 'hint', className: 'mt-1 text-sm text-[hsl(var(--muted-foreground))]' }, copy.noFeaturesHint),
    ]) : null,
    ...groups.map((group) => h('section', { key: group.id, className: 'space-y-3' }, [
      h('div', { key: 'heading', className: 'flex items-center gap-3' }, [
        h('h2', { key: 'label', className: 'text-base font-semibold text-[hsl(var(--foreground))]' }, copy.tierLabel(group.id)),
        h('span', { key: 'count', className: 'rounded-full bg-[hsl(var(--muted))] px-2.5 py-1 text-xs text-[hsl(var(--muted-foreground))]' }, group.features.length),
      ]),
      h('div', { key: 'cards', className: 'grid grid-cols-1 gap-4 xl:grid-cols-2' }, group.features.map((feature) => h(FeatureCard, {
        key: feature.id,
        feature,
        copy,
        busy: busyFeatureID === feature.id,
        onSetEnabled,
        onRestoreDefault,
      }))),
    ])),
  ])
}
