import { UI_PLATFORM_EVENT, UI_PLATFORM_PORTAL_ID, UI_PLATFORM_STORAGE_KEY } from './registry'
import { parseStoredUIProfile, resolveUIProfile } from './resolver'
import type { UIProfile, UIProfileOverrides, UIResolutionInput } from './types'

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

let currentProfile: UIProfile | undefined

export function ensureUIPortalRoot(): HTMLElement {
  let portal = document.getElementById(UI_PLATFORM_PORTAL_ID)
  if (!portal) {
    portal = document.createElement('div')
    portal.id = UI_PLATFORM_PORTAL_ID
    document.body.appendChild(portal)
  }
  return portal
}

export function applyUIProfile(profile: UIProfile, notify = true) {
  currentProfile = { ...profile }
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

  if (notify) {
    window.dispatchEvent(new CustomEvent<UIProfile>(UI_PLATFORM_EVENT, { detail: { ...profile } }))
  }
}

export function getCurrentUIProfile(): UIProfile {
  if (currentProfile) return { ...currentProfile }
  return initializeUIPlatform()
}

export function loadLocalUIProfile(): UIProfileOverrides | undefined {
  return parseStoredUIProfile(localStorage.getItem(UI_PLATFORM_STORAGE_KEY))
}

export function saveLocalUIProfile(overrides: UIProfileOverrides): UIProfile {
  const { componentStyle: _ignored, ...safe } = overrides as UIProfileOverrides & {
    componentStyle?: unknown
  }
  localStorage.setItem(UI_PLATFORM_STORAGE_KEY, JSON.stringify(safe))
  const { profile, warnings } = resolveUIProfile({ localPreferences: safe })
  applyUIProfile(profile)
  reportWarnings(warnings)
  return profile
}

export function resetLocalUIProfile(): UIProfile {
  localStorage.removeItem(UI_PLATFORM_STORAGE_KEY)
  const { profile } = resolveUIProfile()
  applyUIProfile(profile)
  return profile
}

export function previewUIProfile(overrides: UIProfileOverrides): UIProfile {
  const { profile, warnings } = resolveUIProfile({
    localPreferences: loadLocalUIProfile(),
    previewOverrides: overrides,
  })
  applyUIProfile(profile)
  reportWarnings(warnings)
  return profile
}

export function resolveAndApplyUIProfile(input: UIResolutionInput): UIProfile {
  const { profile, warnings } = resolveUIProfile(input)
  applyUIProfile(profile)
  reportWarnings(warnings)
  return profile
}

export function initializeUIPlatform(): UIProfile {
  const { profile, warnings } = resolveUIProfile({ localPreferences: loadLocalUIProfile() })
  applyUIProfile(profile, false)
  reportWarnings(warnings)
  return profile
}

function reportWarnings(warnings: string[]) {
  for (const warning of warnings) {
    console.warn(`[ui-platform] ${warning}`)
  }
}
