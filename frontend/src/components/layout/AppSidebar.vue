<template>
  <aside
    id="lightbridge-sidebar"
    class="sidebar w-64"
    :aria-label="t('nav.mainNavigation', 'Main navigation')"
    :class="{ '-translate-x-full lg:translate-x-0': !mobileOpen }"
  >
    <ReactPageHost
      :load="loadConsoleSidebar"
      :props="sidebarProps"
      :error-message="t('common.error')"
    />
  </aside>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  Bell as BellIcon,
  ChartNoAxesColumn as ChartIcon,
  CircleDollarSign as RechargeSubscriptionIcon,
  CreditCard as CreditCardIcon,
  Database as DatabaseIcon,
  FileText as OrderIcon,
  Folder as FolderIcon,
  Gift as GiftIcon,
  Globe2 as GlobeIcon,
  KeyRound as KeyIcon,
  LayoutDashboard as DashboardIcon,
  Layers3 as ChannelIcon,
  ListOrdered as OrderListIcon,
  MessageSquare as FeedbackIcon,
  Radio as SignalIcon,
  Send as DistributionIcon,
  Server as ServerIcon,
  Sparkles as AppearanceIcon,
  Settings as CogIcon,
  ShieldCheck as ShieldIcon,
  Tag as PriceTagIcon,
  Ticket as TicketIcon,
  TriangleAlert as ErrorAnalysisIcon,
  UserRound as UserIcon,
  UsersRound as UsersIcon,
} from '@lucide/vue'
import { useAdminSettingsStore, useAppStore, useAuthStore, useOnboardingStore } from '@/stores'
import ReactPageHost from '@/console/ReactPageHost.vue'
import type { ConsoleSidebarItem, ConsoleSidebarProps, ConsoleSidebarSection } from '@/console/sidebar/contract'
import { sidebarIconSvgs, type SidebarIconName } from '@/console/sidebar/icons'
import { makeProgressiveSidebarFlag, ProgressiveFeatures } from '@/utils/progressiveFeatures'
import { moduleMenuContributions, resolveModuleText } from '@/modules/runtime/registry'

interface NavItem {
  path: string
  label: string
  icon: unknown
  iconSvg?: string
  hideInSimpleMode?: boolean
  children?: NavItem[]
  /**
   * When true, the parent item only toggles the expand/collapse state and
   * does NOT navigate to its `path`. The `path` is purely a stable key.
   */
  expandOnly?: boolean
  /**
   * 可选的功能开关 getter。返回 false 时菜单项被隐藏；返回 undefined/true 时显示。
   * 宽容策略（undefined → 显示）避免 public settings 未加载完成时菜单闪烁消失。
   * Getter 里访问的 reactive 来源（store / composable）会被 computed 自动追踪，
   * 开关切换时菜单自动更新。
   */
  featureFlag?: () => boolean | undefined
}

interface NavSection {
  key: string
  title?: string
  items: NavItem[]
}

// applyFeatureFlags 递归过滤掉 featureFlag() === false 的节点（含子节点）。
// 使用 `!== false` 宽容语义：undefined（设置未加载）或 true 都视为显示。
function applyFeatureFlags(items: NavItem[]): NavItem[] {
  const out: NavItem[] = []
  for (const item of items) {
    if (item.featureFlag && item.featureFlag() === false) continue
    if (item.children) {
      const children = applyFeatureFlags(item.children)
      if (item.expandOnly && children.length === 0) continue
      out.push({ ...item, children })
    } else {
      out.push(item)
    }
  }
  return out
}

const { t, locale } = useI18n()

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const onboardingStore = useOnboardingStore()
const adminSettingsStore = useAdminSettingsStore()
const loadConsoleSidebar = () => import('@/console/react/ConsoleSidebar')

const mobileOpen = computed(() => appStore.mobileOpen)
const isAdmin = computed(() => authStore.isAdmin)

// Track which parent nav groups are expanded
const expandedGroups = ref<Set<string>>(new Set())

// Site settings from appStore (cached, no flicker)
const siteName = computed(() => appStore.siteName)
const siteLogo = computed(() => appStore.siteLogo)
const siteVersion = computed(() => appStore.siteVersion)


// Progressive module flags go through utils/progressiveFeatures.ts, the same
// registry used by dynamic route registration.
const flagChannelMonitor = makeProgressiveSidebarFlag(ProgressiveFeatures.channelMonitor)
const flagPayment = makeProgressiveSidebarFlag(ProgressiveFeatures.payment)
const flagAvailableChannels = makeProgressiveSidebarFlag(ProgressiveFeatures.availableChannels)
const flagAffiliate = makeProgressiveSidebarFlag(ProgressiveFeatures.affiliate)
const flagRiskControl = makeProgressiveSidebarFlag(ProgressiveFeatures.riskControl)
const flagPrivacyFilter = makeProgressiveSidebarFlag(ProgressiveFeatures.privacyFilter)
const flagAnnouncements = makeProgressiveSidebarFlag(ProgressiveFeatures.announcements)
const flagRedeem = makeProgressiveSidebarFlag(ProgressiveFeatures.redeem)
const flagPromo = makeProgressiveSidebarFlag(ProgressiveFeatures.promo)
const flagProxies = makeProgressiveSidebarFlag(ProgressiveFeatures.proxies)
const flagChannelPricing = makeProgressiveSidebarFlag(ProgressiveFeatures.channelPricing)
const flagSubscriptions = makeProgressiveSidebarFlag(ProgressiveFeatures.subscriptions)
const flagOpsMonitoring = makeProgressiveSidebarFlag(ProgressiveFeatures.opsMonitoring)
const flagModuleRuntime = makeProgressiveSidebarFlag(ProgressiveFeatures.moduleRuntime)

// anyFlags：任一子开关「非 false」即显示（用于分组：只要有一个子项启用，分组就显示）
function anyFlags(...flags: Array<() => boolean | undefined>): () => boolean {
  return () => flags.some((f) => f() !== false)
}

function moduleNavItems(group: string): NavItem[] {
  const normalizedGroup = group.trim().toLowerCase()
  const knownGroups = new Set(['operations', 'channels', 'commerce', 'marketing', 'security', 'system'])
  return moduleMenuContributions.value
    .filter((item) => {
      const requested = (item.group || 'system').trim().toLowerCase()
      const resolved = knownGroups.has(requested) ? requested : 'system'
      return resolved === normalizedGroup
    })
    .map((item) => ({
      path: item.path,
      label: resolveModuleText(item.title, item.title_i18n, locale.value),
      icon: ServerIcon,
      featureFlag: flagModuleRuntime,
    }))
}

// buildSelfNavItems 构造用户自己的导航项（用户端主菜单和管理员的"我的账户"子菜单共享这组声明）。
// withDashboard=true 时包含仪表盘（用户端），false 时不含（管理员的个人区已经有独立仪表盘入口）。
function buildSelfNavItems(withDashboard: boolean): NavItem[] {
  const items: NavItem[] = []
  if (withDashboard) {
    items.push({ path: '/dashboard', label: t('nav.dashboard'), icon: DashboardIcon })
    items.push({ path: '/distributions', label: t('nav.distributions'), icon: DistributionIcon })
  }

  // API 与监控
  items.push({
    path: '/self/api-monitor-group',
    label: t('nav.groupApiMonitor'),
    icon: KeyIcon,
    expandOnly: true,
    children: [
      { path: '/keys', label: t('nav.apiKeys'), icon: KeyIcon },
      { path: '/usage', label: t('nav.usage'), icon: ChartIcon, hideInSimpleMode: true },
      { path: '/model-catalog', label: t('nav.modelCatalog'), icon: DatabaseIcon },
      { path: '/monitor', label: t('nav.channelStatus'), icon: SignalIcon, featureFlag: flagChannelMonitor },
    ],
  })

  // 渠道与订阅
  items.push({
    path: '/self/channel-sub-group',
    label: t('nav.groupChannelSub'),
    icon: CreditCardIcon,
    expandOnly: true,
    children: [
      { path: '/available-channels', label: t('nav.availableChannels'), icon: ChannelIcon, hideInSimpleMode: true, featureFlag: flagAvailableChannels },
      { path: '/subscriptions', label: t('nav.mySubscriptions'), icon: CreditCardIcon, hideInSimpleMode: true, featureFlag: flagSubscriptions },
      { path: '/purchase', label: t('nav.buySubscription'), icon: RechargeSubscriptionIcon, hideInSimpleMode: true, featureFlag: flagPayment },
      { path: '/orders', label: t('nav.myOrders'), icon: OrderListIcon, hideInSimpleMode: true, featureFlag: flagPayment },
      { path: '/redeem', label: t('nav.redeem'), icon: GiftIcon, hideInSimpleMode: true, featureFlag: flagRedeem },
    ],
  })

  // 推广
  items.push({
    path: '/self/affiliate-group',
    label: t('nav.groupAffiliate'),
    icon: UsersIcon,
    expandOnly: true,
    hideInSimpleMode: true,
    featureFlag: flagAffiliate,
    children: [
      { path: '/affiliate', label: t('nav.affiliate'), icon: UsersIcon },
    ],
  })

  // 个人资料
  items.push({ path: '/profile', label: t('nav.profile'), icon: UserIcon })
  if (!withDashboard) {
    items.push({ path: '/distributions', label: t('nav.distributions'), icon: DistributionIcon })
  }

  // 自定义菜单
  for (const item of customMenuItemsForUser.value) {
    items.push({
      path: `/custom/${item.id}`,
      label: item.label,
      icon: null,
      iconSvg: item.icon_svg,
    })
  }

  return items
}

// finalizeNav 合并三重过滤：featureFlag 过滤 + simple 模式过滤。
function finalizeNav(items: NavItem[]): NavItem[] {
  const visible = applyFeatureFlags(items)
  return authStore.isSimpleMode ? visible.filter(item => !item.hideInSimpleMode) : visible
}

// User navigation items (for regular users)
const userNavItems = computed((): NavItem[] => finalizeNav(buildSelfNavItems(true)))

// Personal navigation items (for admin's "My Account" section, without Dashboard).
// Admins access 可用渠道 from this section just like regular users — there is no
// separate admin entry, since the page is purely a user-facing view.
const personalNavItems = computed((): NavItem[] => finalizeNav(buildSelfNavItems(false)))

// Custom menu items filtered by visibility
const customMenuItemsForUser = computed(() => {
  const items = appStore.cachedPublicSettings?.custom_menu_items ?? []
  return items
    .filter((item) => item.visibility === 'user')
    .sort((a, b) => a.sort_order - b.sort_order)
})

const customMenuItemsForAdmin = computed(() => {
  return adminSettingsStore.customMenuItems
    .filter((item) => item.visibility === 'admin')
    .sort((a, b) => a.sort_order - b.sort_order)
})

// Admin navigation items
const adminNavItems = computed((): NavItem[] => {
  const baseItems: NavItem[] = [
    // Dashboard
    { path: '/admin/dashboard', label: t('nav.dashboard'), icon: DashboardIcon },
    { path: '/admin/distributions', label: t('nav.distributions'), icon: DistributionIcon },

    // Operations group
    {
      path: '/admin/ops-group',
      label: t('nav.groupOperations'),
      icon: ChartIcon,
      expandOnly: true,
      featureFlag: flagOpsMonitoring,
      children: [
        { path: '/admin/ops', label: t('nav.ops'), icon: ChartIcon },
        { path: '/admin/error-analysis', label: t('nav.errorAnalysis'), icon: ErrorAnalysisIcon },
        ...moduleNavItems('operations'),
      ],
    },

    // Users & Access group
    {
      path: '/admin/users-group',
      label: t('nav.groupUsersAccess'),
      icon: UsersIcon,
      expandOnly: true,
      hideInSimpleMode: true,
      children: [
        { path: '/admin/users', label: t('nav.users'), icon: UsersIcon },
        { path: '/admin/auth-settings', label: t('nav.authSettings'), icon: ShieldIcon },
      ],
    },

    // Channels & Models group
    {
      path: '/admin/channels-group',
      label: t('nav.groupChannelsModels'),
      icon: ChannelIcon,
      expandOnly: true,
      hideInSimpleMode: true,
      children: [
        { path: '/admin/accounts', label: t('nav.accounts'), icon: GlobeIcon },
        { path: '/admin/groups', label: t('nav.groups'), icon: FolderIcon },
        { path: '/admin/channels/pricing', label: t('nav.channelPricing'), icon: PriceTagIcon, featureFlag: flagChannelPricing },
        { path: '/admin/channels/monitor', label: t('nav.channelMonitor'), icon: SignalIcon, featureFlag: flagChannelMonitor },
        { path: '/admin/model-catalog', label: t('nav.modelCatalog'), icon: DatabaseIcon },
        ...moduleNavItems('channels'),
      ],
    },

    // Commerce group
    {
      path: '/admin/commerce-group',
      label: t('nav.groupCommerce'),
      icon: CreditCardIcon,
      expandOnly: true,
      hideInSimpleMode: true,
      children: [
        { path: '/admin/subscriptions', label: t('nav.subscriptions'), icon: CreditCardIcon, featureFlag: flagSubscriptions },
        { path: '/admin/orders/dashboard', label: t('nav.paymentDashboard'), icon: ChartIcon, featureFlag: flagPayment },
        { path: '/admin/orders', label: t('nav.orderManagement'), icon: OrderIcon, featureFlag: flagPayment },
        { path: '/admin/orders/plans', label: t('nav.paymentPlans'), icon: CreditCardIcon, featureFlag: flagPayment },
        ...moduleNavItems('commerce'),
      ],
    },

    // Marketing group
    {
      path: '/admin/marketing-group',
      label: t('nav.groupMarketing'),
      icon: BellIcon,
      expandOnly: true,
      hideInSimpleMode: true,
      featureFlag: anyFlags(flagAffiliate, flagAnnouncements, flagRedeem, flagPromo),
      children: [
        { path: '/admin/announcements', label: t('nav.announcements'), icon: BellIcon, featureFlag: flagAnnouncements },
        { path: '/admin/affiliates/invites', label: t('nav.affiliateInviteRecords'), icon: UsersIcon, featureFlag: flagAffiliate },
        { path: '/admin/affiliates/rebates', label: t('nav.affiliateRebateRecords'), icon: OrderIcon, featureFlag: flagAffiliate },
        { path: '/admin/affiliates/transfers', label: t('nav.affiliateTransferRecords'), icon: CreditCardIcon, featureFlag: flagAffiliate },
        { path: '/admin/redeem', label: t('nav.redeemCodes'), icon: TicketIcon, featureFlag: flagRedeem },
        { path: '/admin/promo-codes', label: t('nav.promoCodes'), icon: GiftIcon, featureFlag: flagPromo },
        ...moduleNavItems('marketing'),
      ],
    },

    // Security group
    {
      path: '/admin/security-group',
      label: t('nav.groupSecurity'),
      icon: ShieldIcon,
      expandOnly: true,
      hideInSimpleMode: true,
      featureFlag: anyFlags(flagRiskControl, flagPrivacyFilter),
      children: [
        { path: '/admin/risk-control', label: t('nav.riskControl'), icon: ShieldIcon, featureFlag: flagRiskControl },
        { path: '/admin/privacy-filter', label: t('nav.privacyFilter'), icon: ShieldIcon, featureFlag: flagPrivacyFilter },
        ...moduleNavItems('security'),
      ],
    },

    // System group
    {
      path: '/admin/system-group',
      label: t('nav.groupSystem'),
      icon: CogIcon,
      expandOnly: true,
      children: [
        { path: '/admin/features', label: t('nav.featureRegistry'), icon: ServerIcon },
        { path: '/admin/proxies', label: t('nav.proxies'), icon: ServerIcon, featureFlag: flagProxies },
        { path: '/admin/proxy', label: 'Proxy Runtime', icon: ServerIcon, featureFlag: flagProxies },
        ...moduleNavItems('system'),
        { path: '/admin/usage', label: t('nav.usage'), icon: ChartIcon },
        { path: '/admin/feedback', label: t('nav.feedback'), icon: FeedbackIcon },
        { path: '/admin/appearance', label: t('nav.appearance'), icon: AppearanceIcon },
      ],
    },
  ]

  const visible = applyFeatureFlags(baseItems)

  // 简单模式下，在系统设置前插入 API密钥
  if (authStore.isSimpleMode) {
    const filtered = visible.filter(item => !item.hideInSimpleMode)
    filtered.push({ path: '/keys', label: t('nav.apiKeys'), icon: KeyIcon })
    filtered.push({ path: '/admin/settings', label: t('nav.settings'), icon: CogIcon })
    for (const cm of customMenuItemsForAdmin.value) {
      filtered.push({ path: `/custom/${cm.id}`, label: cm.label, icon: null, iconSvg: cm.icon_svg })
    }
    return filtered
  }

  visible.push({ path: '/admin/settings', label: t('nav.settings'), icon: CogIcon })
  for (const cm of customMenuItemsForAdmin.value) {
    visible.push({ path: `/custom/${cm.id}`, label: cm.label, icon: null, iconSvg: cm.icon_svg })
  }
  return visible
})

const navigationSections = computed<NavSection[]>(() => {
  if (isAdmin.value) {
    return [
      { key: 'admin', items: adminNavItems.value },
      ...(!authStore.isSimpleMode
        ? [{ key: 'personal', title: t('nav.myAccount'), items: personalNavItems.value }]
        : []),
    ]
  }

  return appStore.backendModeEnabled
    ? []
    : [{ key: 'personal', items: userNavItems.value }]
})

const sidebarIconNames = new Map<unknown, SidebarIconName>([
  [DashboardIcon, 'dashboard'],
  [KeyIcon, 'key'],
  [ChartIcon, 'chart'],
  [GiftIcon, 'gift'],
  [UserIcon, 'user'],
  [UsersIcon, 'users'],
  [FolderIcon, 'folder'],
  [ChannelIcon, 'channel'],
  [DatabaseIcon, 'database'],
  [CreditCardIcon, 'creditCard'],
  [RechargeSubscriptionIcon, 'recharge'],
  [GlobeIcon, 'globe'],
  [ServerIcon, 'server'],
  [BellIcon, 'bell'],
  [TicketIcon, 'ticket'],
  [CogIcon, 'settings'],
  [OrderIcon, 'order'],
  [OrderListIcon, 'orderList'],
  [DistributionIcon, 'send'],
  [SignalIcon, 'signal'],
  [ShieldIcon, 'shield'],
  [ErrorAnalysisIcon, 'alert'],
  [PriceTagIcon, 'tag'],
  [FeedbackIcon, 'feedback'],
  [AppearanceIcon, 'sparkles'],
])

function toConsoleSidebarItem(item: NavItem): ConsoleSidebarItem {
  const iconName = item.icon ? sidebarIconNames.get(item.icon) : undefined
  return {
    path: item.path,
    label: item.label,
    iconName,
    iconSvg: item.iconSvg || (iconName ? sidebarIconSvgs[iconName] : undefined),
    id:
      item.path === '/admin/accounts'
        ? 'sidebar-channel-manage'
        : item.path === '/admin/groups'
          ? 'sidebar-group-manage'
          : item.path === '/admin/redeem'
            ? 'sidebar-wallet'
            : undefined,
    dataTour: item.path === '/keys' ? 'sidebar-my-keys' : undefined,
    expandOnly: item.expandOnly,
    children: item.children?.map(toConsoleSidebarItem),
  }
}

const consoleSidebarSections = computed<ConsoleSidebarSection[]>(() =>
  navigationSections.value.map((section) => ({
    key: section.key,
    title: section.title,
    items: section.items.map(toConsoleSidebarItem),
  })),
)

function findNavItem(items: NavItem[], path: string): NavItem | undefined {
  for (const item of items) {
    if (item.path === path) return item
    const child = item.children && findNavItem(item.children, path)
    if (child) return child
  }
  return undefined
}

function toggleGroupPath(path: string) {
  const item = navigationSections.value
    .flatMap((section) => section.items)
    .map((item) => findNavItem([item], path))
    .find(Boolean)
  if (item) handleGroupClick(item)
}

const sidebarProps = computed<ConsoleSidebarProps>(() => ({
  sections: consoleSidebarSections.value,
  currentPath: route.path,
  expandedGroupPaths: Array.from(expandedGroups.value),
  labels: {
    mainNavigation: t('nav.mainNavigation'),
    closeNavigation: t('nav.closeNavigation'),
    profile: t('nav.profile'),
    apiKeys: t('nav.apiKeys'),
    settings: t('nav.settings'),
    github: t('nav.github'),
    contactSupport: t('common.contactSupport'),
    restartTour: t('onboarding.restartTour'),
    logout: t('nav.logout'),
  },
  mobileOpen: mobileOpen.value,
  user: authStore.user,
  isAdmin: isAdmin.value,
  contactInfo: appStore.contactInfo,
  showOnboardingButton: !authStore.isSimpleMode && authStore.user?.role === 'admin',
  siteName: siteName.value,
  siteLogo: siteLogo.value,
  siteVersion: siteVersion.value,
  onNavigate: (path) => {
    handleMenuItemClick(path)
    void router.push(path)
  },
  onToggleGroup: toggleGroupPath,
  onCloseMobile: closeMobile,
  onLogout: handleLogout,
  onReplayGuide: handleReplayGuide,
}))

function closeMobile() {
  appStore.setMobileOpen(false)
}

async function handleLogout() {
  try {
    await authStore.logout()
  } catch (error) {
    console.error('Logout error:', error)
  }
  await router.push('/login')
}

function handleReplayGuide() {
  onboardingStore.replay()
}

function handleMenuItemClick(itemPath: string) {
  if (mobileOpen.value) {
    appStore.setMobileOpen(false)
  }

  // Map paths to tour selectors
  const pathToSelector: Record<string, string> = {
    '/admin/groups': '#sidebar-group-manage',
    '/admin/accounts': '#sidebar-channel-manage',
    '/keys': '[data-tour="sidebar-my-keys"]'
  }

  const selector = pathToSelector[itemPath]
  if (selector && onboardingStore.isCurrentStep(selector)) {
    onboardingStore.nextStep(500)
  }
}

function toggleGroup(item: NavItem) {
  if (expandedGroups.value.has(item.path)) {
    expandedGroups.value.delete(item.path)
  } else {
    expandedGroups.value.add(item.path)
  }
}

/**
 * Click handler for collapsible parent items.
 * - When `expandOnly` is true: only toggle expand state.
 * - Otherwise (default, e.g. /admin/orders): navigate to the parent path
 *   (router-link semantics) and ensure the group is expanded.
 */
function handleGroupClick(item: NavItem) {
  if (item.expandOnly) {
    toggleGroup(item)
    return
  }
  // Push to path and ensure expanded
  if (route.path !== item.path) {
    router.push(item.path)
  }
  if (!expandedGroups.value.has(item.path)) {
    expandedGroups.value.add(item.path)
  }
}

// Fetch admin settings (for feature-gated nav items like Ops).
watch(
  isAdmin,
  (v) => {
    if (v) {
      adminSettingsStore.fetch()
    }
  },
  { immediate: true }
)

onMounted(() => {
  if (isAdmin.value) {
    adminSettingsStore.fetch()
  }
})
</script>
