import type { PresetConfig } from 'shadcn/preset'

export type AppearanceMode = 'light' | 'dark'

export interface AppearanceTokens {
  '--background': string
  '--foreground': string
  '--card': string
  '--card-foreground': string
  '--popover': string
  '--popover-foreground': string
  '--primary': string
  '--primary-foreground': string
  '--secondary': string
  '--secondary-foreground': string
  '--muted': string
  '--muted-foreground': string
  '--accent': string
  '--accent-foreground': string
  '--border': string
  '--input': string
  '--ring': string
  '--chart-1': string
  '--chart-2': string
  '--chart-3': string
  '--chart-4': string
  '--chart-5': string
}

export interface AppearanceCompatibility {
  applied: string[]
  deploymentControlled: string[]
  unavailable: string[]
}

export interface RuntimeAppearancePreview {
  code: string
  preset: PresetConfig
  profile: {
    baseColor: 'neutral' | 'stone' | 'zinc'
    chartColor: 'natural' | 'vivid' | 'muted'
    font: 'inter' | 'system'
    heading: 'natural' | 'editorial'
    radius: 'none' | 'small' | 'default' | 'large'
    menu: 'default' | 'subtle' | 'outlined' | 'translucent'
  }
  tokens: Record<AppearanceMode, AppearanceTokens>
  compatibility: AppearanceCompatibility
}

export interface RuntimeAppearanceSettings {
  version: number
  source: string
  preset_code: string
}
