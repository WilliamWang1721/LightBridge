import { runtimeAppearanceFromSettings } from './preset'
import type { AppearanceTokens } from './types'

const STYLE_ID = 'lightbridge-runtime-appearance'
export const RUNTIME_APPEARANCE_EVENT = 'lightbridge:runtime-appearance-changed'
const TOKEN_KEYS: Array<keyof AppearanceTokens> = [
  '--background', '--foreground', '--card', '--card-foreground', '--popover', '--popover-foreground',
  '--primary', '--primary-foreground', '--secondary', '--secondary-foreground', '--muted', '--muted-foreground',
  '--accent', '--accent-foreground', '--border', '--input', '--ring', '--chart-1', '--chart-2', '--chart-3',
  '--chart-4', '--chart-5', '--ui-radius',
]

function tokenBlock(tokens: AppearanceTokens): string {
  return TOKEN_KEYS.map((key) => `${key}: ${tokens[key]};`).join('')
}

export function applyRuntimeAppearance(code?: string | null): void {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  const preview = runtimeAppearanceFromSettings(code)
  let style = document.getElementById(STYLE_ID) as HTMLStyleElement | null

  if (!preview) {
    root.removeAttribute('data-ui-appearance')
    style?.remove()
    window.dispatchEvent(new CustomEvent(RUNTIME_APPEARANCE_EVENT))
    return
  }

  root.setAttribute('data-ui-appearance', preview.code)
  if (!style) {
    style = document.createElement('style')
    style.id = STYLE_ID
    style.setAttribute('data-lightbridge-runtime-appearance', '')
    document.head.appendChild(style)
  }
  style.textContent = `:root[data-ui-appearance] {${tokenBlock(preview.tokens.light)}} .dark[data-ui-appearance] {${tokenBlock(preview.tokens.dark)}} :root[data-ui-appearance] #lightbridge-ui-portal {${tokenBlock(preview.tokens.light)}} .dark[data-ui-appearance] #lightbridge-ui-portal {${tokenBlock(preview.tokens.dark)}}`
  window.dispatchEvent(new CustomEvent(RUNTIME_APPEARANCE_EVENT))
}

export function appearanceTokensStyle(tokens: AppearanceTokens): Record<string, string> {
  return Object.fromEntries(TOKEN_KEYS.map((key) => [key, tokens[key]]))
}
