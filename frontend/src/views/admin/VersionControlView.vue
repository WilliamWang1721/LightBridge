<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex justify-end gap-2">
        <button v-if="canRollback" data-test="rollback-version" class="btn btn-danger" :disabled="busy" @click="openRollback">{{ t('version.rollback') }}</button>
        <button class="btn btn-secondary" :disabled="loading" @click="load(true)">{{ t('version.refresh') }}</button>
      </div>
      <div class="grid gap-4 md:grid-cols-3">
        <div class="card p-4"><p class="text-sm text-gray-500">{{ t('version.currentVersion') }}</p><p class="mt-2 text-2xl font-semibold">{{ displayVersion(currentVersion) }}</p></div>
        <div class="card p-4"><p class="text-sm text-gray-500">{{ t('version.latestVersion') }}</p><p class="mt-2 text-2xl font-semibold">{{ displayVersion(latestVersion) }}</p></div>
        <div class="card p-4"><p class="text-sm text-gray-500">{{ t('version.buildType') }}</p><p class="mt-2 text-2xl font-semibold">{{ buildType }}</p></div>
      </div>
      <div v-if="isContainer" class="rounded-lg border border-blue-200 bg-blue-50 p-4 text-sm text-blue-800">{{ t('version.containerUpgradeHint') }}</div>
      <div v-if="progress" data-test="version-action-progress" class="rounded-lg border border-blue-200 bg-blue-50 p-4 text-sm text-blue-800">{{ progress }}</div>
      <div v-if="errorMessage" class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-800">
        <p class="font-medium">{{ errorTitle }}</p><p class="mt-1 text-sm">{{ errorMessage }}</p><p class="mt-2 text-sm">{{ errorHint }}</p>
      </div>
      <div v-if="success" class="rounded-lg border border-green-200 bg-green-50 p-4 text-green-800">
        <p class="font-medium">{{ action === 'rollback' ? t('version.rollbackComplete') : t('version.updateComplete') }}</p>
        <p v-if="forcedWithoutBackup" class="mt-1 text-sm">已按管理员确认绕过备份完成更新。</p>
        <p v-if="backup" class="mt-1 text-sm">{{ t('version.backupSnapshotCreated', { id: backup.id }) }}</p>
      </div>
      <div class="card overflow-hidden">
        <div v-if="loading" class="p-10 text-center">{{ t('common.loading') }}</div>
        <div v-else class="divide-y">
          <div v-for="release in releases" :key="release.version" class="flex items-center justify-between gap-4 p-4">
            <div><p class="font-semibold">{{ displayVersion(release.version) }}</p><p class="text-xs text-gray-500">{{ release.name }}</p></div>
            <button v-if="canUpdate" data-test="install-version" class="btn btn-primary" :disabled="busy || same(release.version,currentVersion)" @click="openInstall(release)">{{ same(release.version,currentVersion) ? t('version.installed') : t('version.installVersion') }}</button>
          </div>
        </div>
      </div>
      <ConfirmDialog :show="dialog" :title="dialogTitle" :message="dialogMessage" :confirm-text="confirmText" :cancel-text="t('common.cancel')" :danger="action==='rollback'||forceWithoutBackup" @confirm="confirm" @cancel="dialog=false">
        <label v-if="action==='install'" class="mt-3 flex items-start gap-3 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          <input id="version-force-without-backup" v-model="forceWithoutBackup" type="checkbox" class="mt-0.5" />
          <span><strong class="block">强制更新（跳过备份）</strong><span>仅在备份不可用且已接受无法自动恢复数据库的风险时使用。</span></span>
        </label>
      </ConfirmDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { listVersionReleases, performUpdate, rollback, type VersionRelease, type UpdateCapabilities } from '@/api/admin/system'
import type { BackupRecord } from '@/api/admin/backup'

const { t } = useI18n()
const loading = ref(false), busy = ref(false), releases = ref<VersionRelease[]>([]), currentVersion = ref(''), latestVersion = ref(''), buildType = ref('')
const capabilities = ref<UpdateCapabilities>({ deployment_type: 'unknown', can_in_place_update: false, can_rollback: false, can_restart: false })
const dialog = ref(false), action = ref<'install'|'rollback'>('install'), selected = ref<VersionRelease|null>(null), forceWithoutBackup = ref(false), progress = ref(''), errorMessage = ref(''), errorReason = ref(''), success = ref(false), backup = ref<BackupRecord|null>(null), forcedWithoutBackup = ref(false)
const isContainer = computed(() => capabilities.value.deployment_type === 'container')
const canUpdate = computed(() => buildType.value === 'release' && capabilities.value.can_in_place_update)
const canRollback = computed(() => canUpdate.value && capabilities.value.can_rollback)
const dialogTitle = computed(() => action.value === 'rollback' ? t('version.rollbackConfirmTitle') : t('version.installConfirmTitle'))
const dialogMessage = computed(() => action.value === 'rollback' ? t('version.rollbackConfirmMessage') : t('version.installConfirmMessage', { version: displayVersion(selected.value?.version) }))
const confirmText = computed(() => action.value === 'rollback' ? t('version.rollback') : forceWithoutBackup.value ? '强制更新' : t('version.installVersion'))
const backupFailure = computed(() => String(errorReason.value).startsWith('SYSTEM_VERSION_BACKUP_'))
const errorTitle = computed(() => backupFailure.value ? '备份失败，更新未开始' : action.value === 'rollback' ? t('version.rollbackFailed') : t('version.updateFailed'))
const errorHint = computed(() => backupFailure.value ? '请修复备份配置后重试，或重新确认并选择“强制更新（跳过备份）”。' : t('version.actionFailureHint'))
function displayVersion(v?: string) { const n = String(v || '').replace(/^v/i, ''); return n ? `v${n}` : '--' }
function same(a?: string, b?: string) { return String(a || '').replace(/^v/i, '') === String(b || '').replace(/^v/i, '') }
async function load(force = false) { loading.value = true; try { const r = await listVersionReleases(force); releases.value = r.releases || []; currentVersion.value = r.current_version; latestVersion.value = r.latest_version; buildType.value = r.build_type; capabilities.value = r.capabilities } catch (e) { errorMessage.value = (e as Error).message } finally { loading.value = false } }
function reset() { errorMessage.value = ''; errorReason.value = ''; success.value = false; backup.value = null; forcedWithoutBackup.value = false; forceWithoutBackup.value = false }
function openInstall(r: VersionRelease) { reset(); action.value = 'install'; selected.value = r; dialog.value = true }
function openRollback() { reset(); action.value = 'rollback'; selected.value = null; dialog.value = true }
async function confirm() {
  dialog.value = false; busy.value = true
  const label = action.value === 'rollback' ? t('version.rollingBack') : t('version.installing')
  progress.value = action.value === 'install' && !forceWithoutBackup.value ? t('version.backupActionProgress', { action: label }) : t('version.actionProgress', { action: label })
  try {
    const result = action.value === 'rollback' ? await rollback({ backup_current: true }) : await performUpdate({ version: selected.value!.version, backup_current: !forceWithoutBackup.value, force_without_backup: forceWithoutBackup.value })
    success.value = true; backup.value = result.backup || null; forcedWithoutBackup.value = !!result.forced_without_backup; await load(true)
  } catch (e: any) { errorMessage.value = e?.message || t('version.updateFailed'); errorReason.value = String(e?.reason || e?.response?.data?.reason || '') }
  finally { busy.value = false; progress.value = '' }
}
onMounted(() => load())
</script>
