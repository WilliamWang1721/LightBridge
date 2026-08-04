import type { UICapabilities, UIProfile, UIRegistry } from './types'

export const UI_PLATFORM_STORAGE_KEY = 'lightbridge.ui.profile.v1'
export const UI_PLATFORM_PORTAL_ID = 'lightbridge-ui-portal'
export const UI_PLATFORM_EVENT = 'lightbridge:ui-profile-changed'

export const DEFAULT_UI_CAPABILITIES: UICapabilities = {
  supportsDarkMode: true,
  supportsRTL: false,
  supportsCompactSidebar: true,
  supportsMobileDrawer: true,
  supportsDenseTables: true,
  supportsCustomChartPalette: true,
  supportsUserOverrides: true,
  supportsPortalSurfaces: true,
}

export const DEFAULT_UI_PROFILE: UIProfile = {
  mode: 'legacy',
  componentStyle: 'luma',
  layout: 'default',
  baseColor: 'natural',
  chartColor: 'natural',
  heading: 'natural',
  font: 'inter',
  iconLibrary: 'lucide',
  radius: 'default',
  density: 'default',
  menu: 'default',
  menuSize: 'suitable',
  motion: 'default',
  tableStyle: 'default',
}

export const UI_REGISTRY: UIRegistry = {
  layouts: {
    default: { id: 'default', label: 'Default' },
    compact: { id: 'compact', label: 'Compact' },
    spacious: { id: 'spacious', label: 'Spacious' },
  },
  baseColors: {
    natural: { id: 'natural', label: 'Natural' },
    neutral: { id: 'neutral', label: 'Neutral' },
    stone: { id: 'stone', label: 'Stone' },
    zinc: { id: 'zinc', label: 'Zinc' },
  },
  chartColors: {
    natural: { id: 'natural', label: 'Natural' },
    vivid: { id: 'vivid', label: 'Vivid' },
    muted: { id: 'muted', label: 'Muted' },
  },
  headings: {
    natural: { id: 'natural', label: 'Natural' },
    compact: { id: 'compact', label: 'Compact' },
    editorial: { id: 'editorial', label: 'Editorial' },
    strong: { id: 'strong', label: 'Strong' },
  },
  fonts: {
    inter: { id: 'inter', label: 'Inter' },
    system: { id: 'system', label: 'System' },
  },
  iconLibraries: {
    lucide: { id: 'lucide', label: 'Lucide' },
  },
  radii: {
    none: { id: 'none', label: 'None' },
    small: { id: 'small', label: 'Small' },
    default: { id: 'default', label: 'Default' },
    large: { id: 'large', label: 'Large' },
  },
  densities: {
    compact: { id: 'compact', label: 'Compact' },
    default: { id: 'default', label: 'Default' },
    comfortable: { id: 'comfortable', label: 'Comfortable' },
  },
  menus: {
    default: { id: 'default', label: 'Default' },
    subtle: { id: 'subtle', label: 'Subtle' },
    outlined: { id: 'outlined', label: 'Outlined' },
    translucent: { id: 'translucent', label: 'Translucent' },
  },
  menuSizes: {
    compact: { id: 'compact', label: 'Compact' },
    suitable: { id: 'suitable', label: 'Suitable' },
    comfortable: { id: 'comfortable', label: 'Comfortable' },
    wide: { id: 'wide', label: 'Wide' },
  },
  motions: {
    reduced: { id: 'reduced', label: 'Reduced' },
    default: { id: 'default', label: 'Default' },
    expressive: { id: 'expressive', label: 'Expressive' },
  },
  tableStyles: {
    compact: { id: 'compact', label: 'Compact' },
    default: { id: 'default', label: 'Default' },
    comfortable: { id: 'comfortable', label: 'Comfortable' },
  },
}
