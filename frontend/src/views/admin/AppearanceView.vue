<template>
  <AppLayout>
    <ReactPageHost
      :load="loadAppearancePage"
      :props="pageProps"
      :error-message="t('common.error')"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ReactPageHost from '@/console/ReactPageHost.vue'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  getRuntimeAppearance,
  resetRuntimeAppearance,
  updateRuntimeAppearance,
} from '@/api/admin/appearance'
import {
  DEFAULT_RUNTIME_APPEARANCE_PRESET_CODE,
  decodeRuntimeAppearance,
} from '@/appearance/preset'
import type { RuntimeAppearancePreview } from '@/appearance/types'
import type { RuntimeAppearanceSettings } from '@/types'
import type { AppearancePageProps } from '@/console/react/AppearancePage'

const { t } = useI18n()
const appStore = useAppStore()
const appearance = ref<RuntimeAppearanceSettings | null>(null)
const preview = ref<RuntimeAppearancePreview | null>(null)
const loading = ref(true)
const saving = ref(false)
const error = ref('')

function productDefaultAppearance(): RuntimeAppearanceSettings {
  return {
    version: 1,
    source: 'shadcn-preset',
    preset_code: DEFAULT_RUNTIME_APPEARANCE_PRESET_CODE,
  }
}

const loadAppearancePage = () => import('@/console/react/AppearancePage')

function previewCode(code: string) {
  try {
    preview.value = decodeRuntimeAppearance(code)
    error.value = ''
  } catch {
    preview.value = null
    error.value = t('appearance.invalidPreset')
  }
}

async function load() {
  loading.value = true
  try {
    appearance.value = await getRuntimeAppearance() || productDefaultAppearance()
    previewCode(appearance.value.preset_code)
  } catch (requestError) {
    appearance.value = productDefaultAppearance()
    previewCode(appearance.value.preset_code)
    error.value = extractApiErrorMessage(requestError, t('appearance.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function save(code: string) {
  try {
    const next = decodeRuntimeAppearance(code)
    preview.value = next
    saving.value = true
    appearance.value = await updateRuntimeAppearance(next.code)
    appStore.patchPublicSettings({ ui_appearance: appearance.value })
    error.value = ''
    appStore.showSuccess(t('appearance.saveSuccess'))
  } catch (requestError) {
    error.value = requestError instanceof Error && requestError.message.includes('preset')
      ? t('appearance.invalidPreset')
      : extractApiErrorMessage(requestError, t('appearance.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function reset() {
  try {
    saving.value = true
    await resetRuntimeAppearance()
    appearance.value = productDefaultAppearance()
    previewCode(appearance.value.preset_code)
    appStore.patchPublicSettings({ ui_appearance: appearance.value })
    error.value = ''
    appStore.showSuccess(t('appearance.resetSuccess'))
  } catch (requestError) {
    error.value = extractApiErrorMessage(requestError, t('appearance.saveFailed'))
  } finally {
    saving.value = false
  }
}

const pageProps = computed<AppearancePageProps>(() => ({
  appearance: appearance.value,
  preview: preview.value,
  loading: loading.value,
  saving: saving.value,
  error: error.value,
  copy: {
    presetCode: t('appearance.presetCode'),
    presetCodeHint: t('appearance.presetCodeHint'),
    preview: t('appearance.preview'),
    save: t('appearance.save'),
    reset: t('appearance.reset'),
    applied: t('appearance.applied'),
    deploymentControlled: t('appearance.deploymentControlled'),
    unavailable: t('appearance.unavailable'),
    previewCard: t('appearance.previewCard'),
    previewDescription: t('appearance.previewDescription'),
    previewButton: t('appearance.previewButton'),
    previewInput: t('appearance.previewInput'),
    activeItem: t('appearance.activeItem'),
    lightMode: t('appearance.lightMode'),
    darkMode: t('appearance.darkMode'),
    previewBrand: t('appearance.previewBrand'),
    previewSettings: t('appearance.previewSettings'),
    noPreset: t('appearance.noPreset'),
    invalidPreset: t('appearance.invalidPreset'),
    controlledStyle: t('appearance.controlledStyle'),
    controlledIcons: t('appearance.controlledIcons'),
  },
  onPreview: previewCode,
  onSave: (code) => { void save(code) },
  onReset: () => { void reset() },
}))

onMounted(() => { void load() })
</script>
