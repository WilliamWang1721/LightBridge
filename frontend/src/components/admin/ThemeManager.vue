<template>
  <div class="space-y-6">
    <UIPlatformSettings :packages="activePackages" />

    <div class="card">
      <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('admin.settings.uiThemes.title') }}
        </h2>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.uiThemes.description') }}
        </p>
      </div>
      <div class="grid gap-4 p-6 lg:grid-cols-2">
        <div class="space-y-3">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.settings.uiThemes.githubUrl') }}
          </label>
          <div class="flex flex-col gap-2 sm:flex-row">
            <input v-model.trim="githubUrl" type="url" class="input flex-1" placeholder="https://github.com/org/theme">
            <button type="button" class="btn btn-primary" :disabled="busy || !githubUrl" @click="importFromGitHub">
              <Icon name="download" size="sm" class="mr-1.5" />
              {{ t('admin.settings.uiThemes.import') }}
            </button>
          </div>
        </div>
        <div class="space-y-3">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.settings.uiThemes.zipUpload') }}
          </label>
          <div class="flex flex-col gap-2 sm:flex-row">
            <input ref="fileInput" type="file" accept=".zip" class="input flex-1" @change="onFileChange">
            <button type="button" class="btn btn-secondary" :disabled="busy || !selectedFile" @click="uploadSelected">
              <Icon name="upload" size="sm" class="mr-1.5" />
              {{ t('admin.settings.uiThemes.upload') }}
            </button>
          </div>
        </div>
        <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
          <input v-model="replaceExisting" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600">
          {{ t('admin.settings.uiThemes.replaceExisting') }}
        </label>
      </div>
    </div>

    <div v-if="error" class="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/20 dark:text-red-300">
      {{ error }}
    </div>

    <div class="card overflow-hidden">
      <div class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-700">
        <div>
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">
            {{ t('admin.settings.uiThemes.installed') }}
          </h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ copy.installedHint }}</p>
        </div>
        <button type="button" class="btn btn-secondary btn-sm" :disabled="busy" @click="loadThemes">
          <Icon name="refresh" size="sm" class="mr-1.5" />
          {{ t('common.refresh') }}
        </button>
      </div>
      <div v-if="loading" class="flex justify-center py-10">
        <div class="h-7 w-7 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>
      <div v-else-if="themes.length === 0" class="px-6 py-10 text-center text-sm text-gray-500">
        {{ t('admin.settings.uiThemes.empty') }}
      </div>
      <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
        <article v-for="theme in themes" :key="theme.id" class="grid gap-5 px-6 py-6 xl:grid-cols-[240px_minmax(0,1fr)]">
          <div class="space-y-3">
            <div class="aspect-[4/3] overflow-hidden rounded-2xl border border-gray-200 bg-gray-100 dark:border-dark-700 dark:bg-dark-900">
              <img
                v-if="previewURL(theme)"
                :src="previewURL(theme)"
                :alt="`${theme.name} preview`"
                class="h-full w-full object-cover"
                loading="lazy"
              >
              <div v-else class="flex h-full items-center justify-center px-4 text-center text-sm text-gray-400">
                {{ copy.noPreview }}
              </div>
            </div>

            <div class="space-y-1">
              <div class="flex flex-wrap items-center gap-2">
                <h4 class="font-semibold text-gray-900 dark:text-white">{{ theme.name }}</h4>
                <span v-if="theme.active" class="badge badge-success">
                  {{ t('admin.settings.uiThemes.active') }}
                </span>
              </div>
              <code class="inline-flex rounded bg-gray-100 px-2 py-0.5 text-xs dark:bg-dark-700">{{ theme.id }}</code>
              <p class="text-sm text-gray-500 dark:text-gray-400">v{{ theme.version }}</p>
              <p class="break-all text-xs text-gray-400 dark:text-gray-500">{{ theme.source || '-' }}</p>
            </div>
          </div>

          <div class="min-w-0 space-y-5">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h5 class="text-sm font-semibold text-gray-900 dark:text-white">{{ copy.packageParameters }}</h5>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ copy.packageParametersHint }}</p>
              </div>
              <div class="flex flex-wrap gap-2">
                <button
                  type="button"
                  class="btn btn-primary btn-sm"
                  :disabled="busy || theme.active"
                  @click="activate(theme.id)"
                >
                  {{ t('admin.settings.uiThemes.activate') }}
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="busy || !theme.active"
                  @click="deactivate(theme.id)"
                >
                  {{ t('admin.settings.uiThemes.deactivate') }}
                </button>
                <button type="button" class="btn btn-secondary btn-sm" :disabled="busy" @click="resetConfig(theme)">
                  {{ copy.restorePackageDefaults }}
                </button>
                <button type="button" class="btn btn-secondary btn-sm" :disabled="busy" @click="saveConfig(theme)">
                  {{ t('common.save') }}
                </button>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm text-red-600 hover:text-red-700 dark:text-red-400"
                  :disabled="busy"
                  @click="remove(theme.id)"
                >
                  {{ t('common.delete') }}
                </button>
              </div>
            </div>

            <ThemeConfigForm
              :theme-id="theme.id"
              :fields="themeFields(theme)"
              :model-value="configDrafts[theme.id] || {}"
              :errors="configErrors[theme.id] || {}"
              :empty-label="copy.noParameters"
              :on-label="copy.enabled"
              :off-label="copy.disabled"
              @update:model-value="configDrafts[theme.id] = $event"
            />
          </div>
        </article>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import UIPlatformSettings from '@/components/settings/UIPlatformSettings.vue'
import ThemeConfigForm from '@/components/admin/ui/ThemeConfigForm.vue'
import uiThemesAPI, { type UITheme } from '@/api/admin/uiThemes'
import { useUIPlatform } from '@/composables/useUIPlatform'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  buildThemeConfigDraft,
  getThemeConfigFields,
  resetThemeConfig,
  themeAssetURL,
  validateThemeConfig,
} from '@/ui-platform/themePackages'

const { t, locale } = useI18n()
const { profile, updatePreferences } = useUIPlatform()
const themes = ref<UITheme[]>([])
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const githubUrl = ref('')
const selectedFile = ref<File | null>(null)
const replaceExisting = ref(false)
const configDrafts = reactive<Record<string, Record<string, unknown>>>({})
const configErrors = reactive<Record<string, Record<string, string>>>({})

const copy = computed(() => {
  const zh = locale.value.toLowerCase().startsWith('zh')
  return zh
    ? {
        installedHint: 'UI 包继续使用原有 ZIP/GitHub 加载机制，参数由清单自动生成，不再要求手写 JSON。',
        noPreview: '此 UI 包未提供预览图',
        packageParameters: 'UI 包参数',
        packageParametersHint: '只显示 UI 包在 lightbridge-ui.json 中声明的安全参数。组件风格固定为 Luma。',
        restorePackageDefaults: '恢复包默认值',
        noParameters: '此 UI 包没有提供可调整参数。',
        enabled: '启用',
        disabled: '停用',
      }
    : {
        installedHint: 'UI packages keep the existing ZIP/GitHub loader. Settings are generated from the manifest instead of requiring raw JSON.',
        noPreview: 'This UI package does not provide a preview image.',
        packageParameters: 'UI package parameters',
        packageParametersHint: 'Only safe parameters declared in lightbridge-ui.json are shown. The component style remains fixed to Luma.',
        restorePackageDefaults: 'Restore package defaults',
        noParameters: 'This UI package does not provide configurable parameters.',
        enabled: 'Enabled',
        disabled: 'Disabled',
      }
})

const activePackages = computed(() =>
  themes.value.filter((theme) => theme.active).map((theme) => ({ id: theme.id, name: theme.name })),
)

function setThemes(items: UITheme[]) {
  themes.value = items
  const installedIds = new Set(items.map((theme) => theme.id))

  for (const key of Object.keys(configDrafts)) {
    if (!installedIds.has(key)) delete configDrafts[key]
  }
  for (const key of Object.keys(configErrors)) {
    if (!installedIds.has(key)) delete configErrors[key]
  }

  for (const theme of items) {
    configDrafts[theme.id] = buildThemeConfigDraft(theme.manifest, theme.config)
    configErrors[theme.id] = {}
  }
}

async function loadThemes() {
  loading.value = true
  error.value = ''
  try {
    setThemes(await uiThemesAPI.listThemes())
  } catch (err) {
    error.value = extractApiErrorMessage(err)
  } finally {
    loading.value = false
  }
}

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  selectedFile.value = input.files?.[0] ?? null
}

async function run(operation: () => Promise<unknown>) {
  busy.value = true
  error.value = ''
  try {
    await operation()
    await loadThemes()
  } catch (err) {
    error.value = extractApiErrorMessage(err)
  } finally {
    busy.value = false
  }
}

function importFromGitHub() {
  return run(() => uiThemesAPI.importGitHubTheme(githubUrl.value, replaceExisting.value))
}

function uploadSelected() {
  if (!selectedFile.value) return
  return run(() => uiThemesAPI.uploadTheme(selectedFile.value, replaceExisting.value))
}

function activate(id: string) {
  return run(async () => {
    await uiThemesAPI.activateTheme(id)
    updatePreferences({ mode: 'package', activePackageId: id })
  })
}

function deactivate(id: string) {
  return run(async () => {
    await uiThemesAPI.deactivateTheme(id)
    if (profile.value.activePackageId === id) updatePreferences({ mode: 'modern' })
  })
}

function saveConfig(theme: UITheme) {
  const validation = validateThemeConfig(theme.manifest, configDrafts[theme.id] || {})
  configErrors[theme.id] = validation.errors
  if (Object.keys(validation.errors).length) return
  return run(() => uiThemesAPI.updateThemeConfig(theme.id, validation.value))
}

function resetConfig(theme: UITheme) {
  configDrafts[theme.id] = resetThemeConfig(theme.manifest)
  configErrors[theme.id] = {}
}

function remove(id: string) {
  return run(async () => {
    await uiThemesAPI.deleteTheme(id)
    if (profile.value.activePackageId === id) updatePreferences({ mode: 'modern' })
  })
}

function themeFields(theme: UITheme) {
  return getThemeConfigFields(theme.manifest)
}

function previewURL(theme: UITheme) {
  return themeAssetURL(theme.id, theme.preview || theme.manifest.preview)
}

onMounted(loadThemes)
</script>
