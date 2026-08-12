import * as React from 'react'
import { createShadcnElement as h } from './ui/createElement'
import { Badge, Button, Card, CardContent, CardDescription, CardHeader, CardTitle, Input } from './ui'
import { appearanceTokensStyle } from '@/appearance/runtime'
import type { RuntimeAppearancePreview } from '@/appearance/types'
import type { RuntimeAppearanceSettings } from '@/types'

export interface AppearancePageCopy {
  presetCode: string
  presetCodeHint: string
  preview: string
  save: string
  reset: string
  applied: string
  deploymentControlled: string
  unavailable: string
  previewCard: string
  previewDescription: string
  previewButton: string
  previewInput: string
  activeItem: string
  lightMode: string
  darkMode: string
  previewBrand: string
  previewSettings: string
  noPreset: string
  invalidPreset: string
  controlledStyle: string
  controlledIcons: string
}

export interface AppearancePageProps {
  appearance: RuntimeAppearanceSettings | null
  preview: RuntimeAppearancePreview | null
  loading: boolean
  saving: boolean
  error: string
  copy: AppearancePageCopy
  onPreview: (code: string) => void
  onSave: (code: string) => void
  onReset: () => void
}

function chip(text: string, variant: 'secondary' | 'outline' | 'destructive' = 'secondary') {
  return h(Badge, { variant, key: text }, text)
}

export default function AppearancePage({
  appearance,
  preview,
  loading,
  saving,
  error,
  copy,
  onPreview,
  onSave,
  onReset,
}: AppearancePageProps): React.ReactElement {
  const [code, setCode] = React.useState(appearance?.preset_code || '')
  const [mode, setMode] = React.useState<'light' | 'dark'>('light')

  React.useEffect(() => {
    setCode(appearance?.preset_code || '')
  }, [appearance?.preset_code])

  const activePreview = preview || null
  const tokens = activePreview?.tokens[mode]
  const previewStyle = tokens ? appearanceTokensStyle(tokens) : undefined

  return h('div', { className: 'space-y-6' }, [
    h(Card, { key: 'editor' }, [
      h(CardHeader, { key: 'header' }, [
        h(CardTitle, { key: 'title' }, copy.presetCode),
        h(CardDescription, { key: 'hint' }, copy.presetCodeHint),
      ]),
      h(CardContent, { key: 'content', className: 'space-y-4' }, [
        h('form', {
          key: 'form',
          className: 'flex flex-col gap-3 sm:flex-row sm:items-center',
          onSubmit: (event: React.FormEvent<HTMLFormElement>) => {
            event.preventDefault()
            onPreview(code)
          },
        }, [
          h(Input, {
            key: 'code',
            value: code,
            onChange: (event: React.ChangeEvent<HTMLInputElement>) => setCode(event.target.value),
            placeholder: 'b0',
            spellCheck: false,
            'aria-label': copy.presetCode,
            className: 'font-mono sm:flex-1',
            disabled: loading || saving,
          }),
          h(Button, { key: 'preview', type: 'submit', variant: 'outline', disabled: loading || saving }, copy.preview),
          h(Button, { key: 'save', type: 'button', disabled: loading || saving || !code.trim(), onClick: () => onSave(code) }, copy.save),
          h(Button, { key: 'reset', type: 'button', variant: 'ghost', disabled: loading || saving, onClick: onReset }, copy.reset),
        ]),
        error ? h('p', { key: 'error', role: 'alert', className: 'text-sm text-[hsl(var(--destructive))]' }, error) : null,
      ]),
    ]),
    activePreview
      ? h(Card, { key: 'compatibility' }, [
        h(CardHeader, { key: 'header' }, [
          h(CardTitle, { key: 'title' }, activePreview.preset.style),
          h(CardDescription, { key: 'description' }, activePreview.code),
        ]),
        h(CardContent, { key: 'content', className: 'space-y-3' }, [
          h('div', { key: 'applied', className: 'space-y-2' }, [
            h('p', { key: 'label', className: 'text-sm font-medium' }, copy.applied),
            h('div', { key: 'items', className: 'flex flex-wrap gap-2' }, activePreview.compatibility.applied.map((item) => chip(item))),
          ]),
          activePreview.compatibility.deploymentControlled.length
            ? h('div', { key: 'controlled', className: 'space-y-2' }, [
              h('p', { key: 'label', className: 'text-sm font-medium' }, copy.deploymentControlled),
              h('div', { key: 'items', className: 'flex flex-wrap gap-2' }, activePreview.compatibility.deploymentControlled.map((item) => chip(item, 'outline'))),
            ])
            : null,
          activePreview.compatibility.unavailable.length
            ? h('div', { key: 'unavailable', className: 'space-y-2' }, [
              h('p', { key: 'label', className: 'text-sm font-medium' }, copy.unavailable),
              h('div', { key: 'items', className: 'flex flex-wrap gap-2' }, activePreview.compatibility.unavailable.map((item) => chip(item, 'destructive'))),
            ])
            : null,
        ]),
      ])
      : h(Card, { key: 'empty' }, [h(CardContent, { className: 'py-6 text-sm text-[hsl(var(--muted-foreground))]' }, copy.noPreset)]),
    activePreview
      ? h(Card, { key: 'preview' }, [
        h(CardHeader, { key: 'header', className: 'flex-row items-center justify-between space-y-0' }, [
          h('div', { key: 'copy' }, [
            h(CardTitle, { key: 'title' }, copy.previewCard),
            h(CardDescription, { key: 'description' }, copy.previewDescription),
          ]),
          h('div', { key: 'modes', className: 'flex gap-2' }, [
            h(Button, { key: 'light', type: 'button', variant: mode === 'light' ? 'secondary' : 'ghost', size: 'sm', onClick: () => setMode('light') }, copy.lightMode),
            h(Button, { key: 'dark', type: 'button', variant: mode === 'dark' ? 'secondary' : 'ghost', size: 'sm', onClick: () => setMode('dark') }, copy.darkMode),
          ]),
        ]),
        h(CardContent, { key: 'content' }, [
          h('div', { key: 'surface', className: 'grid gap-4 rounded-[var(--ui-radius)] border border-[hsl(var(--border))] bg-[hsl(var(--background))] p-5 lg:grid-cols-[12rem_1fr]', style: previewStyle }, [
            h('nav', { key: 'nav', className: 'space-y-1 rounded-[var(--ui-radius)] border border-[hsl(var(--border))] bg-[hsl(var(--card))] p-2', 'aria-label': copy.previewCard }, [
              h('div', { key: 'label', className: 'px-3 py-2 text-xs font-semibold uppercase tracking-wide text-[hsl(var(--muted-foreground))]' }, copy.previewBrand),
              h('div', { key: 'active', className: 'rounded-[var(--ui-radius)] bg-[hsl(var(--accent))] px-3 py-2 text-sm font-medium text-[hsl(var(--accent-foreground))]' }, copy.activeItem),
              h('div', { key: 'item', className: 'px-3 py-2 text-sm text-[hsl(var(--muted-foreground))]' }, copy.previewSettings),
            ]),
            h('section', { key: 'body', className: 'rounded-[var(--ui-radius)] border border-[hsl(var(--border))] bg-[hsl(var(--card))] p-5' }, [
              h('h3', { key: 'heading', className: 'text-base font-semibold text-[hsl(var(--foreground))]' }, copy.previewCard),
              h('p', { key: 'text', className: 'mt-2 text-sm text-[hsl(var(--muted-foreground))]' }, copy.previewDescription),
              h(Input, { key: 'input', className: 'mt-4', placeholder: copy.previewInput, 'aria-label': copy.previewInput }),
              h(Button, { key: 'button', className: 'mt-3', type: 'button' }, copy.previewButton),
            ]),
          ]),
        ]),
      ])
      : null,
  ]) as React.ReactElement
}
