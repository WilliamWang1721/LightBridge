<template>
  <AppLayout>
    <div class="min-w-0 space-y-5 pb-4 md:space-y-6">
      <div v-if="loading" class="flex items-center justify-center py-12"><LoadingSpinner /></div>
      <template v-else-if="stats">
        <ReactPageHost
          :load="loadUserDashboardPage"
          :props="pageProps"
          :error-message="t('common.error')"
        >
          <template #fallback>
            <UserDashboardStats :stats="stats" :balance="user?.balance || 0" :is-simple="authStore.isSimpleMode" :platform-quotas="platformQuotas" />
            <UserDashboardCharts v-model:startDate="startDate" v-model:endDate="endDate" v-model:granularity="granularity" :loading="loadingCharts" :trend="trendData" :models="modelStats" @dateRangeChange="loadCharts" @granularityChange="loadCharts" @refresh="refreshAll" />
            <div class="grid min-w-0 grid-cols-1 gap-5 xl:grid-cols-3">
              <div class="min-w-0 xl:col-span-2"><UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" /></div>
              <div class="min-w-0"><UserDashboardQuickActions /></div>
            </div>
          </template>
        </ReactPageHost>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'; import { useAuthStore } from '@/stores/auth'; import { useRouter } from 'vue-router'; import { useI18n } from 'vue-i18n'; import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'; import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ReactPageHost from '@/console/ReactPageHost.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'; import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'; import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { UsageLog, TrendDataPoint, ModelStat, PlatformQuotaItem } from '@/types'
import { getMyPlatformQuotas } from '@/api/user'
import type { UserDashboardPageProps } from '@/console/react/UserDashboardPage'

const authStore = useAuthStore(); const router = useRouter(); const user = computed(() => authStore.user)
const { t } = useI18n()
const stats = ref<UserStatsType | null>(null); const loading = ref(false); const loadingUsage = ref(false); const loadingCharts = ref(false)
const trendData = ref<TrendDataPoint[]>([]); const modelStats = ref<ModelStat[]>([]); const recentUsage = ref<UsageLog[]>([])
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)

const loadUserDashboardPage = () => import('@/console/react/UserDashboardPage')

const formatLD = (d: Date) => d.toISOString().split('T')[0]
const startDate = ref(formatLD(new Date(Date.now() - 6 * 86400000))); const endDate = ref(formatLD(new Date())); const granularity = ref('day')

const loadStats = async () => { loading.value = true; try { await authStore.refreshUser(); stats.value = await usageAPI.getDashboardStats() } catch (error) { console.error('Failed to load dashboard stats:', error) } finally { loading.value = false } }
const loadCharts = async () => { loadingCharts.value = true; try { const res = await Promise.all([usageAPI.getDashboardTrend({ start_date: startDate.value, end_date: endDate.value, granularity: granularity.value as any }), usageAPI.getDashboardModels({ start_date: startDate.value, end_date: endDate.value })]); trendData.value = res[0].trend || []; modelStats.value = res[1].models || [] } catch (error) { console.error('Failed to load charts:', error) } finally { loadingCharts.value = false } }
const loadRecent = async () => { loadingUsage.value = true; try { const res = await usageAPI.getByDateRange(startDate.value, endDate.value); recentUsage.value = res.items.slice(0, 5) } catch (error) { console.error('Failed to load recent usage:', error) } finally { loadingUsage.value = false } }
const loadPlatformQuotas = async () => { try { const data = await getMyPlatformQuotas(); platformQuotas.value = data.platform_quotas ?? [] } catch (error) { console.warn('Failed to load platform quotas:', error); platformQuotas.value = [] } }
const refreshAll = () => { loadStats(); loadCharts(); loadRecent(); loadPlatformQuotas() }

const pageProps = computed<UserDashboardPageProps>(() => ({
  stats: stats.value,
  balance: user.value?.balance || 0,
  isSimple: authStore.isSimpleMode,
  loading: loading.value,
  loadingUsage: loadingUsage.value,
  loadingCharts: loadingCharts.value,
  trend: trendData.value,
  models: modelStats.value,
  recentUsage: recentUsage.value,
  platformQuotas: platformQuotas.value,
  startDate: startDate.value,
  endDate: endDate.value,
  granularity: granularity.value,
  copy: {
    refresh: t('common.refresh'),
    balance: t('dashboard.balance'),
    available: t('common.available'),
    apiKeys: t('dashboard.apiKeys'),
    active: t('common.active'),
    todayRequests: t('dashboard.todayRequests'),
    total: t('common.total'),
    todayCost: t('dashboard.todayCost'),
    totalTokens: t('dashboard.totalTokens'),
    todayTokens: t('dashboard.todayTokens'),
    input: t('dashboard.input'),
    output: t('dashboard.output'),
    performance: t('dashboard.performance'),
    avgResponse: t('dashboard.avgResponse'),
    modelDistribution: t('dashboard.modelDistribution'),
    tokenUsageTrend: t('dashboard.tokenUsageTrend'),
    model: t('dashboard.model'),
    requests: t('dashboard.requests'),
    tokens: t('dashboard.tokens'),
    actual: t('dashboard.actual'),
    standard: t('dashboard.standard'),
    noData: t('dashboard.noDataAvailable'),
    timeRange: t('dashboard.timeRange'),
    granularity: t('dashboard.granularity'),
    day: t('dashboard.day'),
    hour: t('dashboard.hour'),
    recentUsage: t('dashboard.recentUsage'),
    last7Days: t('dashboard.last7Days'),
    noUsageRecords: t('dashboard.noUsageRecords'),
    startUsingApi: t('dashboard.startUsingApi'),
    viewAllUsage: t('dashboard.viewAllUsage'),
    quickActions: t('dashboard.quickActions'),
    createApiKey: t('dashboard.createApiKey'),
    generateNewKey: t('dashboard.generateNewKey'),
    viewUsage: t('dashboard.viewUsage'),
    checkDetailedLogs: t('dashboard.checkDetailedLogs'),
    redeemCode: t('dashboard.redeemCode'),
    addBalanceWithCode: t('dashboard.addBalanceWithCode'),
    platformBreakdown: t('dashboard.platformBreakdown'),
    platformBreakdownEmpty: t('dashboard.platformBreakdownEmpty'),
    platformCount: (count) => t('dashboard.platformCount', { count }),
    platformOther: t('dashboard.platformOther'),
    platformQuota: {
      title: t('dashboard.platformQuota.title'),
      daily: t('dashboard.platformQuota.daily'),
      weekly: t('dashboard.platformQuota.weekly'),
      monthly: t('dashboard.platformQuota.monthly'),
      resetsAt: (time) => t('dashboard.platformQuota.resetsAt', { time }),
      disabled: t('dashboard.platformQuota.disabled'),
    },
  },
  onDateChange: (key, value) => {
    if (key === 'startDate') startDate.value = value
    else endDate.value = value
    void loadCharts()
  },
  onGranularityChange: (value) => { granularity.value = value; void loadCharts() },
  onRefresh: refreshAll,
  onNavigate: (path) => { void router.push(path) },
}))

onMounted(() => { refreshAll() })
</script>
