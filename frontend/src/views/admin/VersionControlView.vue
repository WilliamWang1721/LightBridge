<template>
  <AppLayout>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-end">
      <button
        v-if="canRollback"
        type="button"
        data-test="rollback-version"
        class="inline-flex items-center justify-center gap-2 rounded-lg border border-red-200 bg-white px-4 py-2 text-sm font-medium text-red-700 transition-colors hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-red-900/60 dark:bg-dark-800 dark:text-red-300 dark:hover:bg-red-900/20"
        :disabled="loading || updating || restarting"
        @click="confirmRollback"
      >
        <Icon name="sync" size="sm" :stroke-width="2" />
        {{ t('version.rollback') }}
      </button>
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
      v-if="updating && actionProgress"
      data-test="version-action-progress"
      role="status"
      aria-live="polite"
      class="flex items-start gap-3 rounded-xl border border-blue-200 bg-blue-50 p-4 dark:border-blue-800/50 dark:bg-blue-900/20"
    >
      <Icon name="refresh" size="md" :stroke-width="2" class="mt-0.5 flex-shrink-0 animate-spin text-blue-600 dark:text-blue-400" />
      <p class="text-sm text-blue-800 dark:text-blue-200">{{ actionProgress }}</p>
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
          {{ actionFailureTitle }}
        </p>
        <p class="mt-1 break-words text-sm text-red-700/80 dark:text-red-300/80">
          {{ updateError }}
        </p>
        <p class="mt-2 text-sm text-red-700/80 dark:text-red-300/80">
          {{ actionFailureHint }}
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
          {{ actionSuccessTitle }}
        </p>
        <p class="mt-1 text-sm text-green-700/80 dark:text-green-300/80">
          {{ needRestart ? t('version.restartRequired') : t('version.restartNotRequired') }}
        </p>
        <p
          v-if="forcedWithoutBackup"
          data-test="forced-update-success"
          class="mt-2 text-sm font-medium text-amber-700 dark:text-amber-300"
        >
          {{ t('version.forceUpdateCompletedWarning') }}
        </p>
        <div
          v-if="backupSnapshot"
          class="mt-3 rounded-lg border border-green-200 bg-white/70 p-3 dark:border-green-800/60 dark:bg-dark-900/30"
        >
          <p class="text-sm font-medium text-green-800 dark:text-green-200">
            {{ t('version.backupSnapshotCreated', { id: backupSnapshot.id }) }}
          </p>
          <p class="mt-1 text-sm text-green-700/80 dark:text-green-300/80">
            {{ t('version.backupRestoreHint') }}
          </p>
          <a
            :href="backupSettingsPath"
            class="mt-2 inline-flex items-center gap-1 text-sm font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
          >
            {{ t('version.openBackupSettings') }}
            <Icon name="arrowRight" size="xs" :stroke-width="2" />
          </a>
        </div>
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
              data-test="view-upgrade-changes"
              class="inline-flex items-center justify-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 dark:hover:bg-dark-700"
              @click="showUpgradeChanges(release)"
            >
              <Icon name="infoCircle" size="sm" :stroke-width="2" />
              {{ t('version.viewUpgradeChanges') }}
            </button>
            <button
              v-if="canInPlaceUpdate"
              type="button"
              data-test="install-version"
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
      :title="confirmationTitle"
      :message="confirmationMessage"
      :confirm-text="confirmationActionText"
      :cancel-text="t('common.cancel')"
      :danger="selectedAction === 'rollback' || updateWillBypassBackup"
      @confirm="handleConfirmedAction"
      @cancel="confirmDialogOpen = false"
    >
      <div class="rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800/50 dark:bg-amber-900/20">
        <p class="text-sm text-amber-800 dark:text-amber-200">
          {{ t('version.dataSafeHint') }}
        </p>
      </div>
      <label
        v-if="backupFeatureEnabled"
        class="flex cursor-pointer items-start gap-3 rounded-lg border border-primary-200 bg-primary-50 p-3 text-sm dark:border-primary-900/60 dark:bg-primary-900/20"
      >
        <input
          id="version-backup-current"
          v-model="backupCurrent"
          type="checkbox"
          class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        />
        <span>
          <span class="block font-medium text-primary-900 dark:text-primary-100">
            {{ t('version.backupCurrent') }}
          </span>
          <span class="mt-1 block text-primary-800/80 dark:text-primary-200/80">
            {{ t('version.backupCurrentDescription') }}
          </span>
          <span class="mt-2 block text-xs text-primary-800/80 dark:text-primary-200/80">
            {{ t('version.backupScope') }}
          </span>
        </span>
      </label>
      <label
        v-else-if="selectedAction === 'install'"
        class="flex cursor-pointer items-start gap-3 rounded-lg border border-red-200 bg-red-50 p-3 text-sm dark:border-red-900/60 dark:bg-red-900/20"
      >
        <input
          id="version-force-without-backup"
          v-model="forceWithoutBackup"
          type="checkbox"
          class="mt-0.5 h-4 w-4 rounded border-red-300 text-red-600 focus:ring-red-500"
        />
        <span>
          <span class="block font-medium text-red-900 dark:text-red-100">
            {{ t('version.forceWithoutBackup') }}
          </span>
          <span class="mt-1 block text-red-800/80 dark:text-red-200/80">
            {{ t('version.forceWithoutBackupDescription') }}
          </span>
        </span>
      </label>
      <div
        v-if="selectedAction === 'install' && updateWillBypassBackup"
        data-test="force-update-warning"
        class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-200"
      >
        {{ t('version.forceWithoutBackupWarning') }}
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
  rollback,
  restartService,
  type UpdateCapabilities,
  type VersionRelease
} from '@/api/admin/system'
import type { BackupRecord } from '@/api/admin/backup'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import UpgradeChangesDialog from '@/components/common/UpgradeChangesDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { isProgressiveFeatureEnabled, ProgressiveFeatures } from '@/utils/progressiveFeatures'

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
const updateErrorReason = ref('')
const updateSuccess = ref(false)
const needRestart = ref(false)
const restarting = ref(false)
const restartCountdown = ref(0)
const confirmDialogOpen = ref(false)
const selectedAction = ref<'install' | 'rollback'>('install')
const selectedRelease = ref<VersionRelease | null>(null)
const upgradeChangesRelease = ref<VersionRelease | null>(null)
const upgradeChangesOpen = ref(false)
const versionType = ref<'production' | 'preview'>('production')
const backupCurrent = ref(true)
const forceWithoutBackup = ref(false)
const forcedWithoutBackup = ref(false)
const backupSnapshot = ref<BackupRecord | null>(null)
const actionProgress = ref('')
const backupSettingsPath = '/admin/settings/backup'

const isReleaseBuild = computed(() => buildType.value === 'release')
const isContainerDeployment = computed(() => updateCapabilities.value.deployment_type === 'container')
const backupFeatureEnabled = computed(() => isProgressiveFeatureEnabled(ProgressiveFeatures.backup))
const canInPlaceUpdate = computed(
  () => isReleaseBuild.value && updateCapabilities.value.can_in_place_update
)
const canRollback = computed(
  () => canInPlaceUpdate.value && updateCapabilities.value.can_rollback
)
const canRestart = computed(() => updateCapabilities.value.can_restart)
const updateWillBypassBackup = computed(() =>
  selectedAction.value === 'install' && (
    backupFeatureEnabled.value ? !backupCurrent.value : forceWithoutBackup.value
  )
)
const backupFailure = computed(() =>
  selectedAction.value === 'install' && updateErrorReason.value.startsWith('SYSTEM_VERSION_BACKUP_')
)
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
const confirmationTitle = computed(() =>
  selectedAction.value === 'rollback'
    ? t('version.rollbackConfirmTitle')
    : t('version.installConfirmTitle')
)
const confirmationMessage = computed(() =>
  selectedAction.value === 'rollback'
    ? t('version.rollbackConfirmMessage')
    : installConfirmMessage.value
)
const confirmationActionText = computed(() => {
  if (selectedAction.value === 'rollback') return t('version.rollback')
  return updateWillBypassBackup.value ? t('version.forceUpdate') : t('version.installVersion')
})
const actionSuccessTitle = computed(() =>
  selectedAction.value === 'rollback'
    ? t('version.rollbackComplete')
    : t('version.updateComplete')
)
const actionFailureTitle = computed(() => {
  if (backupFailure.value) return t('version.backupFailedTitle')
  return selectedAction.value === 'rollback'
    ? t('version.rollbackFailed')
    : t('version.updateFailed')
})
const actionFailureHint = computed(() =>
  backupFailure.value ? t('version.backupFailedHint') : t('version.actionFailureHint')
)

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

function getErrorReason(error: unknown): string {
  const err = error as {
    reason?: unknown
    response?: { data?: { reason?: unknown } }
  }
  const reason = err.reason ?? err.response?.data?.reason
  return typeof reason === 'string' ? reason : ''
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
  openActionConfirmation('install')
}

function confirmRollback() {
  if (!canRollback.value) return
  selectedRelease.value = null
  openActionConfirmation('rollback')
}

function openActionConfirmation(action: 'install' | 'rollback') {
  selectedAction.value = action
  backupCurrent.value = backupFeatureEnabled.value
  forceWithoutBackup.value = false
  forcedWithoutBackup.value = false
  backupSnapshot.value = null
  updateError.value = ''
  updateErrorReason.value = ''
  updateSuccess.value = false
  needRestart.value = false
  actionProgress.value = ''
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

async function handleConfirmedAction() {
  const action = selectedAction.value
  if (updating.value) return
  if (action === 'install' && (!selectedRelease.value || !canInPlaceUpdate.value)) return
  if (action === 'rollback' && !canRollback.value) return

  confirmDialogOpen.value = false
  updating.value = true
  updateError.value = ''
  updateErrorReason.value = ''
  updateSuccess.value = false
  needRestart.value = false
  forcedWithoutBackup.value = false
  backupSnapshot.value = null
  const actionLabel = action === 'rollback' ? t('version.rollingBack') : t('version.installing')
  actionProgress.value = action === 'install' && !updateWillBypassBackup.value
    ? t('version.backupActionProgress', { action: actionLabel })
    : updateWillBypassBackup.value
      ? t('version.forceUpdateProgress', { action: actionLabel })
      : t('version.actionProgress', { action: actionLabel })

  try {
    const release = selectedRelease.value
    const result = action === 'rollback'
      ? await rollback(backupFeatureEnabled.value ? { backup_current: backupCurrent.value } : {})
      : await performUpdate(
        backupFeatureEnabled.value
          ? backupCurrent.value
            ? { version: release!.version, backup_current: true }
            : {
              version: release!.version,
              backup_current: false,
              force_without_backup: true
            }
          : forceWithoutBackup.value
            ? { version: release!.version, force_without_backup: true }
            : { version: release!.version }
      )
    updateSuccess.value = true
    needRestart.value = result.need_restart
    forcedWithoutBackup.value = !!result.forced_without_backup
    backupSnapshot.value = result.backup || null
    if (action === 'install' && release) {
      upgradeChangesRelease.value = release
      upgradeChangesOpen.value = true
    }
    appStore.clearVersionCache()
    try {
      await loadReleases(true)
    } catch {
      // The install already succeeded; a follow-up refresh failure should not be shown as install failure.
    }
  } catch (error) {
    updateError.value = getErrorMessage(error, t('version.updateFailed'))
    updateErrorReason.value = getErrorReason(error)
  } finally {
    actionProgress.value = ''
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
