import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const {
  getS3Config,
  getSchedule,
  listBackups,
  showError,
  showSuccess,
  showWarning,
} = vi.hoisted(() => ({
  getS3Config: vi.fn(),
  getSchedule: vi.fn(),
  listBackups: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
}))

vi.mock('@/api', () => ({
  adminAPI: {
    backup: {
      getS3Config,
      updateS3Config: vi.fn(),
      testS3Connection: vi.fn(),
      getSchedule,
      updateSchedule: vi.fn(),
      createBackup: vi.fn(),
      listBackups,
      getBackup: vi.fn(),
      deleteBackup: vi.fn(),
      getDownloadURL: vi.fn(),
      restoreBackup: vi.fn(),
    },
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        const values = Object.values(params || {})
        return values.length > 0 ? `${key} ${values.join(' ')}` : key
      },
    }),
  }
})

import BackupView from '../BackupView.vue'

describe('BackupView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getS3Config.mockResolvedValue({
      endpoint: 'https://storage.example.test',
      region: 'auto',
      bucket: 'lightbridge-backups',
      access_key_id: 'configured',
      prefix: 'backups/',
      force_path_style: false,
    })
    getSchedule.mockResolvedValue({
      enabled: false,
      cron_expr: '0 2 * * *',
      retain_days: 14,
      retain_count: 10,
    })
    listBackups.mockResolvedValue({
      items: [
        {
          id: 'version-snapshot-1',
          status: 'completed',
          backup_type: 'postgresql',
          file_name: 'version-snapshot-1.sql.gz',
          s3_key: 'backups/version-snapshot-1.sql.gz',
          size_bytes: 1024,
          triggered_by: 'version_manager',
          started_at: '2026-08-03T00:00:00Z',
          metadata: {
            source_version: '0.1.0',
            version_action: 'update',
            target_version: '0.1.1',
            system_operation_id: 'operation-123',
            initiating_admin_id: 42,
          },
        },
      ],
    })
  })

  it('labels Version Manager snapshots with action metadata while retaining the restore action', async () => {
    const wrapper = mount(BackupView, {
      global: {
        stubs: { Teleport: true },
      },
    })
    await flushPromises()

    const snapshot = wrapper.get('[data-test="version-snapshot-version-snapshot-1"]')
    expect(snapshot.text()).toContain('admin.backup.versionSnapshot.label')
    expect(wrapper.text()).toContain('admin.backup.versionSnapshot.update v0.1.0 v0.1.1')
    expect(wrapper.text()).toContain('admin.backup.versionSnapshot.operationId operation-123')
    expect(wrapper.text()).toContain('admin.backup.versionSnapshot.initiatedBy 42')
    expect(wrapper.text()).toContain('admin.backup.trigger.versionManager')
    expect(wrapper.get('[data-test="restore-backup-version-snapshot-1"]').text()).toContain('admin.backup.actions.restore')
  })
})
