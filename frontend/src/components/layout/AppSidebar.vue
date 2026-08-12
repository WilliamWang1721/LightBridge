<template>
  <Sidebar
    id="lightbridge-sidebar"
    class="sidebar"
    :aria-label="t('nav.mainNavigation', 'Main navigation')"
    :class="[
      sidebarCollapsed ? 'w-[72px]' : 'w-64',
      { '-translate-x-full lg:translate-x-0': !mobileOpen }
    ]"
  >
    <!-- Logo/Brand -->
    <SidebarHeader class="sidebar-header" :class="{ 'sidebar-header-collapsed': sidebarCollapsed }">
      <div class="lightbridge-brand-mark" aria-hidden="true"></div>
      <div class="sidebar-brand" :class="{ 'sidebar-brand-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">
        <span class="sidebar-brand-title text-lg font-bold text-gray-900 dark:text-white">
          {{ siteName }}
        </span>
        <div class="sidebar-version-row">
          <!-- Version Badge -->
          <VersionBadge :version="siteVersion" />
          <span class="sidebar-version-product">LightBridge</span>
        </div>
      </div>
    </SidebarHeader>

    <ReactPageHost
      :load="loadConsoleSidebar"
      :props="sidebarProps"
      :error-message="t('common.error')"
    >
      <template #fallback>
        <!-- Vue fallback: kept until the React shell is proven on the running console. -->
        <SidebarContent class="sidebar-nav scrollbar-hide" :aria-label="t('nav.mainNavigation', 'Main navigation')">
      <SidebarGroup v-for="section in navigationSections" :key="section.key" class="sidebar-section !p-0">
        <SidebarGroupLabel
          v-if="section.title"
          class="sidebar-section-title"
          :class="{ 'sidebar-section-title-collapsed': sidebarCollapsed }"
          :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
        >
          <span
            class="sidebar-section-title-text"
            :class="{ 'sidebar-section-title-text-collapsed': sidebarCollapsed }"
          >{{ section.title }}</span>
        </SidebarGroupLabel>

        <SidebarMenu>
          <SidebarMenuItem v-for="item in section.items" :key="item.path">
          <template v-if="item.children?.length">
            <SidebarMenuButton
              as="button"
              type="button"
              class="sidebar-link sidebar-nav-item mb-1 w-full"
              :class="{
                'sidebar-link-active': isGroupActive(item) && !isGroupExpanded(item),
                'sidebar-link-collapsed': sidebarCollapsed
              }"
              :title="sidebarCollapsed ? item.label : undefined"
              :aria-expanded="!sidebarCollapsed && isGroupExpanded(item)"
              @click="handleGroupClick(item)"
            >
              <component :is="item.icon" class="h-5 w-5 flex-shrink-0" />
              <span
                class="sidebar-label sidebar-label-flex"
                :class="{ 'sidebar-label-collapsed': sidebarCollapsed }"
                :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
              >
                <span class="min-w-0 truncate">{{ item.label }}</span>
                <ChevronDownIcon
                  class="h-4 w-4 flex-shrink-0 transition-transform duration-200"
                  :class="isGroupExpanded(item) ? 'rotate-180' : ''"
                />
              </span>
            </SidebarMenuButton>
            <div
              v-if="!sidebarCollapsed && isGroupExpanded(item)"
              class="sidebar-subnav mb-1 ml-4 border-l border-gray-200 pl-2 dark:border-dark-600"
            >
              <SidebarMenuButton
                v-for="child in item.children"
                :key="child.path"
                :as="RouterLink"
                :to="child.path"
                class="sidebar-link sidebar-nav-item mb-0.5 py-1.5 text-sm"
                :class="{ 'sidebar-link-active': route.path === child.path }"
                :aria-current="route.path === child.path ? 'page' : undefined"
                @click="handleMenuItemClick(child.path)"
              >
                <component :is="child.icon" class="h-4 w-4 flex-shrink-0" />
                <span>{{ child.label }}</span>
              </SidebarMenuButton>
            </div>
          </template>

          <SidebarMenuButton
            v-else
            :as="RouterLink"
            :to="item.path"
            class="sidebar-link sidebar-nav-item mb-1"
            :class="{ 'sidebar-link-active': isActive(item.path), 'sidebar-link-collapsed': sidebarCollapsed }"
            :title="sidebarCollapsed ? item.label : undefined"
            :aria-current="isActive(item.path) ? 'page' : undefined"
            :data-tour="item.path === '/keys' ? 'sidebar-my-keys' : undefined"
            :id="
              item.path === '/admin/accounts'
                ? 'sidebar-channel-manage'
                : item.path === '/admin/groups'
                  ? 'sidebar-group-manage'
                  : item.path === '/admin/redeem'
                    ? 'sidebar-wallet'
                    : undefined
            "
            @click="handleMenuItemClick(item.path)"
          >
            <span v-if="item.iconSvg" class="h-5 w-5 flex-shrink-0 sidebar-svg-icon" v-html="sanitizeSvg(item.iconSvg)"></span>
            <component v-else :is="item.icon" class="h-5 w-5 flex-shrink-0" />
            <span
              class="sidebar-label"
              :class="{ 'sidebar-label-collapsed': sidebarCollapsed }"
              :aria-hidden="sidebarCollapsed ? 'true' : 'false'"
            >{{ item.label }}</span>
          </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarGroup>
        </SidebarContent>

        <SidebarFooter class="sidebar-footer mt-auto border-t border-gray-100 p-3 dark:border-dark-800">
      <div
        v-if="!sidebarCollapsed"
        class="sidebar-footer-controls"
      >
        <div class="relative">
          <SidebarMenuButton
            as="button"
            type="button"
            @click="toggleTheme"
            class="sidebar-link sidebar-footer-action w-full"
            :class="{ 'sidebar-link-collapsed': sidebarCollapsed }"
            :title="sidebarCollapsed ? (isDark ? t('nav.lightMode') : t('nav.darkMode')) : undefined"
            :aria-label="isDark ? t('nav.lightMode') : t('nav.darkMode')"
          >
            <SunIcon v-if="isDark" class="h-5 w-5 flex-shrink-0" />
            <MoonIcon v-else class="h-5 w-5 flex-shrink-0" />
            <span class="sidebar-label sidebar-footer-label" :class="{ 'sidebar-label-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">{{
              isDark ? t('nav.lightMode') : t('nav.darkMode')
            }}</span>
          </SidebarMenuButton>

        </div>

        <LocaleSwitcher variant="sidebar" />
      </div>

      <!-- Collapse Button -->
      <SidebarMenuButton
        as="button"
        type="button"
        @click="toggleSidebar"
        class="sidebar-link w-full"
        :class="{ 'sidebar-link-collapsed': sidebarCollapsed }"
        :title="sidebarCollapsed ? t('nav.expand') : t('nav.collapse')"
      >
        <ChevronDoubleLeftIcon v-if="!sidebarCollapsed" class="h-5 w-5 flex-shrink-0" />
        <ChevronDoubleRightIcon v-else class="h-5 w-5 flex-shrink-0" />
        <span class="sidebar-label sidebar-footer-label" :class="{ 'sidebar-label-collapsed': sidebarCollapsed }" :aria-hidden="sidebarCollapsed ? 'true' : 'false'">{{ t('nav.collapse') }}</span>
        </SidebarMenuButton>
        </SidebarFooter>
        <transition name="fade">
          <button
            v-if="mobileOpen"
            type="button"
            class="fixed inset-0 z-30 border-0 bg-black/50 p-0 lg:hidden"
            :aria-label="t('nav.closeNavigation')"
            @click="closeMobile"
          />
        </transition>
      </template>
    </ReactPageHost>
  </Sidebar>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  Bell as BellIcon,
  ChartNoAxesColumn as ChartIcon,
  ChevronDown as ChevronDownIcon,
  ChevronsLeft as ChevronDoubleLeftIcon,
  ChevronsRight as ChevronDoubleRightIcon,
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
  Moon as MoonIcon,
  Radio as SignalIcon,
  Server as ServerIcon,
  Settings as CogIcon,
  ShieldCheck as ShieldIcon,
  Sun as SunIcon,
  Tag as PriceTagIcon,
  Ticket as TicketIcon,
  TriangleAlert as ErrorAnalysisIcon,
  UserRound as UserIcon,
  UsersRound as UsersIcon,
} from '@lucide/vue'
import { useAdminSettingsStore, useAppStore, useAuthStore, useOnboardingStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import VersionBadge from '@/components/common/VersionBadge.vue'
import ReactPageHost from '@/console/ReactPageHost.vue'
import type { ConsoleSidebarItem, ConsoleSidebarProps, ConsoleSidebarSection } from '@/console/sidebar/contract'
import { sidebarIconSvgs, type SidebarIconName } from '@/console/sidebar/icons'
import { availableLocales, setLocale } from '@/i18n'
import { sanitizeSvg } from '@/utils/sanitize'
import { makeProgressiveSidebarFlag, ProgressiveFeatures } from '@/utils/progressiveFeatures'
import { moduleMenuContributions, resolveModuleText } from '@/modules/runtime/registry'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'

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

const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const mobileOpen = computed(() => appStore.mobileOpen)
const isAdmin = computed(() => authStore.isAdmin)
const isDark = ref(document.documentElement.classList.contains('dark'))

// Track which parent nav groups are expanded
const expandedGroups = ref<Set<string>>(new Set())

// Site settings from appStore (cached, no flicker)
const siteName = computed(() => appStore.siteName)
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
  [SignalIcon, 'signal'],
  [ShieldIcon, 'shield'],
  [ErrorAnalysisIcon, 'alert'],
  [PriceTagIcon, 'tag'],
  [FeedbackIcon, 'feedback'],
])

function toConsoleSidebarItem(item: NavItem): ConsoleSidebarItem {
  const iconName = item.icon ? sidebarIconNames.get(item.icon) : undefined
  return {
    path: item.path,
    label: item.label,
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
    lightMode: t('nav.lightMode'),
    darkMode: t('nav.darkMode'),
    collapse: t('nav.collapse'),
    expand: t('nav.expand'),
    closeNavigation: t('nav.closeNavigation'),
    locale: t('nav.languageSelect'),
  },
  localeOptions: availableLocales.map(({ code, name }) => ({ value: code, label: name })),
  currentLocale: locale.value,
  isDark: isDark.value,
  collapsed: sidebarCollapsed.value,
  mobileOpen: mobileOpen.value,
  onNavigate: (path) => {
    handleMenuItemClick(path)
    void router.push(path)
  },
  onToggleGroup: toggleGroupPath,
  onToggleTheme: toggleTheme,
  onLocaleChange: (code) => { void setLocale(code) },
  onToggleCollapse: toggleSidebar,
  onCloseMobile: closeMobile,
}))

function toggleSidebar() {
  appStore.toggleSidebar()
}

function setTheme(theme: 'light' | 'dark') {
  isDark.value = theme === 'dark'
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', theme)
}

function toggleTheme() {
  setTheme(isDark.value ? 'light' : 'dark')
}

function closeMobile() {
  appStore.setMobileOpen(false)
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

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(path + '/')
}

function isGroupActive(item: NavItem): boolean {
  if (!item.children) return false
  return item.children.some(child => route.path === child.path)
}

function isGroupExpanded(item: NavItem): boolean {
  return expandedGroups.value.has(item.path) || isGroupActive(item)
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
 * - When sidebar is collapsed: navigate to the first child (quick access).
 * - When `expandOnly` is true: only toggle expand state.
 * - Otherwise (default, e.g. /admin/orders): navigate to the parent path
 *   (router-link semantics) and ensure the group is expanded.
 */
function handleGroupClick(item: NavItem) {
  if (sidebarCollapsed.value) {
    // 收起状态下点击分组图标，导航到第一个子项
    if (item.children?.length) {
      router.push(item.children[0].path)
    }
    return
  }
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

// Initialize theme
const savedTheme = localStorage.getItem('theme')
if (
  savedTheme === 'dark' ||
  (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
) {
  isDark.value = true
  document.documentElement.classList.add('dark')
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

<style scoped>
.lightbridge-brand-mark {
  display: block !important;
  box-sizing: border-box !important;
  width: 28px !important;
  min-width: 28px !important;
  max-width: 28px !important;
  height: 28px !important;
  min-height: 28px !important;
  max-height: 28px !important;
  flex: 0 0 28px !important;
  padding: 0 !important;
  border: 0 !important;
  border-radius: 0 !important;
  background: #e42313 !important;
  box-shadow: none !important;
  opacity: 1 !important;
  filter: none !important;
  transform: none !important;
  transition: none !important;
}

.sidebar-header-collapsed {
  gap: 0;
  padding-left: 1.125rem;
  padding-right: 1.125rem;
}

.sidebar-brand {
  min-width: 0;
  flex: 1 1 auto;
  white-space: nowrap;
  transition:
    max-width 0.22s ease,
    opacity 0.14s ease,
    transform 0.14s ease;
  max-width: 12rem;
}

.sidebar-brand-collapsed {
  max-width: 0;
  overflow: hidden;
  opacity: 0;
  transform: translateX(-4px);
  pointer-events: none;
}

.sidebar-brand-title {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-version-row {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  min-width: 0;
}

.sidebar-version-product {
  color: rgb(107 114 128);
  font-size: 0.75rem;
  font-weight: 300;
  line-height: 1rem;
}

.dark .sidebar-version-product {
  color: rgb(156 163 175);
}

.sidebar-footer-controls {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.sidebar-footer-controls-collapsed {
  grid-template-columns: 1fr;
}

.sidebar-footer-action {
  justify-content: flex-start;
  min-width: 0;
}

.sidebar-footer-label {
  margin-left: 0.25rem;
}

.sidebar-link-collapsed {
  gap: 0;
  justify-content: center;
  overflow: visible;
  padding-left: 0;
  padding-right: 0;
}

.sidebar-section-title {
  position: relative;
  display: flex;
  align-items: center;
  min-height: 1.25rem;
  overflow: hidden;
  white-space: nowrap;
}

.sidebar-section-title-text {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    opacity 0.16s ease,
    transform 0.16s ease;
}

.sidebar-section-title::after {
  content: '';
  position: absolute;
  left: 0.75rem;
  right: 0.75rem;
  top: 50%;
  height: 1px;
  background: rgb(229 231 235);
  opacity: 0;
  transform: translateY(-50%);
  transition: opacity 0.18s ease;
}

.dark .sidebar-section-title::after {
  background: rgb(55 65 81);
}

.sidebar-section-title-text-collapsed {
  opacity: 0;
  transform: translateX(-4px);
}

.sidebar-section-title-collapsed::after {
  opacity: 1;
  transition-delay: 0.08s;
}

.sidebar-label {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition:
    max-width 0.2s ease,
    opacity 0.12s ease,
    transform 0.12s ease;
  max-width: 12rem;
}

.sidebar-label-flex {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.sidebar-label-collapsed {
  max-width: 0;
  opacity: 0;
  transform: translateX(-4px);
  pointer-events: none;
}

/* Custom SVG icon in sidebar: constrain size without overriding uploaded SVG colors */
.sidebar-svg-icon {
  color: currentColor;
}

.sidebar-svg-icon :deep(svg) {
  display: block;
  width: 1.25rem;
  height: 1.25rem;
}
</style>
