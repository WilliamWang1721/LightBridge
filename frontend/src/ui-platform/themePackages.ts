export type UIThemeConfigFieldType = 'color' | 'text' | 'select' | 'number' | 'boolean'

export interface UIThemeConfigField {
  key: string
  label: string
  type: UIThemeConfigFieldType
  default?: unknown
  options?: string[]
  description?: string
  placeholder?: string
  min?: number
  max?: number
  step?: number
}

export interface UIThemeManifest {
  id?: string
  name?: string
  version?: string
  entry_css?: string
  preview?: string
  config?: UIThemeConfigField[]
  meta?: Record<string, unknown>
}

export interface ThemeConfigValidationResult {
  value: Record<string, unknown>
  errors: Record<string, string>
}

export function parseThemeManifest(value: unknown): UIThemeManifest {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return {}
  return value as UIThemeManifest
}

export function getThemeConfigFields(manifest: unknown): UIThemeConfigField[] {
  const parsed = parseThemeManifest(manifest)
  if (!Array.isArray(parsed.config)) return []

  return parsed.config.filter((field): field is UIThemeConfigField => {
    return Boolean(
      field &&
        typeof field.key === 'string' &&
        field.key.trim() &&
        typeof field.label === 'string' &&
        ['color', 'text', 'select', 'number', 'boolean'].includes(field.type),
    )
  })
}

export function buildThemeConfigDraft(
  manifest: unknown,
  current: Record<string, unknown> | undefined,
): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  const source = current || {}

  for (const field of getThemeConfigFields(manifest)) {
    if (Object.prototype.hasOwnProperty.call(source, field.key)) {
      result[field.key] = coerceThemeConfigValue(field, source[field.key])
    } else if (field.default !== undefined) {
      result[field.key] = coerceThemeConfigValue(field, field.default)
    } else {
      result[field.key] = emptyValueForField(field)
    }
  }

  return result
}

export function resetThemeConfig(manifest: unknown): Record<string, unknown> {
  return buildThemeConfigDraft(manifest, {})
}

export function validateThemeConfig(
  manifest: unknown,
  draft: Record<string, unknown>,
): ThemeConfigValidationResult {
  const value: Record<string, unknown> = {}
  const errors: Record<string, string> = {}

  for (const field of getThemeConfigFields(manifest)) {
    const normalized = coerceThemeConfigValue(field, draft[field.key])
    const error = validateField(field, normalized)
    if (error) errors[field.key] = error
    value[field.key] = normalized
  }

  return { value, errors }
}

export function themeAssetURL(themeId: string, assetPath?: string): string {
  if (!assetPath) return ''
  const normalized = assetPath.replace(/^\/+/, '')
  return `/ui-themes/${encodeURIComponent(themeId)}/${normalized
    .split('/')
    .map((segment) => encodeURIComponent(segment))
    .join('/')}`
}

function emptyValueForField(field: UIThemeConfigField): unknown {
  switch (field.type) {
    case 'boolean':
      return false
    case 'number':
      return field.min ?? 0
    default:
      return ''
  }
}

function coerceThemeConfigValue(field: UIThemeConfigField, value: unknown): unknown {
  switch (field.type) {
    case 'boolean':
      if (typeof value === 'string') return value === 'true'
      return Boolean(value)
    case 'number': {
      const parsed = typeof value === 'number' ? value : Number(value)
      return Number.isFinite(parsed) ? parsed : field.min ?? 0
    }
    case 'select': {
      const rendered = value == null ? '' : String(value)
      if (!field.options?.length || field.options.includes(rendered)) return rendered
      return field.default == null ? field.options[0] : String(field.default)
    }
    case 'color':
    case 'text':
      return value == null ? '' : String(value)
  }
}

function validateField(field: UIThemeConfigField, value: unknown): string | undefined {
  if (field.type === 'select' && field.options?.length && !field.options.includes(String(value))) {
    return 'Select one of the options provided by the UI package.'
  }

  if (field.type === 'number') {
    const number = Number(value)
    if (!Number.isFinite(number)) return 'Enter a valid number.'
    if (field.min !== undefined && number < field.min) return `Minimum value is ${field.min}.`
    if (field.max !== undefined && number > field.max) return `Maximum value is ${field.max}.`
  }

  if (field.type === 'color') {
    const color = String(value)
    if (color && !/^#[0-9a-f]{3,8}$/i.test(color)) return 'Use a hexadecimal color value.'
  }

  return undefined
}
