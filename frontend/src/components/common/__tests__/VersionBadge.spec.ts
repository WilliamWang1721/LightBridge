import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

const appStore = vi.hoisted(() => ({
  versionLoading: false,
  currentVersion: '0.1.0',
  latestVersion: '0.1.1',
  hasUpdate: true,
  releaseInfo: null,
  buildType: 'release',
  updateCapabilities: {
    deployment_type: 'container',
    can_in_place_update: false,
    can_rollback: false,
    can_restart: false
  },
  fetchVersion: vi.fn(),
  clearVersionCache: vi.fn()
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

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn()
  })
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({ isAdmin: true }),
  useAppStore: () => appStore
}))

vi.mock('@/api/admin/system', () => ({
  performUpdate: vi.fn(),
  restartService: vi.fn()
}))

vi.mock('@/components/common/UpgradeChangesDialog.vue', () => ({
  default: { template: '<div />' }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: { template: '<span />' }
}))

import VersionBadge from '../VersionBadge.vue'

describe('VersionBadge', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    appStore.versionLoading = false
    appStore.currentVersion = '0.1.0'
    appStore.latestVersion = '0.1.1'
    appStore.hasUpdate = true
    appStore.releaseInfo = null
    appStore.buildType = 'release'
    appStore.updateCapabilities = {
      deployment_type: 'container',
      can_in_place_update: false,
      can_rollback: false,
      can_restart: false
    }
    appStore.fetchVersion.mockResolvedValue(undefined)
  })

  it('shows Compose guidance instead of local update or restart actions for containers', async () => {
    const wrapper = mount(VersionBadge)

    await wrapper.find('button').trigger('click')
    await nextTick()

    expect(wrapper.text()).toContain('version.containerUpgradeTitle')
    expect(wrapper.text()).toContain('version.containerUpgradePull')
    expect(wrapper.text()).toContain('version.containerUpgradeUp')
    expect(wrapper.text()).not.toContain('version.updateNow')
    expect(wrapper.text()).not.toContain('version.restartNow')
  })
})
