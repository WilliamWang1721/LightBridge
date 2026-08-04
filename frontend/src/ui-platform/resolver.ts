import { DEFAULT_UI_PROFILE, UI_REGISTRY } from './registry'
import type { UIAxis, UIProfile, UIProfileOverrides, UIResolutionInput, UIResolutionResult } from './types'

const registryByAxis: Record<UIAxis, Record<string, { id: string }>> = {
  layout: UI_REGISTRY.layouts,
  baseColor: UI_REGISTRY.baseColors,
  chartColor: UI_REGISTRY.chartColors,
  heading: UI_REGISTRY.headings,
  font: UI_REGISTRY.fonts,
  iconLibrary: UI_REGISTRY.iconLibraries,
  radius: UI_REGISTRY.radii,
  density: UI_REGISTRY.densities,
  menu: UI_REGISTRY.menus,
  menuSize: UI_REGISTRY.menuSizes,
  motion: UI_REGISTRY.motions,
  tableStyle: UI_REGISTRY.tableStyles,
}

const axes = Object.keys(registryByAxis) as UIAxis[]

function applyLayer(target: UIProfile, layer: UIProfileOverrides | undefined) {
  if (!layer) return
  Object.assign(target, layer)
}

export function resolveUIProfile(input: UIResolutionInput = {}): UIResolutionResult {
  const profile: UIProfile = { ...DEFAULT_UI_PROFILE }
  const warnings: string[] = []

  applyLayer(profile, input.adminDefaults)
  applyLayer(profile, input.packageDefaults)
  applyLayer(profile, input.localPreferences)
  applyLayer(profile, input.userPreferences)
  applyLayer(profile, input.previewOverrides)

  profile.componentStyle = 'luma'

  if (!['legacy', 'modern', 'package'].includes(profile.mode)) {
    warnings.push(`Unsupported UI mode "${String(profile.mode)}"; using legacy.`)
    profile.mode = 'legacy'
  }

  for (const axis of axes) {
    const selected = profile[axis]
    if (!registryByAxis[axis][selected]) {
      warnings.push(`Unsupported ${axis} value "${selected}"; using ${DEFAULT_UI_PROFILE[axis]}.`)
      profile[axis] = DEFAULT_UI_PROFILE[axis]
    }
  }

  if (profile.mode !== 'package') {
    delete profile.activePackageId
  } else if (!profile.activePackageId) {
    warnings.push('Package mode requires an active package; using modern mode.')
    profile.mode = 'modern'
  }

  return { profile, warnings }
}

export function parseStoredUIProfile(raw: string | null): UIProfileOverrides | undefined {
  if (!raw) return undefined
  try {
    const value = JSON.parse(raw)
    if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined
    const { componentStyle: _ignored, ...safe } = value as Record<string, unknown>
    return safe as UIProfileOverrides
  } catch {
    return undefined
  }
}
