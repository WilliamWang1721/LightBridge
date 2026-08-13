<template>
  <AppLayout
    @refresh="loadDashboardStats"
    @customize-dashboard="showCustomizePanel = true"
  >
    <ReactPageHost
      :load="loadDashboardPage"
      :props="dashboardPageProps"
      :error-message="t('common.error')"
    />
    <DashboardCustomizePanel
      v-model:enabled-small-keys="enabledSmallPanelKeys"
      v-model:enabled-large-keys="enabledLargePanelKeys"
      :show="showCustomizePanel"
      :small-panels="dashboardSmallPanels"
      :large-panels="dashboardLargePanels"
      :small-limit="MAX_SMALL_PANELS"
      @reset="resetDashboardLayout"
      @close="showCustomizePanel = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { useTimeRangeStore } from '@/stores/timeRange'

import { adminAPI } from '@/api/admin'
import type {
  DashboardStats,
  TrendDataPoint,
  ModelStat,
  UserUsageTrendPoint,
  UserSpendingRankingItem
} from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import ReactPageHost from '@/console/ReactPageHost.vue'
import type { DashboardStatsPageProps } from '@/console/react/DashboardStats'
import type { ModelDistributionPageProps } from '@/console/react/ModelDistribution'
import type { DashboardPageProps } from '@/console/react/DashboardPage'
import DashboardCustomizePanel from '@/components/admin/dashboard/DashboardCustomizePanel.vue'

const appStore = useAppStore()
const router = useRouter()
const { t } = useI18n()
const stats = ref<DashboardStats | null>(null)
const loading = ref(false)
const chartsLoading = ref(false)
const userTrendLoading = ref(false)
const rankingLoading = ref(false)
const rankingError = ref(false)

// Chart data
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const userTrend = ref<UserUsageTrendPoint[]>([])
const rankingItems = ref<UserSpendingRankingItem[]>([])
const rankingTotalActualCost = ref(0)
const rankingTotalRequests = ref(0)
const rankingTotalTokens = ref(0)
let chartLoadSeq = 0
let usersTrendLoadSeq = 0
let rankingLoadSeq = 0
const rankingLimit = 12
const DASHBOARD_LAYOUT_STORAGE_KEY = 'lb-admin-dashboard-layout-v1'
const MAX_SMALL_PANELS = 16

interface DashboardPanelOption {
  key: string
  labelKey: string
}

interface DashboardLayoutPreference {
  small: string[]
  large: string[]
}

const dashboardSmallPanels: DashboardPanelOption[] = [
  { key: 'apiKeys', labelKey: 'admin.dashboard.apiKeys' },
  { key: 'accounts', labelKey: 'admin.dashboard.accounts' },
  { key: 'todayRequests', labelKey: 'admin.dashboard.todayRequests' },
  { key: 'users', labelKey: 'admin.dashboard.users' },
  { key: 'todayTokens', labelKey: 'admin.dashboard.todayTokens' },
  { key: 'totalTokens', labelKey: 'admin.dashboard.totalTokens' },
  { key: 'performance', labelKey: 'admin.dashboard.performance' },
  { key: 'avgResponse', labelKey: 'admin.dashboard.avgResponse' }
]

const dashboardLargePanels: DashboardPanelOption[] = [
  { key: 'usageCharts', labelKey: 'admin.dashboard.customize.usageCharts' },
  { key: 'userTrend', labelKey: 'admin.dashboard.userUsageTrend' }
]

const defaultDashboardLayout: DashboardLayoutPreference = {
  small: dashboardSmallPanels.map((panel) => panel.key),
  large: dashboardLargePanels.map((panel) => panel.key)
}

const initialDashboardLayout = loadDashboardLayout()
const enabledSmallPanelKeys = ref<string[]>(initialDashboardLayout.small)
const enabledLargePanelKeys = ref<string[]>(initialDashboardLayout.large)
const showCustomizePanel = ref(false)
const enabledSmallPanels = computed(() => panelsForKeys(enabledSmallPanelKeys.value, dashboardSmallPanels))

// Date range
// 时间范围 / 颗粒度由顶部菜单栏的全局 store 驱动
const timeRangeStore = useTimeRangeStore()
const granularity = computed<'day' | 'hour'>({
  get: () => timeRangeStore.granularity,
  set: (v) => timeRangeStore.setGranularity(v)
})
const startDate = computed(() => timeRangeStore.startDate)
const endDate = computed(() => timeRangeStore.endDate)

// 监听 store 变化，自动重新加载图表数据
watch(
  [() => timeRangeStore.startDate, () => timeRangeStore.endDate, () => timeRangeStore.granularity],
  () => {
    loadDashboardStats()
  }
)

watch(
  [enabledSmallPanelKeys, enabledLargePanelKeys],
  () => {
    saveDashboardLayout()
  },
  { deep: true }
)

function panelsForKeys(keys: string[], panels: DashboardPanelOption[]): DashboardPanelOption[] {
  const panelMap = new Map(panels.map((panel) => [panel.key, panel]))
  return keys
    .map((key) => panelMap.get(key))
    .filter((panel): panel is DashboardPanelOption => Boolean(panel))
}

function sanitizePanelKeys(
  value: unknown,
  panels: DashboardPanelOption[],
  limit?: number
): string[] {
  if (!Array.isArray(value)) return []
  const availableKeys = new Set(panels.map((panel) => panel.key))
  const seenKeys = new Set<string>()
  const next: string[] = []

  value.forEach((item) => {
    if (typeof item !== 'string' || !availableKeys.has(item) || seenKeys.has(item)) return
    seenKeys.add(item)
    next.push(item)
  })

  return typeof limit === 'number' ? next.slice(0, limit) : next
}

function createDefaultDashboardLayout(): DashboardLayoutPreference {
  return {
    small: [...defaultDashboardLayout.small],
    large: [...defaultDashboardLayout.large]
  }
}

function loadDashboardLayout(): DashboardLayoutPreference {
  try {
    const raw = localStorage.getItem(DASHBOARD_LAYOUT_STORAGE_KEY)
    if (!raw) return createDefaultDashboardLayout()
    const parsed = JSON.parse(raw) as Partial<DashboardLayoutPreference>
    return {
      small: Array.isArray(parsed.small)
        ? sanitizePanelKeys(parsed.small, dashboardSmallPanels, MAX_SMALL_PANELS)
        : [...defaultDashboardLayout.small],
      large: Array.isArray(parsed.large)
        ? sanitizePanelKeys(parsed.large, dashboardLargePanels)
        : [...defaultDashboardLayout.large]
    }
  } catch {
    return createDefaultDashboardLayout()
  }
}

function saveDashboardLayout() {
  try {
    localStorage.setItem(
      DASHBOARD_LAYOUT_STORAGE_KEY,
      JSON.stringify({
        small: enabledSmallPanelKeys.value,
        large: enabledLargePanelKeys.value
      })
    )
  } catch {
    /* ignore persistence failures */
  }
}

function resetDashboardLayout() {
  enabledSmallPanelKeys.value = [...defaultDashboardLayout.small]
  enabledLargePanelKeys.value = [...defaultDashboardLayout.large]
}

// Format helpers
const formatTokens = (value: number | undefined): string => {
  if (value === undefined || value === null) return '0'
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const formatNumber = (value: number): string => {
  return value.toLocaleString()
}

const formatCost = (value: number | null | undefined): string => {
  const normalized = typeof value === 'number' && Number.isFinite(value) ? value : 0
  if (normalized >= 1000) {
    return (normalized / 1000).toFixed(2) + 'K'
  } else if (normalized >= 1) {
    return normalized.toFixed(2)
  } else if (normalized >= 0.01) {
    return normalized.toFixed(3)
  }
  return normalized.toFixed(4)
}

const formatDuration = (ms: number): string => {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`
  }
  return `${Math.round(ms)}ms`
}

const loadDashboardPage = () => import('@/console/react/DashboardPage')

const dashboardStatsPageProps = computed<DashboardStatsPageProps>(() => {
  const current = stats.value
  if (!current) {
    return {
      cards: enabledSmallPanels.value.map((panel, order) => ({
        key: panel.key,
        label: t(panel.labelKey),
        value: '—',
        order
      }))
    }
  }

  const cards: DashboardStatsPageProps['cards'][number][] = []
  for (const panel of enabledSmallPanels.value) {
    const order = enabledSmallPanelKeys.value.indexOf(panel.key)
    const base = { key: panel.key, label: t(panel.labelKey), order: order < 0 ? 0 : order }

    if (panel.key === 'apiKeys') {
      cards.push({ ...base, value: current.total_api_keys, hint: `${current.active_api_keys} ${t('common.active')}` })
    } else if (panel.key === 'accounts') {
      cards.push({
        ...base,
        value: current.total_accounts,
        meta: {
          text: `${current.normal_accounts} ${t('common.active')}`,
          alert: current.error_accounts > 0 ? `${current.error_accounts} ${t('common.error')}` : undefined,
        },
      })
    } else if (panel.key === 'todayRequests') {
      cards.push({ ...base, value: current.today_requests, hint: `${t('common.total')}: ${formatNumber(current.total_requests)}` })
    } else if (panel.key === 'users') {
      cards.push({ ...base, value: `+${current.today_new_users}`, hint: `${t('common.total')}: ${formatNumber(current.total_users)}` })
    } else if (panel.key === 'todayTokens') {
      cards.push({
        ...base,
        value: formatTokens(current.today_tokens),
        meta: { text: `$${formatCost(current.today_actual_cost)} / $${formatCost(current.today_account_cost)} / $${formatCost(current.today_cost)}` },
      })
    } else if (panel.key === 'totalTokens') {
      cards.push({
        ...base,
        value: formatTokens(current.total_tokens),
        meta: { text: `$${formatCost(current.total_actual_cost)} / $${formatCost(current.total_account_cost)} / $${formatCost(current.total_cost)}` },
      })
    } else if (panel.key === 'performance') {
      cards.push({ ...base, value: `${formatTokens(current.rpm)} RPM`, hint: `${formatTokens(current.tpm)} TPM` })
    } else if (panel.key === 'avgResponse') {
      cards.push({ ...base, value: formatDuration(current.average_duration_ms), hint: `${current.active_users} ${t('admin.dashboard.activeUsers')}` })
    }
  }

  return { cards }
})

const tokenUsageTrendPageProps = computed(() => ({
  trendData: trendData.value,
  loading: chartsLoading.value,
  title: t('admin.dashboard.tokenUsageTrend'),
  noData: t('admin.dashboard.noDataAvailable'),
}))

const goToUserUsage = (item: UserSpendingRankingItem) => {
  void router.push({
    path: '/admin/usage',
    query: {
      user_id: String(item.user_id),
      start_date: startDate.value,
      end_date: endDate.value
    }
  })
}

const modelDistributionPageProps = computed<ModelDistributionPageProps>(() => ({
  modelStats: modelStats.value,
  rankingItems: rankingItems.value,
  rankingTotalActualCost: rankingTotalActualCost.value,
  rankingTotalRequests: rankingTotalRequests.value,
  rankingTotalTokens: rankingTotalTokens.value,
  loading: chartsLoading.value,
  rankingLoading: rankingLoading.value,
  rankingError: rankingError.value,
  startDate: startDate.value,
  endDate: endDate.value,
  labels: {
    modelDistribution: t('admin.dashboard.modelDistribution'),
    spendingRankingTitle: t('admin.dashboard.spendingRankingTitle'),
    viewModelDistribution: t('admin.dashboard.viewModelDistribution'),
    viewSpendingRanking: t('admin.dashboard.viewSpendingRanking'),
    spendingRankingUser: t('admin.dashboard.spendingRankingUser'),
    spendingRankingRequests: t('admin.dashboard.spendingRankingRequests'),
    spendingRankingTokens: t('admin.dashboard.spendingRankingTokens'),
    spendingRankingSpend: t('admin.dashboard.spendingRankingSpend'),
    spendingRankingOther: t('admin.dashboard.spendingRankingOther'),
    model: t('admin.dashboard.model'),
    requests: t('admin.dashboard.requests'),
    tokens: t('admin.dashboard.tokens'),
    actual: t('admin.dashboard.actual'),
    accountCost: t('admin.dashboard.accountCost'),
    standard: t('admin.dashboard.standard'),
    noData: t('admin.dashboard.noDataAvailable'),
    failedToLoad: t('admin.dashboard.failedToLoad'),
    userPrefix: (id: number) => t('admin.redeem.userPrefix', { id }),
  },
  onRankingClick: goToUserUsage,
}))

const dashboardPageProps = computed<DashboardPageProps>(() => ({
  loading: loading.value,
  cards: dashboardStatsPageProps.value.cards,
  modelDistribution: modelDistributionPageProps.value,
  tokenUsageTrend: tokenUsageTrendPageProps.value,
  userTrend: userTrend.value,
  userTrendLoading: userTrendLoading.value,
  labels: {
    recentUsage: t('admin.dashboard.recentUsage'),
    userUsageTrend: t('admin.dashboard.userUsageTrend'),
    requests: t('admin.dashboard.requests'),
    tokens: t('admin.dashboard.tokens'),
    noData: t('admin.dashboard.noDataAvailable'),
    userPrefix: (id: number) => t('admin.redeem.userPrefix', { id }),
  },
}))

// Load data
const loadDashboardSnapshot = async (includeStats: boolean) => {
  const currentSeq = ++chartLoadSeq
  if (includeStats && !stats.value) {
    loading.value = true
  }
  chartsLoading.value = true
  try {
    const response = await adminAPI.dashboard.getSnapshotV2({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      include_stats: includeStats,
      include_trend: true,
      include_model_stats: true,
      include_group_stats: false,
      include_users_trend: false
    })
    if (currentSeq !== chartLoadSeq) return
    if (includeStats && response.stats) {
      stats.value = response.stats
    }
    trendData.value = response.trend || []
    modelStats.value = response.models || []
  } catch (error) {
    if (currentSeq !== chartLoadSeq) return
    appStore.showError(t('admin.dashboard.failedToLoad'))
    console.error('Error loading dashboard snapshot:', error)
  } finally {
    if (currentSeq === chartLoadSeq) {
      loading.value = false
      chartsLoading.value = false
    }
  }
}

const loadUsersTrend = async () => {
  const currentSeq = ++usersTrendLoadSeq
  userTrendLoading.value = true
  try {
    const response = await adminAPI.dashboard.getUserUsageTrend({
      start_date: startDate.value,
      end_date: endDate.value,
      granularity: granularity.value,
      limit: 12
    })
    if (currentSeq !== usersTrendLoadSeq) return
    userTrend.value = response.trend || []
  } catch (error) {
    if (currentSeq !== usersTrendLoadSeq) return
    console.error('Error loading users trend:', error)
    userTrend.value = []
  } finally {
    if (currentSeq === usersTrendLoadSeq) {
      userTrendLoading.value = false
    }
  }
}

const loadUserSpendingRanking = async () => {
  const currentSeq = ++rankingLoadSeq
  rankingLoading.value = true
  rankingError.value = false
  try {
    const response = await adminAPI.dashboard.getUserSpendingRanking({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: rankingLimit
    })
    if (currentSeq !== rankingLoadSeq) return
    rankingItems.value = response.ranking || []
    rankingTotalActualCost.value = response.total_actual_cost || 0
    rankingTotalRequests.value = response.total_requests || 0
    rankingTotalTokens.value = response.total_tokens || 0
  } catch (error) {
    if (currentSeq !== rankingLoadSeq) return
    console.error('Error loading user spending ranking:', error)
    rankingItems.value = []
    rankingTotalActualCost.value = 0
    rankingTotalRequests.value = 0
    rankingTotalTokens.value = 0
    rankingError.value = true
  } finally {
    if (currentSeq === rankingLoadSeq) {
      rankingLoading.value = false
    }
  }
}

const loadDashboardStats = async () => {
  await Promise.all([
    loadDashboardSnapshot(true),
    loadUsersTrend(),
    loadUserSpendingRanking()
  ])
}

onMounted(() => {
  loadDashboardStats()
})
</script>

<style scoped>
</style>
