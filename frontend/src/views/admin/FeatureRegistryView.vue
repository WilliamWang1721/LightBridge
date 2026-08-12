<template>
  <AppLayout>
    <ReactPageHost
      :load="loadFeatureRegistryPage"
      :props="pageProps"
      :error-message="t('common.error')"
    >
      <template #fallback>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('featureRegistry.title') }}</h1>
          <p class="mt-1 max-w-3xl text-sm text-gray-500 dark:text-dark-400">{{ t('featureRegistry.description') }}</p>
        </div>
        <button class="btn btn-secondary w-fit" :disabled="loading" @click="loadFeatures">
          <Icon name="refresh" size="sm" :stroke-width="2" :class="{ 'animate-spin': loading }" />
          {{ t('featureRegistry.refresh') }}
        </button>
      </div>

      <div class="flex flex-wrap items-center gap-2 text-sm">
        <span v-if="profile" class="rounded-full bg-primary-50 px-3 py-1 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
          {{ t('featureRegistry.profile', { profile }) }}
        </span>
        <span class="rounded-full bg-gray-100 px-3 py-1 text-gray-600 dark:bg-dark-700 dark:text-dark-300">
          {{ t('featureRegistry.featureCount', { count: features.length }) }}
        </span>
      </div>

      <div v-if="error" class="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-200">
        <Icon name="xCircle" size="md" :stroke-width="2" class="mt-0.5 shrink-0" />
        <p class="whitespace-pre-line break-words">{{ error }}</p>
      </div>

      <div v-if="!loading && features.length === 0 && !error" class="card p-6 text-center">
        <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('featureRegistry.noFeatures') }}</p>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('featureRegistry.noFeaturesHint') }}</p>
      </div>

      <section v-for="group in featureGroups" :key="group.id" class="space-y-3">
        <div class="flex items-center gap-3">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ tierLabel(group.id) }}</h2>
          <span class="rounded-full bg-gray-100 px-2.5 py-1 text-xs text-gray-600 dark:bg-dark-700 dark:text-dark-300">{{ group.features.length }}</span>
        </div>

        <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <article v-for="feature in group.features" :key="feature.id" class="card p-5" :data-test="`feature-${feature.id}`">
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <h3 class="truncate font-semibold text-gray-900 dark:text-white">{{ feature.label || feature.id }}</h3>
                  <span class="rounded-full px-2 py-0.5 text-xs font-medium" :class="enabledClass(feature)">
                    {{ feature.enabled ? t('featureRegistry.enabled') : t('featureRegistry.disabled') }}
                  </span>
                  <span v-if="feature.requiresRestart" class="rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
                    {{ t('featureRegistry.restartRequired') }}
                  </span>
                </div>
                <p class="mt-1 font-mono text-xs text-gray-500 dark:text-dark-400">{{ feature.id }}</p>
              </div>

              <Toggle
                :model-value="feature.configuredEnabled"
                :disabled="isBusy(feature.id) || feature.controllable !== true"
                :aria-label="feature.label || feature.id"
                @update:model-value="setEnabled(feature, $event)"
              />
            </div>

            <div class="mt-4 flex flex-wrap gap-2 text-xs">
              <span class="rounded bg-gray-100 px-2 py-1 text-gray-600 dark:bg-dark-700 dark:text-dark-300">{{ activationLabel(feature.activation) }}</span>
              <span class="rounded bg-gray-100 px-2 py-1 text-gray-600 dark:bg-dark-700 dark:text-dark-300">{{ t('featureRegistry.minimumProfile', { profile: feature.minimumProfile || '—' }) }}</span>
              <span class="rounded px-2 py-1" :class="feature.controllable === true ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'">
                {{ feature.controllable === true ? t('featureRegistry.configurable') : t('featureRegistry.notConfigurable') }}
              </span>
              <span v-if="feature.available === false" class="rounded bg-red-50 px-2 py-1 text-red-700 dark:bg-red-900/30 dark:text-red-200">{{ t('featureRegistry.unavailable') }}</span>
              <span v-if="feature.override !== null && feature.override !== undefined" class="rounded bg-violet-50 px-2 py-1 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300">
                {{ feature.override ? t('featureRegistry.overrideEnabled') : t('featureRegistry.overrideDisabled') }}
              </span>
            </div>

            <dl class="mt-4 grid gap-3 border-t border-gray-100 pt-4 text-sm dark:border-dark-700">
              <div class="flex flex-wrap gap-x-2 gap-y-1">
                <dt class="text-gray-500 dark:text-dark-400">{{ t('featureRegistry.configuredState', { state: feature.configuredEnabled ? t('featureRegistry.enabled') : t('featureRegistry.disabled') }) }}</dt>
                <dd class="text-gray-700 dark:text-dark-200">· {{ t('featureRegistry.runtimeState', { state: feature.enabled ? t('featureRegistry.enabled') : t('featureRegistry.disabled') }) }}</dd>
              </div>
              <div v-if="feature.reason" class="text-gray-600 dark:text-dark-300">
                {{ t('featureRegistry.reason', { reason: reasonLabel(feature.reason) }) }}
              </div>
              <div>
                <dt class="mb-1 text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">{{ t('featureRegistry.dependencies') }}</dt>
                <dd class="flex flex-wrap gap-1.5">
                  <span v-if="!feature.dependencies?.length" class="text-sm text-gray-500 dark:text-dark-400">{{ t('featureRegistry.noDependencies') }}</span>
                  <span v-for="dependency in feature.dependencies" :key="dependency" class="rounded bg-gray-100 px-2 py-1 font-mono text-xs text-gray-700 dark:bg-dark-700 dark:text-dark-200">{{ dependency }}</span>
                </dd>
              </div>
              <div v-if="feature.surfaces?.length">
                <dt class="mb-1 text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">{{ t('featureRegistry.surfaces') }}</dt>
                <dd class="flex flex-wrap gap-1.5">
                  <span v-for="surface in feature.surfaces" :key="surface" class="rounded bg-gray-100 px-2 py-1 font-mono text-xs text-gray-700 dark:bg-dark-700 dark:text-dark-200">{{ surface }}</span>
                </dd>
              </div>
              <div>
                <dt class="mb-1 text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-dark-400">{{ t('featureRegistry.runtimeComponents') }}</dt>
                <dd>
                  <span v-if="!feature.runtimeComponents?.length" class="text-sm text-gray-500 dark:text-dark-400">{{ t('featureRegistry.noRuntimeComponents') }}</span>
                  <div v-else class="space-y-1.5">
                    <div v-for="component in feature.runtimeComponents" :key="runtimeComponentKey(component)" class="rounded bg-gray-100 px-2 py-1.5 dark:bg-dark-700">
                      <span class="font-mono text-xs" :class="runtimeComponentStatusClass(component)">{{ runtimeComponentLabel(component) }}</span>
                      <p v-if="component.lastError" class="mt-1 break-words text-xs text-red-700 dark:text-red-300" :data-test="`runtime-error-${feature.id}`">
                        {{ component.lastError }}
                      </p>
                    </div>
                  </div>
                </dd>
              </div>
            </dl>

            <div v-if="feature.override !== null && feature.override !== undefined" class="mt-4 flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700">
              <button class="btn btn-secondary btn-sm" :disabled="isBusy(feature.id)" @click="restoreDefault(feature)">
                {{ t('featureRegistry.restoreDefault') }}
              </button>
            </div>
          </article>
        </div>
      </section>
    </div>
      </template>
    </ReactPageHost>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ReactPageHost from '@/console/ReactPageHost.vue'
import type { FeatureRegistryPageProps } from '@/console/react/FeatureRegistryPage'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import {
  clearFeatureControlOverride,
  listFeatureControls,
  setFeatureControl,
  type FeatureControlState,
  type FeatureRuntimeComponent,
} from '@/api/admin/features'
import { extractApiErrorMessage } from '@/utils/apiError'

type FeatureGroupID = 'core' | 'optional' | 'extension' | 'other'

const { t } = useI18n()
const loading = ref(false)
const error = ref('')
const profile = ref<string | undefined>()
const features = ref<FeatureControlState[]>([])
const busyFeatureID = ref('')

const loadFeatureRegistryPage = () => import('@/console/react/FeatureRegistryPage')

const featureGroups = computed(() => {
  const buckets = new Map<FeatureGroupID, FeatureControlState[]>([
    ['core', []],
    ['optional', []],
    ['extension', []],
    ['other', []],
  ])
  for (const feature of features.value) {
    const tier: FeatureGroupID = feature.tier === 'core' || feature.tier === 'optional' || feature.tier === 'extension'
      ? feature.tier
      : 'other'
    buckets.get(tier)?.push(feature)
  }
  return Array.from(buckets.entries())
    .filter(([, groupFeatures]) => groupFeatures.length > 0)
    .map(([id, groupFeatures]) => ({
      id,
      features: groupFeatures.slice().sort((left, right) => left.label.localeCompare(right.label) || left.id.localeCompare(right.id)),
    }))
})

const pageProps = computed<FeatureRegistryPageProps>(() => ({
  loading: loading.value,
  error: error.value,
  profile: profile.value,
  groups: featureGroups.value,
  busyFeatureID: busyFeatureID.value,
  copy: {
    refresh: t('featureRegistry.refresh'),
    profile: (value) => t('featureRegistry.profile', { profile: value }),
    featureCount: (count) => t('featureRegistry.featureCount', { count }),
    enabled: t('featureRegistry.enabled'),
    disabled: t('featureRegistry.disabled'),
    restartRequired: t('featureRegistry.restartRequired'),
    configurable: t('featureRegistry.configurable'),
    notConfigurable: t('featureRegistry.notConfigurable'),
    unavailable: t('featureRegistry.unavailable'),
    overrideEnabled: t('featureRegistry.overrideEnabled'),
    overrideDisabled: t('featureRegistry.overrideDisabled'),
    minimumProfile: (value) => t('featureRegistry.minimumProfile', { profile: value }),
    configuredState: (state) => t('featureRegistry.configuredState', { state }),
    runtimeState: (state) => t('featureRegistry.runtimeState', { state }),
    reason: (value) => t('featureRegistry.reason', { reason: value }),
    dependencies: t('featureRegistry.dependencies'),
    noDependencies: t('featureRegistry.noDependencies'),
    surfaces: t('featureRegistry.surfaces'),
    runtimeComponents: t('featureRegistry.runtimeComponents'),
    noRuntimeComponents: t('featureRegistry.noRuntimeComponents'),
    restoreDefault: t('featureRegistry.restoreDefault'),
    noFeatures: t('featureRegistry.noFeatures'),
    noFeaturesHint: t('featureRegistry.noFeaturesHint'),
    tierLabel,
    activationLabel,
    reasonLabel,
  },
  onRefresh: () => { void loadFeatures() },
  onSetEnabled: (feature, enabled) => { void setEnabled(feature, enabled) },
  onRestoreDefault: (feature) => { void restoreDefault(feature) },
}))

function applyOverview(next: { features: FeatureControlState[]; profile?: string }) {
  features.value = next.features
  profile.value = next.profile
}

function isBusy(id: string): boolean {
  return busyFeatureID.value === id
}

function messageOf(err: unknown): string {
  return extractApiErrorMessage(err, t('featureRegistry.operationFailed'))
}

async function loadFeatures() {
  loading.value = true
  error.value = ''
  try {
    const result = await listFeatureControls()
    features.value = result.features
    profile.value = result.profile
  } catch (err) {
    features.value = []
    profile.value = undefined
    error.value = messageOf(err)
  } finally {
    loading.value = false
  }
}

async function setEnabled(feature: FeatureControlState, enabled: boolean) {
  if (feature.controllable !== true || isBusy(feature.id)) return
  busyFeatureID.value = feature.id
  error.value = ''
  try {
    applyOverview(await setFeatureControl(feature.id, enabled))
  } catch (err) {
    error.value = messageOf(err)
  } finally {
    busyFeatureID.value = ''
  }
}

async function restoreDefault(feature: FeatureControlState) {
  if (isBusy(feature.id)) return
  busyFeatureID.value = feature.id
  error.value = ''
  try {
    applyOverview(await clearFeatureControlOverride(feature.id))
  } catch (err) {
    error.value = messageOf(err)
  } finally {
    busyFeatureID.value = ''
  }
}

function tierLabel(tier: FeatureGroupID): string {
  return t(`featureRegistry.tiers.${tier}`)
}

function activationLabel(activation: string): string {
  const key = `featureRegistry.activation.${activation}`
  const translated = t(key)
  return translated === key ? activation : translated
}

function reasonLabel(reason: string): string {
  const key = `featureRegistry.reasons.${reason}`
  const translated = t(key)
  return translated === key ? reason : translated
}

function enabledClass(feature: FeatureControlState): string {
  return feature.enabled
    ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
    : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
}

function runtimeComponentLabel(component: FeatureRuntimeComponent): string {
  const status = component.status || (component.running ? 'running' : component.started ? 'started' : '')
  return [component.name || component.feature || JSON.stringify(component), status].filter(Boolean).join(' · ')
}

function runtimeComponentKey(component: FeatureRuntimeComponent): string {
  return `${component.name || ''}:${component.feature || ''}:${component.status || ''}:${component.updatedAt || ''}`
}

function runtimeComponentStatusClass(component: FeatureRuntimeComponent): string {
  return component.status === 'error' || component.lastError
    ? 'text-red-700 dark:text-red-300'
    : 'text-gray-700 dark:text-dark-200'
}

onMounted(loadFeatures)
</script>
