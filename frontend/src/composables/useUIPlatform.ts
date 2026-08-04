import { computed, readonly, ref } from 'vue'
import { DEFAULT_UI_PROFILE, UI_PLATFORM_EVENT, UI_REGISTRY } from '@/ui-platform/registry'
import {
  getCurrentUIProfile,
  initializeUIPlatform,
  previewUIProfile,
  resetLocalUIProfile,
  saveLocalUIProfile,
} from '@/ui-platform/runtime'
import type { UIProfile, UIProfileOverrides, UIRegistryEntry } from '@/ui-platform/types'

const profile = ref<UIProfile>({ ...DEFAULT_UI_PROFILE })
let initialized = false
let listenerInstalled = false

function syncFromRuntime() {
  profile.value = { ...getCurrentUIProfile() }
}

function ensureInitialized() {
  if (!initialized) {
    profile.value = { ...initializeUIPlatform() }
    initialized = true
  }

  if (!listenerInstalled && typeof window !== 'undefined') {
    window.addEventListener(UI_PLATFORM_EVENT, (event) => {
      profile.value = { ...(event as CustomEvent<UIProfile>).detail }
    })
    listenerInstalled = true
  }
}

function toOverrides(value: UIProfile): UIProfileOverrides {
  const { componentStyle: _fixed, ...overrides } = value
  return overrides
}

function updatePreferences(patch: UIProfileOverrides) {
  ensureInitialized()
  const next = {
    ...toOverrides(profile.value),
    ...patch,
  }
  profile.value = saveLocalUIProfile(next)
}

function previewPreferences(patch: UIProfileOverrides) {
  ensureInitialized()
  profile.value = previewUIProfile(patch)
}

function resetPreferences() {
  profile.value = resetLocalUIProfile()
}

function entries(record: Record<string, UIRegistryEntry>) {
  return Object.values(record)
}

export function useUIPlatform() {
  ensureInitialized()

  return {
    profile: readonly(profile),
    isLegacy: computed(() => profile.value.mode === 'legacy'),
    isModern: computed(() => profile.value.mode === 'modern'),
    isPackage: computed(() => profile.value.mode === 'package'),
    options: {
      layouts: entries(UI_REGISTRY.layouts),
      baseColors: entries(UI_REGISTRY.baseColors),
      chartColors: entries(UI_REGISTRY.chartColors),
      headings: entries(UI_REGISTRY.headings),
      fonts: entries(UI_REGISTRY.fonts),
      iconLibraries: entries(UI_REGISTRY.iconLibraries),
      radii: entries(UI_REGISTRY.radii),
      densities: entries(UI_REGISTRY.densities),
      menus: entries(UI_REGISTRY.menus),
      menuSizes: entries(UI_REGISTRY.menuSizes),
      motions: entries(UI_REGISTRY.motions),
      tableStyles: entries(UI_REGISTRY.tableStyles),
    },
    updatePreferences,
    previewPreferences,
    resetPreferences,
    syncFromRuntime,
  }
}
