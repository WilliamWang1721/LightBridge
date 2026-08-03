<template>
  <AppLayout>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-end">
      <button
        type="button"
        class="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 dark:hover:bg-dark-700"
        :disabled="loading"
        @click="loadReleases(true)"
      >
        <Icon name="refresh" size="sm" :stroke-width="2" :class="{ 'animate-spin': loading }" />
        {{ t('version.refresh') }}
      </button>
    </div>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('version.currentVersion') }}</p>
        <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
          {{ displayVersion(currentVersion) }}
        </p>
      </div>
      <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('version.latestVersion') }}</p>
        <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
          {{ displayVersion(latestVersion) }}
        </p>
      </div>
      <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('version.buildType') }}</p>
        <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
          {{ isReleaseBuild ? t('version.releaseBuild') : t('version.sourceMode') }}
        </p>
      </div>
      <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('version.versionType') }}</p>
        <div class="mt-2 inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-600 dark:bg-dark-700">
          <button
            type="button"
            :class="[
              'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
              versionType === 'production'
                ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                : 'text-gray-500 hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-200'
            ]"
            @click="versionType = 'production'"
          >
            {{ t('version.production') }}
          </button>
          <button
            type="button"
            :class="[
              'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
              versionType === 'preview'
                ? 'bg-white text-amber-700 shadow-sm dark:bg-dark-800 dark:text-amber-300'
                : 'text-gray-500 hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-200'
            ]"
            @click="versionType = 'preview'"
          >
            {{ t('version.preview') }}
          </button>
        </div>
      </div>
    </div>

    <div
      v-if="isContainerDeployment"
      class="flex items-start gap-3 rounded-xl border border-blue-200 bg-blue-50 p-4 dark:border-blue-800/50 dark:bg-blue-900/20"
    >
      <Icon
        name="infoCircle"
        size="md"
        :stroke-width="2"
        class="mt-0.5 flex-shrink-0 text-blue-600 dark:text-blue-400"
      />
      <div>
        <p class="text-sm font-medium text-blue-800 dark:text-blue-200">
          {{ t('version.containerUpgradeTitle') }}
        </p>
        <p class="mt-1 text-sm text-blue-700/80 dark:text-blue-300/80">
          {{ t('version.containerUpgradeHint') }}
        </p>
        <div class="mt-3 flex flex-wrap gap-2 text-xs">
          <code class="rounded bg-blue-100 px-2 py-1 font-mono text-blue-800 dark:bg-blue-900/50 dark:text-blue-100">{{ t('version.containerUpgradePull') }}</code>
          <code class="rounded bg-blue-100 px-2 py-1 font-mono text-blue-800 dark:bg-blue-900/50 dark:text-blue-100">{{ t('version.containerUpgradeUp') }}</code>
        </div>
      </div>
    </div>

    <div
      v-else-if="buildType && !isReleaseBuild"
      class="flex items-start gap-3 rounded-xl border border-blue-200 bg-blue-50 p-4 dark:border-blue-800/50 dark:bg-blue-900/20"
    >
      <Icon
        name="infoCircle"
        size="md"
        :stroke-width="2"
        class="mt-0.5 flex-shrink-0 text-blue-600 dark:text-blue-400"
      />
      <div>
        <p class="text-sm font-medium text-blue-800 dark:text-blue-200">
          {{ t('version.sourceBuildInstallDisabled') }}
        </p>
        <p class="mt-1 text-sm text-blue-700/80 dark:text-blue-300/80">
          {{ t('version.sourceModeHint') }}
        </p>
      </div>
    </div>

    <div
      v-if="updateError"
      class="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20"
    >
      <Icon
        name="xCircle"
        size="md"
        :stroke-width="2"
        class="mt-0.5 flex-shrink-0 text-red-600 dark:text-red-400"
      />
      <div class="min-w-0">
        <p class="text-sm font-medium text-red-800 dark:text-red-200">
          {{ t('version.updateFailed') }}
        </p>
        <p class="mt-1 break-words text-sm text-red-700/80 dark:text-red-300/80">
          {{ updateError }}
        </p>
      </div>
    </div>

    <div
      v-if="updateSuccess"
      class="flex items-start gap-3 rounded-xl border border-green-200 bg-green-50 p-4 dark:border-green-800/50 dark:bg-green-900/20"
    >
      <Icon
        name="checkCircle"
        size="md"
        :stroke-width="2"
        class="mt-0.5 flex-shrink-0 text-green-600 dark:text-green-400"
      />
      <div class="min-w-0 flex-1">
        <p class="text-sm font-medium text-green-800 dark:text-green-200">
          {{ t('version.updateComplete') }}
        </p>
        <p class="mt-1 text-sm text-green-700/80 dark:text-green-300/80">
          {{ t('version.restartRequired') }}
        </p>
      </div>
      <button
        v-if="needRestart && canRestart"
        type="button"
        class="inline-flex items-center gap-2 rounded-lg bg-green-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-green-700 disabled:cursor-not-allowed disabled:opacity-60"
        :disabled="restarting"
        @click="handleRestart"
      >
        <Icon name="refresh" size="sm" :stroke-width="2" :class="{ 'animate-spin': restarting }" />
        <span>
          {{ restarting ? t('version.restarting') : t('version.restartNow') }}
          <span v-if="restarting && restartCountdown > 0">({{ restartCountdown }}s)</span>
        </span>
      </button>
    </div>

    <div class="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
      <div v-if="loading" class="flex items-center justify-center py-16">
        <Icon name="refresh" size="lg" :stroke-width="2" class="animate-spin text-primary-500" />
      </div>

      <div v-else-if="loadError" class="px-5 py-10 text-center">
        <Icon name="exclamationCircle" size="xl" :stroke-width="2" class="mx-auto text-red-500" />
        <p class="mt-3 text-sm font-medium text-gray-900 dark:text-white">
          {{ t('version.loadVersionsFailed') }}
        </p>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ loadError }}</p>
      </div>

      <div v-else-if="publishedReleases.length === 0" class="px-5 py-10 text-center">
        <Icon name="inbox" size="xl" :stroke-width="2" class="mx-auto text-gray-400" />
        <p class="mt-3 text-sm font-medium text-gray-900 dark:text-white">
          {{ t('version.noVersionsOfType') }}
        </p>
      </div>

      <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
        <div
          v-for="release in publishedReleases"
          :key="release.version"
          class="grid gap-4 px-5 py-4 md:grid-cols-[1fr_auto] md:items-center"
        >
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">
                {{ displayVersion(release.version) }}
              </h3>
              <span
                v-if="isSameVersion(release.version, currentVersion)"
                class="rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700 dark:bg-green-900/30 dark:text-green-300"
              >
                {{ t('version.current') }}
              </span>
              <span
                v-if="isSameVersion(release.version, latestVersion)"
                class="rounded-full bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
              >
                {{ t('version.latest') }}
              </span>
              <span
                v-if="release.prerelease"
                class="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
              >
                {{ t('version.preview') }}
              </span>
              <span
                v-else
                class="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
              >
                {{ t('version.production') }}
              </span>
            </div>
            <div class="mt-2 flex flex-wrap items-center gap-3 text-xs text-gray-500 dark:text-dark-400">
              <span>{{ formatDate(release.published_at) }}</span>
              <a
                v-if="release.html_url && release.html_url !== '#'"
                :href="release.html_url"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex items-center gap-1 text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
              >
                {{ t('version.viewRelease') }}
                <Icon name="externalLink" size="xs" :stroke-width="2" />
              </a>
            </div>
          </div>

          <div class="flex items-center gap-2 md:justify-end">
            <button
              v-if="canRestart"
              type="button"
              class="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 dark:hover:bg-dark-700"
              @click="showUpgradeChanges(release)"
            >
              <Icon name="infoCircle" size="sm" :stroke-width="2" />
              {{ t('version.viewUpgradeChanges') }}
            </button>
            <button
              v-if="canInPlaceUpdate"
              type="button"
              class="inline-flex items-center justify-center gap-2 rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50"
              :disabled="updating || restarting || isSameVersion(release.version, currentVersion)"
              @click="confirmInstall(release)"
            >
              <Icon name="download" size="sm" :stroke-width="2" />
              {{
                isSameVersion(release.version, currentVersion)
                  ? t('version.installed')
                  : t('version.installVersion')
              }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <ConfirmDialog
      v-if="canInPlaceUpdate"
      :show="confirmDialogOpen"
      :title="t('version.installConfirmTitle')"
      :message="installConfirmMessage"
      :confirm-text="t('version.installVersion')"
      :cancel-text="t('common.cancel')"
      @confirm="handleInstall"
      @cancel="confirmDialogOpen = false"
    >
      <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800/50 dark:bg-amber-900/20">
        <p class="text-sm text-amber-800 dark:text-amber-200">
          {{ t('version.dataSafeHint') }}
        </p>
      </div>
    </ConfirmDialog>

    <UpgradeChangesDialog
      v-if="canRestart"
      :show="upgradeChangesOpen"
      :version="upgradeChangesRelease?.version"
      :body="upgradeChangesRelease?.body"
      :html-url="upgradeChangesRelease?.html_url"
      :can-upgrade="canInPlaceUpdate && !isSameVersion(upgradeChangesRelease?.version, currentVersion)"
      :upgrading="updating"
      :restarting="restarting"
      @close="upgradeChangesOpen = false"
      @upgrade="handleUpgradeFromDialog"
      @restart="handleRestart"
    />
  </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  listVersionReleases,
  performUpdate,
  restartService,
  type UpdateCapabilities,
  type VersionRelease
} from '@/api/admin/system'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import UpgradeChangesDialog from '@/components/common/UpgradeChangesDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const loadError = ref('')
const releases = ref<VersionRelease[]>([])
const currentVersion = ref('')
const latestVersion = ref('')
const buildType = ref('')
const updateCapabilities = ref<UpdateCapabilities>({
  deployment_type: 'unknown',
  can_in_place_update: false,
  can_rollback: false,
  can_restart: false
})
const updating = ref(false)
const updateError = ref('')
const updateSuccess = ref(false)
const needRestart = ref(false)
const restarting = ref(false)
const restartCountdown = ref(0)
const confirmDialogOpen = ref(false)
const selectedRelease = ref<VersionRelease | null>(null)
const upgradeChangesRelease = ref<VersionRelease | null>(null)
const upgradeChangesOpen = ref(false)
const versionType = ref<'production' | 'preview'>('production')

const isReleaseBuild = computed(() => buildType.value === 'release')
const isContainerDeployment = computed(() => updateCapabilities.value.deployment_type === 'container')
const canInPlaceUpdate = computed(
  () => isReleaseBuild.value && updateCapabilities.value.can_in_place_update
)
const canRestart = computed(() => updateCapabilities.value.can_restart)
const publishedReleases = computed(() =>
  releases.value.filter((release) => {
    if (release.draft) return false
    return versionType.value === 'preview' ? !!release.prerelease : !release.prerelease
  })
)
const installConfirmMessage = computed(() => {
  const version = selectedRelease.value?.version ? displayVersion(selectedRelease.value.version) : ''
  return t('version.installConfirmMessage', { version })
})

function normalizeVersion(version?: string): string {
  return String(version || '').trim().replace(/^v/i, '')
}

function isSameVersion(left?: string, right?: string): boolean {
  return normalizeVersion(left) === normalizeVersion(right)
}

function displayVersion(version?: string): string {
  const normalized = normalizeVersion(version)
  return normalized ? `v${normalized}` : '--'
}

function formatDate(value?: string): string {
  if (!value) return t('common.notAvailable')
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

function getErrorMessage(error: unknown, fallback: string): string {
  const err = error as {
    response?: { data?: { message?: string } }
    message?: string
  }
  return err.response?.data?.message || err.message || fallback
}

async function loadReleases(force = false) {
  loading.value = true
  loadError.value = ''
  try {
    const [versionInfo, releaseData] = await Promise.all([
      appStore.fetchVersion(force),
      listVersionReleases(force)
    ])

    currentVersion.value = releaseData.current_version || versionInfo?.current_version || appStore.currentVersion
    latestVersion.value = releaseData.latest_version || versionInfo?.latest_version || appStore.latestVersion
    buildType.value = releaseData.build_type || versionInfo?.build_type || appStore.buildType
    updateCapabilities.value = releaseData.capabilities || versionInfo?.capabilities || appStore.updateCapabilities
    releases.value = releaseData.releases || []
  } catch (error) {
    loadError.value = getErrorMessage(error, t('version.loadVersionsFailed'))
    currentVersion.value = appStore.currentVersion
    latestVersion.value = appStore.latestVersion
    buildType.value = appStore.buildType
    updateCapabilities.value = appStore.updateCapabilities
  } finally {
    loading.value = false
  }
}

function confirmInstall(release: VersionRelease) {
  if (!canInPlaceUpdate.value) return
  selectedRelease.value = release
  confirmDialogOpen.value = true
}

function showUpgradeChanges(release: VersionRelease) {
  if (!canRestart.value) return
  upgradeChangesRelease.value = release
  upgradeChangesOpen.value = true
}

function handleUpgradeFromDialog() {
  if (!upgradeChangesRelease.value) return
  const release = upgradeChangesRelease.value
  upgradeChangesOpen.value = false
  confirmInstall(release)
}

async function handleInstall() {
  if (!selectedRelease.value || !canInPlaceUpdate.value || updating.value) return

  confirmDialogOpen.value = false
  updating.value = true
  updateError.value = ''
  updateSuccess.value = false
  needRestart.value = false

  try {
    const release = selectedRelease.value
    const result = await performUpdate({ version: selectedRelease.value.version })
    updateSuccess.value = true
    needRestart.value = result.need_restart
    upgradeChangesRelease.value = release
    upgradeChangesOpen.value = true
    appStore.clearVersionCache()
    try {
      await loadReleases(true)
    } catch {
      // The install already succeeded; a follow-up refresh failure should not be shown as install failure.
    }
  } catch (error) {
    updateError.value = getErrorMessage(error, t('version.updateFailed'))
  } finally {
    updating.value = false
  }
}

async function handleRestart() {
  if (!canRestart.value || restarting.value) return

  restarting.value = true
  restartCountdown.value = 8

  try {
    await restartService()
  } catch {
    // The restart request may drop the connection before a response is returned.
  }

  const countdownInterval = window.setInterval(() => {
    restartCountdown.value--
    if (restartCountdown.value <= 0) {
      window.clearInterval(countdownInterval)
      checkServiceAndReload()
    }
  }, 1000)
}

async function checkServiceAndReload() {
  for (let i = 0; i < 5; i++) {
    try {
      const response = await fetch('/health', { method: 'GET', cache: 'no-cache' })
      if (response.ok) {
        window.location.reload()
        return
      }
    } catch {
      // Service is still restarting.
    }
    await new Promise((resolve) => setTimeout(resolve, 1000))
  }

  window.location.reload()
}

onMounted(() => {
  loadReleases(false)
})
</script>
