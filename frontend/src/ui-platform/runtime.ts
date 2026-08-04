import { UI_PLATFORM_PORTAL_ID, UI_PLATFORM_STORAGE_KEY } from './registry'
import { parseStoredUIProfile, resolveUIProfile } from './resolver'
import type { UIProfile, UIProfileOverrides } from './types'

const profileAttributes: Array<[keyof UIProfile, string]> = [
  ['mode', 'data-ui-mode'],
  ['componentStyle', 'data-ui-style'],
  ['layout', 'data-ui-layout'],
  ['baseColor', 'data-ui-base-color'],
  ['chartColor', 'data-ui-chart-color'],
  ['heading', 'data-ui-heading'],
  ['font', 'data-ui-font'],
  ['iconLibrary', 'data-ui-icons'],
  ['radius', 'data-ui-radius'],
  ['density', 'data-ui-density'],
  ['menu', 'data-ui-menu'],
  ['menuSize', 'data-ui-menu-size'],
  ['motion', 'data-ui-motion'],
  ['tableStyle', 'data-ui-table'],
]

export function ensureUIPortalRoot(): HTMLElement {
  let portal = document.getElementById(UI_PLATFORM_PORTAL_ID)
  if (!portal) {
    portal = document.createElement('div')
    portal.id = UI_PLATFORM_PORTAL_ID
    document.body.appendChild(portal)
  }
  return portal
}

export function applyUIProfile(profile: UIProfile) {
  const roots = [document.documentElement, ensureUIPortalRoot()]

  for (const root of roots) {
    for (const [key, attribute] of profileAttributes) {
      root.setAttribute(attribute, String(profile[key]))
    }

    if (profile.activePackageId) {
      root.setAttribute('data-ui-package', profile.activePackageId)
    } else {
      root.removeAttribute('data-ui-package')
    }
  }
}

export function loadLocalUIProfile(): UIProfileOverrides | undefined {
  return parseStoredUIProfile(localStorage.getItem(UI_PLATFORM_STORAGE_KEY))
}

export function saveLocalUIProfile(profile: UIProfileOverrides) {
  const { componentStyle: _ignored, ...safe } = profile as UIProfileOverrides & {
    componentStyle?: unknown
  }
  localStorage.setItem(UI_PLATFORM_STORAGE_KEY, JSON.stringify(safe))
}

export function initializeUIPlatform(): UIProfile {
  const { profile, warnings } = resolveUIProfile({ localPreferences: loadLocalUIProfile() })
  applyUIProfile(profile)

  for (const warning of warnings) {
    console.warn(`[ui-platform] ${warning}`)
  }

  return profile
}
