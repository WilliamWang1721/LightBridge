import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { UpdateResult } from '@/api/admin/system'

const { listVersionReleases, performUpdate, rollback, backupFeatureEnabled } = vi.hoisted(() => ({
  listVersionReleases: vi.fn(),
  performUpdate: vi.fn(),
  rollback: vi.fn(),
  backupFeatureEnabled: { value: true }
}))

const appStore = vi.hoisted(() => ({
  fetchVersion: vi.fn(),
  clearVersionCache: vi.fn(),
  currentVersion: '0.1.0',
  latestVersion: '0.1.1',
  buildType: 'release',
  updateCapabilities: {
    deployment_type: 'container',
    can_in_place_update: false,
    can_rollback: false,
    can_restart: false
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        const values = Object.values(params || {})
        return values.length > 0 ? `${key} ${values.join(' ')}` : key
      }
    })
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/api/admin/system', () => ({
  listVersionReleases,
  performUpdate,
  rollback,
  restartService: vi.fn()
}))

vi.mock('@/utils/progressiveFeatures', () => ({
  isProgressiveFeatureEnabled: () => backupFeatureEnabled.value,
  ProgressiveFeatures: { backup: { id: 'backup' } }
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: { template: '<div><slot /></div>' }
}))

vi.mock('@/components/common/ConfirmDialog.vue', () => ({
  default: {
    props: ['show'],
    emits: ['confirm', 'cancel'],
    template: `
      <div v-if="show" data-test="confirm-dialog">
        <slot />
        <button type="button" data-test="confirm-action" @click="$emit('confirm')">Confirm</button>
      </div>
    `
  }
}))

vi.mock('@/components/common/UpgradeChangesDialog.vue', () => ({
  default: {
    props: ['show'],
    emits: ['upgrade'],
    template: `
      <div v-if="show" data-test="upgrade-changes-dialog">
        <button type="button" data-test="upgrade-from-dialog" @click="$emit('upgrade')">Upgrade</button>
      </div>
    `
  }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: { template: '<span />' }
}))

import VersionControlView from '../VersionControlView.vue'

const containerCapabilities = {
  deployment_type: 'container',
  can_in_place_update: false,
  can_rollback: false,
  can_restart: false
}

const binaryCapabilities = {
  deployment_type: 'binary',
  can_in_place_update: true,
  can_rollback: true,
  can_restart: true
}

function configureCapabilities(capabilities: typeof containerCapabilities | typeof binaryCapabilities) {
  appStore.updateCapabilities = { ...capabilities }
  appStore.fetchVersion.mockResolvedValue({
    current_version: '0.1.0',
    latest_version: '0.1.1',
    has_update: true,
    build_type: 'release',
    cached: false,
    capabilities: { ...capabilities }
  })
  listVersionReleases.mockResolvedValue({
    current_version: '0.1.0',
    latest_version: '0.1.1',
    build_type: 'release',
    capabilities: { ...capabilities },
    releases: [
      {
        version: '0.1.1',
        name: 'v0.1.1',
        body: 'notes',
        published_at: '2026-08-02T00:00:00Z',
        html_url: 'https://example.test/releases/v0.1.1'
      }
    ]
  })
}

function mountView() {
  return mount(VersionControlView)
}

async function openInstallConfirmation() {
  const wrapper = mountView()
  await flushPromises()
  await wrapper.get('[data-test="install-version"]').trigger('click')
  return wrapper
}

describe('VersionControlView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    backupFeatureEnabled.value = true
    appStore.currentVersion = '0.1.0'
    appStore.latestVersion = '0.1.1'
    appStore.buildType = 'release'
    configureCapabilities(containerCapabilities)
    performUpdate.mockResolvedValue({ message: 'updated', need_restart: true })
    rollback.mockResolvedValue({ message: 'rolled back', need_restart: true })
  })

  it('shows Compose upgrade guidance and hides local lifecycle actions for containers', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('version.containerUpgradeTitle')
    expect(wrapper.text()).toContain('version.containerUpgradePull')
    expect(wrapper.text()).toContain('version.containerUpgradeUp')
    expect(wrapper.find('[data-test="install-version"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="rollback-version"]').exists()).toBe(false)
    expect(wrapper.find('#version-backup-current').exists()).toBe(false)
  })

  it('keeps the safe default when the backup feature is disabled', async () => {
    backupFeatureEnabled.value = false
    configureCapabilities({ ...binaryCapabilities, can_rollback: false })
    const wrapper = await openInstallConfirmation()

    expect(wrapper.find('#version-backup-current').exists()).toBe(false)
    expect(wrapper.get<HTMLInputElement>('#version-force-without-backup').element.checked).toBe(false)
    await wrapper.get('[data-test="confirm-action"]').trigger('click')
    await flushPromises()

    expect(performUpdate).toHaveBeenCalledWith({ version: '0.1.1' })
  })

  it('requires an explicit force choice when the backup feature is disabled', async () => {
    backupFeatureEnabled.value = false
    configureCapabilities({ ...binaryCapabilities, can_rollback: false })
    performUpdate.mockResolvedValue({
      message: 'updated',
      need_restart: true,
      forced_without_backup: true
    })
    const wrapper = await openInstallConfirmation()

    await wrapper.get<HTMLInputElement>('#version-force-without-backup').setValue(true)
    expect(wrapper.get('[data-test="force-update-warning"]').text()).toContain('version.forceWithoutBackupWarning')
    await wrapper.get('[data-test="confirm-action"]').trigger('click')
    await flushPromises()

    expect(performUpdate).toHaveBeenCalledWith({
      version: '0.1.1',
      force_without_backup: true
    })
    expect(wrapper.get('[data-test="forced-update-success"]').text()).toContain('version.forceUpdateCompletedWarning')
  })

  it('defaults the backup option to selected, reports backup sequencing, and exposes the created snapshot', async () => {
    configureCapabilities(binaryCapabilities)
    let resolveUpdate: (value: UpdateResult) => void = () => undefined
    performUpdate.mockImplementation(
      () => new Promise<UpdateResult>((resolve) => {
        resolveUpdate = resolve
      })
    )

    const wrapper = await openInstallConfirmation()
    const backupCheckbox = wrapper.get<HTMLInputElement>('#version-backup-current')
    expect(backupCheckbox.element.checked).toBe(true)
    expect(wrapper.text()).toContain('version.backupScope')

    await wrapper.get('[data-test="confirm-action"]').trigger('click')
    expect(performUpdate).toHaveBeenCalledWith({ version: '0.1.1', backup_current: true })
    expect(wrapper.get('[data-test="version-action-progress"]').text()).toContain('version.backupActionProgress')

    resolveUpdate({
      message: 'updated',
      need_restart: true,
      backup: {
        id: 'snapshot-123',
        status: 'completed',
        backup_type: 'postgresql',
        file_name: 'snapshot-123.sql.gz',
        s3_key: 'backups/snapshot-123.sql.gz',
        size_bytes: 1024,
        triggered_by: 'manual',
        started_at: '2026-08-03T00:00:00Z',
        metadata: {
          source_version: '0.1.0',
          version_action: 'update',
          target_version: '0.1.1',
          system_operation_id: 'operation-123',
          initiating_admin_id: 42
        }
      }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('version.backupSnapshotCreated snapshot-123')
    expect(wrapper.get('a[href="/admin/settings/backup"]').text()).toContain('version.openBackupSettings')
  })

  it('sends an explicit force flag when the administrator unselects backup', async () => {
    configureCapabilities(binaryCapabilities)
    const wrapper = await openInstallConfirmation()

    await wrapper.get<HTMLInputElement>('#version-backup-current').setValue(false)
    expect(wrapper.get('[data-test="force-update-warning"]').text()).toContain('version.forceWithoutBackupWarning')
    await wrapper.get('[data-test="confirm-action"]').trigger('click')
    await flushPromises()

    expect(performUpdate).toHaveBeenCalledWith({
      version: '0.1.1',
      backup_current: false,
      force_without_backup: true
    })
  })

  it('routes changelog-triggered installs through the same backup confirmation', async () => {
    configureCapabilities(binaryCapabilities)
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="view-upgrade-changes"]').trigger('click')
    await wrapper.get('[data-test="upgrade-from-dialog"]').trigger('click')

    expect(wrapper.get('[data-test="confirm-dialog"]').exists()).toBe(true)
    expect(wrapper.get<HTMLInputElement>('#version-backup-current').element.checked).toBe(true)
  })

  it('confirms rollback separately and sends its selected backup payload', async () => {
    configureCapabilities(binaryCapabilities)
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="rollback-version"]').trigger('click')
    expect(wrapper.get<HTMLInputElement>('#version-backup-current').element.checked).toBe(true)
    await wrapper.get('[data-test="confirm-action"]').trigger('click')
    await flushPromises()

    expect(rollback).toHaveBeenCalledWith({ backup_current: true })
    expect(performUpdate).not.toHaveBeenCalled()
  })

  it('shows an update-not-started message for pre-update backup failures', async () => {
    configureCapabilities(binaryCapabilities)
    performUpdate.mockRejectedValue({
      message: 'snapshot upload failed',
      reason: 'SYSTEM_VERSION_BACKUP_FAILED'
    })
    const wrapper = await openInstallConfirmation()

    await wrapper.get('[data-test="confirm-action"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('snapshot upload failed')
    expect(wrapper.text()).toContain('version.backupFailedTitle')
    expect(wrapper.text()).toContain('version.backupFailedHint')
    expect(wrapper.text()).not.toContain('version.updateComplete')
  })

  it('shows a retryable generic failure without reporting a successful action', async () => {
    configureCapabilities(binaryCapabilities)
    performUpdate.mockRejectedValue(new Error('binary replacement failed'))
    const wrapper = await openInstallConfirmation()

    await wrapper.get('[data-test="confirm-action"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('binary replacement failed')
    expect(wrapper.text()).toContain('version.actionFailureHint')
    expect(wrapper.text()).not.toContain('version.updateComplete')
    expect(wrapper.find('a[href="/admin/settings/backup"]').exists()).toBe(false)
  })
})
