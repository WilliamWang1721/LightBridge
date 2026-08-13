import {
  decodePreset,
  isPresetCode,
  isValidPreset,
  type PresetConfig,
} from 'shadcn/preset'
import type { AppearanceCompatibility, AppearanceTokens, RuntimeAppearancePreview } from './types'

export const DEFAULT_RUNTIME_APPEARANCE_PRESET_CODE = 'b7Br6guwa'

const LIGHT_TOKENS: AppearanceTokens = {
  '--background': '0 0% 98%',
  '--foreground': '0 0% 9%',
  '--card': '0 0% 100%',
  '--card-foreground': '0 0% 12%',
  '--popover': '0 0% 100%',
  '--popover-foreground': '0 0% 12%',
  '--primary': '0 0% 13%',
  '--primary-foreground': '0 0% 100%',
  '--secondary': '0 0% 94%',
  '--secondary-foreground': '0 0% 16%',
  '--muted': '0 0% 94%',
  '--muted-foreground': '0 0% 42%',
  '--accent': '0 0% 92%',
  '--accent-foreground': '0 0% 14%',
  '--border': '0 0% 87%',
  '--input': '0 0% 87%',
  '--ring': '0 0% 28%',
  '--chart-1': '0 0% 12%',
  '--chart-2': '0 0% 28%',
  '--chart-3': '0 0% 44%',
  '--chart-4': '0 0% 60%',
  '--chart-5': '0 0% 76%',
  '--ui-radius': '0.625rem',
}

const DARK_TOKENS: AppearanceTokens = {
  '--background': '0 0% 8%',
  '--foreground': '0 0% 94%',
  '--card': '0 0% 11%',
  '--card-foreground': '0 0% 94%',
  '--popover': '0 0% 11%',
  '--popover-foreground': '0 0% 94%',
  '--primary': '0 0% 93%',
  '--primary-foreground': '0 0% 10%',
  '--secondary': '0 0% 17%',
  '--secondary-foreground': '0 0% 90%',
  '--muted': '0 0% 16%',
  '--muted-foreground': '0 0% 65%',
  '--accent': '0 0% 21%',
  '--accent-foreground': '0 0% 94%',
  '--border': '0 0% 22%',
  '--input': '0 0% 22%',
  '--ring': '0 0% 75%',
  '--chart-1': '0 0% 92%',
  '--chart-2': '0 0% 78%',
  '--chart-3': '0 0% 64%',
  '--chart-4': '0 0% 50%',
  '--chart-5': '0 0% 36%',
  '--ui-radius': '0.625rem',
}

const BASE_COLOR_TOKENS: Record<string, Partial<AppearanceTokens>> = {
  neutral: {},
  stone: {
    '--background': '30 12% 98%', '--foreground': '24 10% 10%', '--muted': '30 8% 94%',
    '--muted-foreground': '25 5% 42%', '--border': '25 7% 86%', '--input': '25 7% 86%',
  },
  zinc: {
    '--background': '0 0% 98%', '--foreground': '240 10% 10%', '--muted': '240 5% 94%',
    '--muted-foreground': '240 4% 42%', '--border': '240 6% 86%', '--input': '240 6% 86%',
  },
  mist: {
    '--background': '210 20% 98%', '--foreground': '215 18% 10%', '--muted': '210 16% 94%',
    '--muted-foreground': '215 10% 42%', '--border': '214 15% 86%', '--input': '214 15% 86%',
  },
}

const CHART_TOKENS: Record<string, Partial<AppearanceTokens>> = {
  neutral: {},
  vivid: {
    '--chart-1': '221 83% 53%', '--chart-2': '271 81% 56%', '--chart-3': '38 92% 50%',
    '--chart-4': '0 84% 60%', '--chart-5': '190 90% 44%',
  },
  muted: {
    '--chart-1': '0 0% 24%', '--chart-2': '0 0% 38%', '--chart-3': '0 0% 52%',
    '--chart-4': '0 0% 66%', '--chart-5': '0 0% 80%',
  },
  mist: {
    '--chart-1': '215 18% 22%', '--chart-2': '215 14% 36%', '--chart-3': '212 11% 50%',
    '--chart-4': '210 9% 64%', '--chart-5': '208 8% 78%',
  },
}

const SUPPORTED_BASE_COLORS = new Set(Object.keys(BASE_COLOR_TOKENS))
const SUPPORTED_CHART_COLORS = new Set(Object.keys(CHART_TOKENS))
const SUPPORTED_FONTS = new Set(['inter', 'system'])
const RADIUS_TOKENS: Record<string, string> = {
  none: '0',
  small: '0.375rem',
  default: '0.625rem',
  medium: '0.5rem',
  large: '0.875rem',
}

const SUPPORTED_RADII = new Set(Object.keys(RADIUS_TOKENS))

function cloneTokens(tokens: AppearanceTokens): AppearanceTokens {
  return { ...tokens }
}

function makeTokens(mode: 'light' | 'dark', preset: PresetConfig): AppearanceTokens {
  const tokens = cloneTokens(mode === 'light' ? LIGHT_TOKENS : DARK_TOKENS)
  Object.assign(tokens, BASE_COLOR_TOKENS[preset.baseColor] || {})
  Object.assign(tokens, CHART_TOKENS[preset.chartColor || 'neutral'] || {})
  tokens['--ui-radius'] = RADIUS_TOKENS[preset.radius] || RADIUS_TOKENS.default
  return tokens
}

function compatibilityFor(preset: PresetConfig): AppearanceCompatibility {
  const applied: string[] = []
  const unavailable: string[] = []
  const deploymentControlled = [`style:${preset.style} → luma`, `iconLibrary:${preset.iconLibrary} → lucide`]

  if (SUPPORTED_BASE_COLORS.has(preset.baseColor)) applied.push(`baseColor:${preset.baseColor}`)
  else unavailable.push(`baseColor:${preset.baseColor}`)
  if (SUPPORTED_CHART_COLORS.has(preset.chartColor || 'neutral')) applied.push(`chartColor:${preset.chartColor || 'neutral'}`)
  else unavailable.push(`chartColor:${preset.chartColor || 'neutral'}`)
  if (SUPPORTED_FONTS.has(preset.font)) applied.push(`font:${preset.font}`)
  else unavailable.push(`font:${preset.font}`)
  if (preset.fontHeading === 'inherit' || SUPPORTED_FONTS.has(preset.fontHeading)) applied.push(`fontHeading:${preset.fontHeading}`)
  else unavailable.push(`fontHeading:${preset.fontHeading}`)
  if (SUPPORTED_RADII.has(preset.radius)) applied.push(`radius:${preset.radius}`)
  else unavailable.push(`radius:${preset.radius}`)
  if (preset.menuAccent === 'subtle' && preset.menuColor === 'default') applied.push('menu:default')
  else unavailable.push(`menu:${preset.menuAccent}/${preset.menuColor}`)

  if (preset.theme === 'neutral') applied.push('theme:neutral')
  else deploymentControlled.push(`theme:${preset.theme} → neutral grayscale`)
  return { applied, deploymentControlled, unavailable }
}

export function decodeRuntimeAppearance(code: string): RuntimeAppearancePreview {
  const normalized = code.trim()
  if (!isPresetCode(normalized) || !isValidPreset(normalized)) throw new Error('Invalid shadcn preset code')
  const preset = decodePreset(normalized)
  if (!preset) throw new Error('Unsupported shadcn preset')

  const compatibility = compatibilityFor(preset)
  const profile: RuntimeAppearancePreview['profile'] = {
    baseColor: SUPPORTED_BASE_COLORS.has(preset.baseColor) ? preset.baseColor as 'neutral' | 'stone' | 'zinc' | 'mist' : 'neutral',
    chartColor: preset.chartColor === 'mist' ? 'muted' : 'natural',
    font: SUPPORTED_FONTS.has(preset.font) ? preset.font as 'inter' | 'system' : 'inter',
    heading: preset.fontHeading === 'inherit' ? 'natural' : 'editorial',
    radius: SUPPORTED_RADII.has(preset.radius) ? preset.radius as 'none' | 'small' | 'default' | 'medium' | 'large' : 'default',
    menu: preset.menuAccent === 'subtle' && preset.menuColor === 'default' ? 'default' : 'default',
  }

  return {
    code: normalized,
    preset,
    profile,
    tokens: { light: makeTokens('light', preset), dark: makeTokens('dark', preset) },
    compatibility,
  }
}

export function runtimeAppearanceFromSettings(code?: string | null): RuntimeAppearancePreview | null {
  const resolvedCode = code?.trim() || DEFAULT_RUNTIME_APPEARANCE_PRESET_CODE
  try {
    return decodeRuntimeAppearance(resolvedCode)
  } catch (error) {
    console.warn('[appearance] ignored invalid runtime preset', error)
    return decodeRuntimeAppearance(DEFAULT_RUNTIME_APPEARANCE_PRESET_CODE)
  }
}
