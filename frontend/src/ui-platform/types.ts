export type UIMode = 'legacy' | 'modern' | 'package'
export type ComponentStyle = 'luma'

export type UIAxis =
  | 'layout'
  | 'baseColor'
  | 'chartColor'
  | 'heading'
  | 'font'
  | 'iconLibrary'
  | 'radius'
  | 'density'
  | 'menu'
  | 'menuSize'
  | 'motion'
  | 'tableStyle'

export interface UICapabilities {
  supportsDarkMode: boolean
  supportsRTL: boolean
  supportsCompactSidebar: boolean
  supportsMobileDrawer: boolean
  supportsDenseTables: boolean
  supportsCustomChartPalette: boolean
  supportsUserOverrides: boolean
  supportsPortalSurfaces: boolean
}

export interface UIProfile {
  mode: UIMode
  componentStyle: ComponentStyle
  layout: string
  baseColor: string
  chartColor: string
  heading: string
  font: string
  iconLibrary: string
  radius: string
  density: string
  menu: string
  menuSize: string
  motion: string
  tableStyle: string
  activePackageId?: string
}

export type UIProfileOverrides = Partial<Omit<UIProfile, 'componentStyle'>> & {
  componentStyle?: never
}

export interface UIRegistryEntry {
  id: string
  label: string
  description?: string
  capabilities?: Partial<UICapabilities>
}

export interface UIRegistry {
  layouts: Record<string, UIRegistryEntry>
  baseColors: Record<string, UIRegistryEntry>
  chartColors: Record<string, UIRegistryEntry>
  headings: Record<string, UIRegistryEntry>
  fonts: Record<string, UIRegistryEntry>
  iconLibraries: Record<string, UIRegistryEntry>
  radii: Record<string, UIRegistryEntry>
  densities: Record<string, UIRegistryEntry>
  menus: Record<string, UIRegistryEntry>
  menuSizes: Record<string, UIRegistryEntry>
  motions: Record<string, UIRegistryEntry>
  tableStyles: Record<string, UIRegistryEntry>
}

export interface UIResolutionInput {
  adminDefaults?: UIProfileOverrides
  packageDefaults?: UIProfileOverrides
  localPreferences?: UIProfileOverrides
  userPreferences?: UIProfileOverrides
  previewOverrides?: UIProfileOverrides
}

export interface UIResolutionResult {
  profile: UIProfile
  warnings: string[]
}
