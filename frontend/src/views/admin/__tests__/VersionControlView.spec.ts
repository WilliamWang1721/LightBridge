import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const { listVersionReleases } = vi.hoisted(() => ({
  listVersionReleases: vi.fn()
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
      t: (key: string) => key
    })
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/api/admin/system', () => ({
  listVersionReleases,
  performUpdate: vi.fn(),
  restartService: vi.fn()
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: { template: '<div><slot /></div>' }
}))

vi.mock('@/components/common/ConfirmDialog.vue', () => ({
  default: { template: '<div><slot /></div>' }
}))

vi.mock('@/components/common/UpgradeChangesDialog.vue', () => ({
  default: { template: '<div />' }
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

describe('VersionControlView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    appStore.currentVersion = '0.1.0'
    appStore.latestVersion = '0.1.1'
    appStore.buildType = 'release'
    appStore.updateCapabilities = { ...containerCapabilities }
    appStore.fetchVersion.mockResolvedValue({
      current_version: '0.1.0',
      latest_version: '0.1.1',
      has_update: true,
      build_type: 'release',
      cached: false,
      capabilities: { ...containerCapabilities }
    })
    listVersionReleases.mockResolvedValue({
      current_version: '0.1.0',
      latest_version: '0.1.1',
      build_type: 'release',
      capabilities: { ...containerCapabilities },
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
  })

  it('shows Compose upgrade guidance and hides local lifecycle actions for containers', async () => {
    const wrapper = mount(VersionControlView, {
      global: {
        stubs: {
          Icon: true,
          ConfirmDialog: true,
          UpgradeChangesDialog: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('version.containerUpgradeTitle')
    expect(wrapper.text()).toContain('version.containerUpgradePull')
    expect(wrapper.text()).toContain('version.containerUpgradeUp')
    expect(wrapper.text()).not.toContain('version.installVersion')
    expect(wrapper.text()).not.toContain('version.restartNow')
    expect(wrapper.text()).not.toContain('version.viewUpgradeChanges')
  })
})
