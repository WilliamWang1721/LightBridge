import { computed, readonly, ref } from 'vue'
import uiProfileAPI from '@/api/uiProfile'
import {
  DEFAULT_UI_PROFILE,
  UI_PLATFORM_EVENT,
  UI_PLATFORM_STORAGE_KEY,
  UI_REGISTRY,
} from '@/ui-platform/registry'
import {
  applyUIProfile,
  getCurrentUIProfile,
  initializeUIPlatform,
  loadLocalUIProfile,
} from '@/ui-platform/runtime'
import { resolveUIProfile, sanitizeUIProfileOverrides } from '@/ui-platform/resolver'
import type { UIProfile, UIProfileOverrides, UIRegistryEntry } from '@/ui-platform/types'
import { RUNTIME_APPEARANCE_EVENT } from '@/appearance/runtime'
import { runtimeAppearanceFromSettings } from '@/appearance/preset'

const profile = ref<UIProfile>({ ...DEFAULT_UI_PROFILE })
const accountPreferences = ref<UIProfileOverrides | undefined>()
const accountSyncing = ref(false)
const accountSyncError = ref('')
let initialized = false
let listenerInstalled = false
let accountProfileLoaded = false
let hydrationPromise: Promise<void> | null = null
let accountRequestGeneration = 0
let pendingAccountPreferences: UIProfileOverrides | undefined

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
    window.addEventListener(RUNTIME_APPEARANCE_EVENT, () => {
      profile.value = applyResolvedProfile()
    })
    listenerInstalled = true
  }
}

function toOverrides(value: UIProfile): UIProfileOverrides {
  const { componentStyle: _fixed, ...overrides } = value
  return sanitizeUIProfileOverrides(overrides) || {}
}

function storeLocalPreferences(value: UIProfileOverrides) {
  localStorage.setItem(UI_PLATFORM_STORAGE_KEY, JSON.stringify(sanitizeUIProfileOverrides(value) || {}))
}

function applyResolvedProfile(previewOverrides?: UIProfileOverrides) {
  const adminDefaults = runtimeAppearanceFromSettings(
    typeof window !== 'undefined' ? window.__APP_CONFIG__?.ui_appearance?.preset_code : undefined,
  )?.profile as UIProfileOverrides | undefined
  const { profile: resolved, warnings } = resolveUIProfile({
    adminDefaults,
    localPreferences: loadLocalUIProfile(),
    userPreferences: accountPreferences.value,
    previewOverrides,
  })
  applyUIProfile(resolved)
  for (const warning of warnings) console.warn(`[ui-platform] ${warning}`)
  return resolved
}

async function hydrateAccountPreferences() {
  ensureInitialized()
  if (hydrationPromise) return hydrationPromise

  const generation = ++accountRequestGeneration
  accountSyncing.value = true
  accountSyncError.value = ''
  const request = (async () => {
    try {
      const result = sanitizeUIProfileOverrides(await uiProfileAPI.getUIProfile()) || {}
      if (generation !== accountRequestGeneration) return
      const pending = pendingAccountPreferences
      accountPreferences.value = pending ?? result
      accountProfileLoaded = true
      profile.value = applyResolvedProfile()

      if (pending !== undefined) {
        pendingAccountPreferences = undefined
        const saved = sanitizeUIProfileOverrides(await uiProfileAPI.updateUIProfile(pending)) || {}
        if (generation !== accountRequestGeneration) return
        accountPreferences.value = saved
        profile.value = applyResolvedProfile()
      }
    } catch (error) {
      if (generation !== accountRequestGeneration) return
      accountSyncError.value = error instanceof Error ? error.message : 'Failed to load UI preferences'
      console.warn('[ui-platform] account preference hydration failed; using local preferences', error)
    } finally {
      if (generation === accountRequestGeneration) {
        accountSyncing.value = false
        hydrationPromise = null
      }
    }
  })()
  hydrationPromise = request
  return request
}

function clearAccountPreferences() {
  accountRequestGeneration += 1
  hydrationPromise = null
  accountPreferences.value = undefined
  pendingAccountPreferences = undefined
  accountProfileLoaded = false
  accountSyncing.value = false
  accountSyncError.value = ''
  profile.value = applyResolvedProfile()
}

async function updatePreferences(patch: UIProfileOverrides) {
  ensureInitialized()
  const next = sanitizeUIProfileOverrides({ ...toOverrides(profile.value), ...patch }) || {}

  storeLocalPreferences(next)
  accountPreferences.value = accountProfileLoaded ? next : accountPreferences.value
  profile.value = applyResolvedProfile()
  if (!accountProfileLoaded) {
    pendingAccountPreferences = next
    return profile.value
  }

  hydrationPromise = null
  const generation = ++accountRequestGeneration
  accountSyncing.value = true
  accountSyncError.value = ''
  try {
    const result = sanitizeUIProfileOverrides(await uiProfileAPI.updateUIProfile(next)) || {}
    if (generation !== accountRequestGeneration) return profile.value
    accountPreferences.value = result
    profile.value = applyResolvedProfile()
  } catch (error) {
    if (generation !== accountRequestGeneration) return profile.value
    accountSyncError.value = error instanceof Error ? error.message : 'Failed to save UI preferences'
    console.warn('[ui-platform] account preference save failed; local preferences remain active', error)
  } finally {
    if (generation === accountRequestGeneration) accountSyncing.value = false
  }
  return profile.value
}

function previewPreferences(patch: UIProfileOverrides) {
  ensureInitialized()
  profile.value = applyResolvedProfile(sanitizeUIProfileOverrides(patch))
}

async function resetPreferences() {
  localStorage.removeItem(UI_PLATFORM_STORAGE_KEY)
  accountPreferences.value = accountProfileLoaded ? {} : undefined
  profile.value = applyResolvedProfile()
  if (!accountProfileLoaded) {
    pendingAccountPreferences = {}
    return profile.value
  }

  hydrationPromise = null
  const generation = ++accountRequestGeneration
  accountSyncing.value = true
  accountSyncError.value = ''
  try {
    await uiProfileAPI.resetUIProfile()
    if (generation === accountRequestGeneration) accountPreferences.value = {}
  } catch (error) {
    if (generation !== accountRequestGeneration) return profile.value
    accountSyncError.value = error instanceof Error ? error.message : 'Failed to reset UI preferences'
    console.warn('[ui-platform] account preference reset failed', error)
  } finally {
    if (generation === accountRequestGeneration) accountSyncing.value = false
  }
  return profile.value
}

function entries(record: Record<string, UIRegistryEntry>) {
  return Object.values(record)
}

export function useUIPlatform() {
  ensureInitialized()
  return {
    profile: readonly(profile),
    accountSyncing: readonly(accountSyncing),
    accountSyncError: readonly(accountSyncError),
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
    updatePreferences, previewPreferences, resetPreferences,
    hydrateAccountPreferences, clearAccountPreferences, syncFromRuntime,
  }
}
