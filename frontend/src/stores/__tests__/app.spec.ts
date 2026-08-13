import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAppStore } from '@/stores/app'
import { getPublicSettings } from '@/api/auth'

// Mock API 模块
vi.mock('@/api/admin/system', () => ({
  checkUpdates: vi.fn(),
  unavailableUpdateCapabilities: {
    deployment_type: 'unknown',
    can_in_place_update: false,
    can_rollback: false,
    can_restart: false,
  },
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: vi.fn(),
}))

function createPublicSettings(overrides: Record<string, unknown> = {}) {
  return {
    registration_enabled: false,
    email_verify_enabled: false,
    force_email_on_third_party_signup: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: true,
    password_reset_enabled: false,
    invitation_code_enabled: false,
    turnstile_enabled: false,
    turnstile_site_key: '',
    site_name: 'LightBridge',
    site_logo: '',
    site_subtitle: '',
    api_base_url: '',
    contact_info: '',
    doc_url: '',
    home_content: '',
    hide_ccs_import_button: false,
    payment_enabled: false,
    risk_control_enabled: false,
    privacy_filter_enabled: false,
    table_default_page_size: 20,
    table_page_size_options: [10, 20, 50, 100],
    custom_menu_items: [],
    custom_endpoints: [],
    linuxdo_oauth_enabled: false,
    wechat_oauth_enabled: false,
    oidc_oauth_enabled: false,
    oidc_oauth_provider_name: 'OIDC',
    github_oauth_enabled: false,
    google_oauth_enabled: false,
    backend_mode_enabled: false,
    version: '1.0.0',
    balance_low_notify_enabled: false,
    account_quota_notify_enabled: false,
    balance_low_notify_threshold: 0,
    channel_monitor_enabled: true,
    channel_monitor_default_interval_seconds: 60,
    available_channels_enabled: false,
    affiliate_enabled: false,
    announcements_enabled: true,
    redeem_enabled: true,
    promo_enabled: true,
    proxies_enabled: true,
    channel_pricing_enabled: true,
    deployment_mode: 'distribution',
    ...overrides,
  } as any
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

describe('useAppStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.useFakeTimers()
    localStorage.clear()
    // 清除 window.__APP_CONFIG__
    delete (window as any).__APP_CONFIG__
  })

  afterEach(() => {
    vi.useRealTimers()
    localStorage.clear()
  })

  // --- Toast 消息管理 ---

  describe('Toast 消息管理', () => {
    it('showSuccess 创建 success 类型 toast', () => {
      const store = useAppStore()
      const id = store.showSuccess('操作成功')

      expect(id).toMatch(/^toast-/)
      expect(store.toasts).toHaveLength(1)
      expect(store.toasts[0].type).toBe('success')
      expect(store.toasts[0].message).toBe('操作成功')
    })

    it('showError 创建 error 类型 toast', () => {
      const store = useAppStore()
      store.showError('出错了')

      expect(store.toasts).toHaveLength(1)
      expect(store.toasts[0].type).toBe('error')
      expect(store.toasts[0].message).toBe('出错了')
    })

    it('showWarning 创建 warning 类型 toast', () => {
      const store = useAppStore()
      store.showWarning('警告信息')

      expect(store.toasts).toHaveLength(1)
      expect(store.toasts[0].type).toBe('warning')
    })

    it('showInfo 创建 info 类型 toast', () => {
      const store = useAppStore()
      store.showInfo('提示信息')

      expect(store.toasts).toHaveLength(1)
      expect(store.toasts[0].type).toBe('info')
    })

    it('toast 在指定 duration 后自动消失', () => {
      const store = useAppStore()
      store.showSuccess('临时消息', 3000)

      expect(store.toasts).toHaveLength(1)

      vi.advanceTimersByTime(3000)

      expect(store.toasts).toHaveLength(0)
    })

    it('hideToast 移除指定 toast', () => {
      const store = useAppStore()
      const id = store.showSuccess('消息1')
      store.showError('消息2')

      expect(store.toasts).toHaveLength(2)

      store.hideToast(id)

      expect(store.toasts).toHaveLength(1)
      expect(store.toasts[0].type).toBe('error')
    })

    it('clearAllToasts 清除所有 toast', () => {
      const store = useAppStore()
      store.showSuccess('消息1')
      store.showError('消息2')
      store.showWarning('消息3')

      expect(store.toasts).toHaveLength(3)

      store.clearAllToasts()

      expect(store.toasts).toHaveLength(0)
    })

    it('hasActiveToasts 正确反映 toast 状态', () => {
      const store = useAppStore()
      expect(store.hasActiveToasts).toBe(false)

      store.showSuccess('消息')
      expect(store.hasActiveToasts).toBe(true)

      store.clearAllToasts()
      expect(store.hasActiveToasts).toBe(false)
    })

    it('多个 toast 的 ID 唯一', () => {
      const store = useAppStore()
      const id1 = store.showSuccess('消息1')
      const id2 = store.showSuccess('消息2')
      const id3 = store.showSuccess('消息3')

      expect(id1).not.toBe(id2)
      expect(id2).not.toBe(id3)
    })
  })

  // --- 移动端侧边栏 ---

  describe('移动端侧边栏管理', () => {
    it('toggleMobileSidebar 切换移动端状态', () => {
      const store = useAppStore()
      expect(store.mobileOpen).toBe(false)

      store.toggleMobileSidebar()
      expect(store.mobileOpen).toBe(true)

      store.toggleMobileSidebar()
      expect(store.mobileOpen).toBe(false)
    })
  })

  // --- Loading 状态 ---

  describe('Loading 状态管理', () => {
    it('setLoading 管理引用计数', () => {
      const store = useAppStore()
      expect(store.loading).toBe(false)

      store.setLoading(true)
      expect(store.loading).toBe(true)

      store.setLoading(true) // 两次 true
      expect(store.loading).toBe(true)

      store.setLoading(false) // 第一次 false，计数还是 1
      expect(store.loading).toBe(true)

      store.setLoading(false) // 第二次 false，计数为 0
      expect(store.loading).toBe(false)
    })

    it('setLoading(false) 不会使计数为负', () => {
      const store = useAppStore()

      store.setLoading(false)
      store.setLoading(false)
      expect(store.loading).toBe(false)

      store.setLoading(true)
      expect(store.loading).toBe(true)

      store.setLoading(false)
      expect(store.loading).toBe(false)
    })

    it('withLoading 自动管理 loading 状态', async () => {
      const store = useAppStore()

      const result = await store.withLoading(async () => {
        expect(store.loading).toBe(true)
        return 'done'
      })

      expect(result).toBe('done')
      expect(store.loading).toBe(false)
    })

    it('withLoading 错误时也恢复 loading 状态', async () => {
      const store = useAppStore()

      await expect(
        store.withLoading(async () => {
          throw new Error('操作失败')
        })
      ).rejects.toThrow('操作失败')

      expect(store.loading).toBe(false)
    })

    it('withLoadingAndError 错误时显示 toast 并返回 null', async () => {
      const store = useAppStore()

      const result = await store.withLoadingAndError(async () => {
        throw new Error('网络错误')
      })

      expect(result).toBeNull()
      expect(store.loading).toBe(false)
      expect(store.toasts).toHaveLength(1)
      expect(store.toasts[0].type).toBe('error')
    })
  })

  // --- reset ---

  describe('reset', () => {
    it('重置所有 UI 状态', () => {
      const store = useAppStore()

      store.setLoading(true)
      store.showSuccess('消息')

      store.reset()

      expect(store.loading).toBe(false)
      expect(store.toasts).toHaveLength(0)
    })
  })

  // --- 公开设置 ---

  describe('公开设置加载', () => {
    it('从 window.__APP_CONFIG__ 初始化', () => {
      const windowAny = window as any
      windowAny.__APP_CONFIG__ = {
        site_name: 'TestSite',
        site_logo: '/logo.png',
        version: '1.0.0',
        contact_info: 'test@test.com',
        api_base_url: 'https://api.test.com',
        doc_url: 'https://docs.test.com',
      }

      const store = useAppStore()
      const result = store.initFromInjectedConfig()

      expect(result).toBe(true)
      expect(store.siteName).toBe('TestSite')
      expect(store.siteLogo).toBe('/logo.png')
      expect(store.siteVersion).toBe('1.0.0')
      expect(store.publicSettingsLoaded).toBe(true)
    })

    it('无注入配置时返回 false', () => {
      const store = useAppStore()
      const result = store.initFromInjectedConfig()

      expect(result).toBe(false)
      expect(store.publicSettingsLoaded).toBe(false)
    })

    it('clearPublicSettingsCache 清除缓存', () => {
      const windowAny = window as any
      windowAny.__APP_CONFIG__ = { site_name: 'Test' }
      const store = useAppStore()
      store.initFromInjectedConfig()

      expect(store.publicSettingsLoaded).toBe(true)

      store.clearPublicSettingsCache()

      expect(store.publicSettingsLoaded).toBe(false)
      expect(store.cachedPublicSettings).toBeNull()
    })

    it('fetchPublicSettings(force) 会同步更新运行时注入配置', async () => {
      vi.mocked(getPublicSettings).mockResolvedValue({
        registration_enabled: false,
        email_verify_enabled: false,
        registration_email_suffix_whitelist: [],
        promo_code_enabled: true,
        password_reset_enabled: false,
        invitation_code_enabled: false,
        turnstile_enabled: false,
        turnstile_site_key: '',
        site_name: 'Updated Site',
        site_logo: '',
        site_subtitle: '',
        api_base_url: '',
        contact_info: '',
        doc_url: '',
        home_content: '',
        hide_ccs_import_button: false,
        purchase_subscription_enabled: false,
        purchase_subscription_url: '',
        table_default_page_size: 1000,
        table_page_size_options: [20, 100, 1000],
        custom_menu_items: [],
        custom_endpoints: [],
        linuxdo_oauth_enabled: false,
        backend_mode_enabled: false,
        version: '1.0.0'
      })

      const store = useAppStore()
      await store.fetchPublicSettings(true)

      expect((window as any).__APP_CONFIG__.table_default_page_size).toBe(1000)
      expect((window as any).__APP_CONFIG__.table_page_size_options).toEqual([20, 100, 1000])
      expect(localStorage.getItem('table-page-size')).toBeNull()
      expect(localStorage.getItem('table-page-size-source')).toBeNull()
    })

    it('patchPublicSettings 会乐观合并缓存与运行时注入配置', () => {
      const store = useAppStore()
      ;(window as any).__APP_CONFIG__ = createPublicSettings({ redeem_enabled: false })
      store.initFromInjectedConfig()

      const patched = store.patchPublicSettings({ redeem_enabled: true })

      expect(patched?.redeem_enabled).toBe(true)
      expect(store.cachedPublicSettings?.redeem_enabled).toBe(true)
      expect((window as any).__APP_CONFIG__.redeem_enabled).toBe(true)
    })

    it('强制刷新不会被进行中的旧请求吞掉，也不会让旧结果覆盖乐观开关', async () => {
      const stale = deferred<ReturnType<typeof createPublicSettings>>()
      const fresh = deferred<ReturnType<typeof createPublicSettings>>()
      vi.mocked(getPublicSettings)
        .mockReturnValueOnce(stale.promise)
        .mockReturnValueOnce(fresh.promise)

      const store = useAppStore()
      ;(window as any).__APP_CONFIG__ = createPublicSettings({ redeem_enabled: false })
      store.initFromInjectedConfig()

      const firstRequest = store.fetchPublicSettings(true)
      store.patchPublicSettings({ redeem_enabled: true })
      const forcedRefresh = store.fetchPublicSettings(true)

      stale.resolve(createPublicSettings({ redeem_enabled: false }))
      await firstRequest
      await Promise.resolve()

      expect(store.cachedPublicSettings?.redeem_enabled).toBe(true)
      expect(getPublicSettings).toHaveBeenCalledTimes(2)

      fresh.resolve(createPublicSettings({ redeem_enabled: true }))
      await forcedRefresh

      expect(store.cachedPublicSettings?.redeem_enabled).toBe(true)
      expect((window as any).__APP_CONFIG__.redeem_enabled).toBe(true)
    })
  })
})
