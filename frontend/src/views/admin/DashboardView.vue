<template>
  <AppLayout
    @refresh="loadDashboardStats"
    @customize-dashboard="showCustomizePanel = true"
  >
    <div class="min-w-0 space-y-5 pb-4 md:space-y-6">
      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else-if="stats">
        <!-- Small Panels -->
        <div v-if="enabledSmallPanels.length > 0" class="grid min-w-0 grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <DashboardStatCard
            v-if="isSmallPanelEnabled('apiKeys')"
            icon="key"
            :label="t('admin.dashboard.apiKeys')"
            :value="stats.total_api_keys"
            :hint="`${stats.active_api_keys} ${t('common.active')}`"
            :style="smallPanelOrderStyle('apiKeys')"
          />
          <DashboardStatCard
            v-if="isSmallPanelEnabled('accounts')"
            icon="server"
            :label="t('admin.dashboard.accounts')"
            :value="stats.total_accounts"
            :style="smallPanelOrderStyle('accounts')"
          >
            <template #meta>
              <p class="mt-1 text-xs text-[hsl(var(--muted-foreground))]">
                {{ stats.normal_accounts }} {{ t('common.active') }}
                <span v-if="stats.error_accounts > 0" class="ml-1 text-red-600 dark:text-red-400">
                  {{ stats.error_accounts }} {{ t('common.error') }}
                </span>
              </p>
            </template>
          </DashboardStatCard>
          <DashboardStatCard
            v-if="isSmallPanelEnabled('todayRequests')"
            icon="chart"
            :label="t('admin.dashboard.todayRequests')"
            :value="stats.today_requests"
            :hint="`${t('common.total')}: ${formatNumber(stats.total_requests)}`"
            :style="smallPanelOrderStyle('todayRequests')"
          />
          <DashboardStatCard
            v-if="isSmallPanelEnabled('users')"
            icon="userPlus"
            :label="t('admin.dashboard.users')"
            :value="`+${stats.today_new_users}`"
            :hint="`${t('common.total')}: ${formatNumber(stats.total_users)}`"
            :style="smallPanelOrderStyle('users')"
          />
          <DashboardStatCard
            v-if="isSmallPanelEnabled('todayTokens')"
            icon="cube"
            :label="t('admin.dashboard.todayTokens')"
            :value="formatTokens(stats.today_tokens)"
            :style="smallPanelOrderStyle('todayTokens')"
          >
            <template #meta>
              <p class="mt-1 text-xs text-[hsl(var(--muted-foreground))]">
                ${{ formatCost(stats.today_actual_cost) }} / ${{ formatCost(stats.today_account_cost) }} / ${{ formatCost(stats.today_cost) }}
              </p>
            </template>
          </DashboardStatCard>
          <DashboardStatCard
            v-if="isSmallPanelEnabled('totalTokens')"
            icon="database"
            :label="t('admin.dashboard.totalTokens')"
            :value="formatTokens(stats.total_tokens)"
            :style="smallPanelOrderStyle('totalTokens')"
          >
            <template #meta>
              <p class="mt-1 text-xs text-[hsl(var(--muted-foreground))]">
                ${{ formatCost(stats.total_actual_cost) }} / ${{ formatCost(stats.total_account_cost) }} / ${{ formatCost(stats.total_cost) }}
              </p>
            </template>
          </DashboardStatCard>
          <DashboardStatCard
            v-if="isSmallPanelEnabled('performance')"
            icon="bolt"
            :label="t('admin.dashboard.performance')"
            :value="`${formatTokens(stats.rpm)} RPM`"
            :hint="`${formatTokens(stats.tpm)} TPM`"
            :style="smallPanelOrderStyle('performance')"
          />
          <DashboardStatCard
            v-if="isSmallPanelEnabled('avgResponse')"
            icon="clock"
            :label="t('admin.dashboard.avgResponse')"
            :value="formatDuration(stats.average_duration_ms)"
            :hint="`${stats.active_users} ${t('admin.dashboard.activeUsers')}`"
            :style="smallPanelOrderStyle('avgResponse')"
          />
        </div>

        <!-- Large Panels -->
        <div v-if="enabledLargePanels.length > 0" class="flex min-w-0 flex-col gap-5 md:gap-6">
          <template v-for="panel in enabledLargePanels" :key="panel.key">
            <!-- Charts Grid -->
            <div v-if="panel.key === 'usageCharts'" class="grid min-w-0 grid-cols-1 gap-5 xl:grid-cols-2">
              <ModelDistributionChart
                :model-stats="modelStats"
                :enable-ranking-view="true"
                :ranking-items="rankingItems"
                :ranking-total-actual-cost="rankingTotalActualCost"
                :ranking-total-requests="rankingTotalRequests"
                :ranking-total-tokens="rankingTotalTokens"
                :loading="chartsLoading"
                :ranking-loading="rankingLoading"
                :ranking-error="rankingError"
                :start-date="startDate"
                :end-date="endDate"
                @ranking-click="goToUserUsage"
              />
              <TokenUsageTrend :trend-data="trendData" :loading="chartsLoading" />
            </div>

            <!-- User Usage Trend (Full Width) -->
            <Card v-else-if="panel.key === 'userTrend'" class="p-4">
              <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('admin.dashboard.recentUsage') }} (Top 12)
              </h3>
              <div class="h-64">
                <div v-if="userTrendLoading" class="flex h-full items-center justify-center">
                  <LoadingSpinner size="md" />
                </div>
                <Line v-else-if="userTrendChartData" :data="userTrendChartData" :options="lineOptions" />
                <div
                  v-else
                  class="flex h-full items-center justify-center text-sm text-gray-500 dark:text-gray-400"
                >
                  {{ t('admin.dashboard.noDataAvailable') }}
                </div>
              </div>
            </Card>
          </template>
        </div>
      </template>
    </div>
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
import { ref, computed, markRaw, onMounted, watch, type CSSProperties } from 'vue'
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
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import DashboardStatCard from '@/components/admin/dashboard/DashboardStatCard.vue'
import { Card } from '@/components/ui/card'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import DashboardCustomizePanel from '@/components/admin/dashboard/DashboardCustomizePanel.vue'

import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler
)

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
const enabledSmallPanelSet = computed(() => new Set(enabledSmallPanelKeys.value))
const enabledSmallPanels = computed(() => panelsForKeys(enabledSmallPanelKeys.value, dashboardSmallPanels))
const enabledLargePanels = computed(() => panelsForKeys(enabledLargePanelKeys.value, dashboardLargePanels))

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

function isSmallPanelEnabled(key: string): boolean {
  return enabledSmallPanelSet.value.has(key)
}

function smallPanelOrderStyle(key: string): CSSProperties {
  const order = enabledSmallPanelKeys.value.indexOf(key)
  return { order: order < 0 ? 0 : order }
}

// Dark mode detection
const isDarkMode = computed(() => {
  return document.documentElement.classList.contains('dark')
})

// Chart colors
const chartColors = computed(() => ({
  text: isDarkMode.value ? '#e5e7eb' : '#374151',
  grid: isDarkMode.value ? '#374151' : '#e5e7eb'
}))

// Line chart options (for user trend chart)
const lineOptions = computed(() => markRaw({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const,
      labels: {
        color: chartColors.value.text,
        usePointStyle: true,
        pointStyle: 'circle',
        padding: 15,
        font: {
          size: 11
        }
      }
    },
    tooltip: {
      itemSort: (a: any, b: any) => {
        const aValue = typeof a?.raw === 'number' ? a.raw : Number(a?.parsed?.y ?? 0)
        const bValue = typeof b?.raw === 'number' ? b.raw : Number(b?.parsed?.y ?? 0)
        return bValue - aValue
      },
      callbacks: {
        label: (context: any) => {
          return `${context.dataset.label}: ${formatTokens(context.raw)}`
        }
      }
    }
  },
  scales: {
    x: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        }
      }
    },
    y: {
      grid: {
        color: chartColors.value.grid
      },
      ticks: {
        color: chartColors.value.text,
        font: {
          size: 10
        },
        callback: (value: string | number) => formatTokens(Number(value))
      }
    }
  }
}))

// User trend chart data
const userTrendChartData = computed(() => {
  if (!userTrend.value?.length) return null

  const getDisplayName = (point: UserUsageTrendPoint): string => {
    const username = point.username?.trim()
    if (username) {
      return username
    }

    const email = point.email?.trim()
    if (email) {
      return email
    }

    return t('admin.redeem.userPrefix', { id: point.user_id })
  }

  // Group by user_id to avoid merging different users with the same display name
  const userGroups = new Map<number, { name: string; data: Map<string, number> }>()
  const allDates = new Set<string>()

  userTrend.value.forEach((point) => {
    allDates.add(point.date)
    const key = point.user_id
    if (!userGroups.has(key)) {
      userGroups.set(key, { name: getDisplayName(point), data: new Map() })
    }
    userGroups.get(key)!.data.set(point.date, point.tokens)
  })

  const sortedDates = Array.from(allDates).sort()
  const colors = [
    '#171717',
    '#404040',
    '#525252',
    '#737373',
    '#a3a3a3',
    '#d4d4d4',
    '#262626',
    '#666666',
    '#8a8a8a',
    '#b5b5b5',
    '#333333',
    '#e5e5e5'
  ]

  const datasets = Array.from(userGroups.values()).map((group, idx) => ({
    label: group.name,
    data: sortedDates.map((date) => group.data.get(date) || 0),
    borderColor: colors[idx % colors.length],
    backgroundColor: `${colors[idx % colors.length]}20`,
    fill: false,
    tension: 0.3
  }))

  return markRaw({
    labels: sortedDates,
    datasets
  })
})

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
