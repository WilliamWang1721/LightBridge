<template>
  <AppLayout>
    <ReactPageHost
      :load="loadChannelStatusPage"
      :props="pageProps"
      :error-message="t('common.error')"
    >
      <template #fallback>
    <MonitorHero
      :overall-status="overallStatus"
      :interval-seconds="DEFAULT_INTERVAL_SECONDS"
      :window="currentWindow"
      :loading="loading"
      :auto-refresh="autoRefresh"
      @update:window="handleWindowChange"
      @refresh="manualReload"
    />

    <MonitorCardGrid
      :items="items"
      :window="currentWindow"
      :countdown-seconds="countdown"
      :loading="loading"
      :detail-cache="detailCache"
      @card-click="openDetail"
    />

    <MonitorDetailDialog
      :show="showDetail"
      :monitor-id="detailTarget?.id ?? null"
      :title="detailTitle"
      @close="closeDetail"
    />
      </template>
    </ReactPageHost>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import {
  list as listChannelMonitorViews,
  status as fetchChannelMonitorDetail,
  type UserMonitorView,
  type UserMonitorDetail,
} from '@/api/channelMonitor'
import AppLayout from '@/components/layout/AppLayout.vue'
import ReactPageHost from '@/console/ReactPageHost.vue'
import type { ChannelStatusPageProps } from '@/console/react/ChannelStatusPage'
import MonitorHero, {
  type MonitorWindow,
  type OverallStatus,
} from '@/components/user/monitor/MonitorHero.vue'
import MonitorCardGrid from '@/components/user/monitor/MonitorCardGrid.vue'
import MonitorDetailDialog from '@/components/user/MonitorDetailDialog.vue'
import { DEFAULT_INTERVAL_SECONDS, STATUS_OPERATIONAL } from '@/constants/channelMonitor'
import { useAutoRefresh } from '@/composables/useAutoRefresh'

const { t } = useI18n()
const appStore = useAppStore()

// ── State ──
const items = ref<UserMonitorView[]>([])
const loading = ref(false)
const currentWindow = ref<MonitorWindow>('7d')
const detailCache = reactive<Record<number, UserMonitorDetail>>({})
const showDetail = ref(false)
const detailTarget = ref<UserMonitorView | null>(null)

let abortController: AbortController | null = null

const autoRefresh = useAutoRefresh({
  storageKey: 'channel-status-auto-refresh',
  intervals: [30, 60, 120] as const,
  defaultInterval: DEFAULT_INTERVAL_SECONDS,
  onRefresh: () => reload(true),
  shouldPause: () => document.hidden || loading.value,
})
const countdown = autoRefresh.countdown

// ── Computed ──
const overallStatus = computed<OverallStatus>(() => {
  if (items.value.length === 0) return 'operational'
  for (const it of items.value) {
    if (it.primary_status === 'failed' || it.primary_status === 'error') return 'degraded'
    if (it.primary_status !== STATUS_OPERATIONAL) return 'degraded'
  }
  return 'operational'
})

const detailTitle = computed(() => {
  return detailTarget.value?.name || t('channelStatus.detailTitle')
})

const loadChannelStatusPage = () => import('@/console/react/ChannelStatusPage')

// ── Loaders ──
async function reload(silent = false) {
  if (abortController) abortController.abort()
  const ctrl = new AbortController()
  abortController = ctrl
  if (!silent) loading.value = true
  try {
    const res = await listChannelMonitorViews({ signal: ctrl.signal })
    if (ctrl.signal.aborted || abortController !== ctrl) return
    items.value = res.items || []
  } catch (err: unknown) {
    const e = err as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('channelStatus.loadError')))
  } finally {
    if (abortController === ctrl) {
      if (!silent) loading.value = false
      countdown.value = DEFAULT_INTERVAL_SECONDS
      abortController = null
    }
  }
}

async function manualReload() {
  await reload(false)
  // After base reload, refresh any cached detail records so non-7d availability
  // values stay in sync without forcing the user to switch tabs again.
  if (currentWindow.value !== '7d') {
    await Promise.all(items.value.map(it => loadDetail(it.id, true)))
  }
}

async function loadDetail(id: number, force = false) {
  if (!force && detailCache[id]) return
  try {
    detailCache[id] = await fetchChannelMonitorDetail(id)
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('channelStatus.detailLoadError')))
  }
}

async function ensureDetailsForWindow() {
  if (currentWindow.value === '7d') return
  await Promise.all(items.value.map(it => loadDetail(it.id)))
}

// ── Handlers ──
async function handleWindowChange(value: MonitorWindow) {
  currentWindow.value = value
  await ensureDetailsForWindow()
}

function openDetail(row: UserMonitorView) {
  detailTarget.value = row
  showDetail.value = true
}

function closeDetail() {
  showDetail.value = false
  detailTarget.value = null
}

const pageProps = computed<ChannelStatusPageProps>(() => ({
  items: items.value,
  loading: loading.value,
  overallStatus: overallStatus.value,
  currentWindow: currentWindow.value,
  countdown: countdown.value,
  detailCache,
  autoRefresh: {
    enabled: autoRefresh.enabled.value,
    intervalSeconds: autoRefresh.intervalSeconds.value,
    intervals: autoRefresh.intervals,
  },
  copy: {
    windowTab: {
      '7d': t('channelStatus.windowTab.7d'),
      '15d': t('channelStatus.windowTab.15d'),
      '30d': t('channelStatus.windowTab.30d'),
    },
    overall: {
      operational: t('channelStatus.overall.operational'),
      degraded: t('channelStatus.overall.degraded'),
    },
    status: {
      operational: t('monitorCommon.status.operational'),
      degraded: t('monitorCommon.status.degraded'),
      failed: t('monitorCommon.status.failed'),
      error: t('monitorCommon.status.error'),
      unknown: t('monitorCommon.status.unknown'),
    },
    providers: {
      openai: t('monitorCommon.providers.openai'),
      anthropic: t('monitorCommon.providers.anthropic'),
      gemini: t('monitorCommon.providers.gemini'),
    },
    refresh: t('common.refresh'),
    detailTitle: t('channelStatus.detailTitle'),
    closeDetail: t('channelStatus.closeDetail'),
    loading: t('common.loading'),
    detailLoadError: t('channelStatus.detailLoadError'),
    emptyTitle: t('channelStatus.empty.title'),
    emptyDescription: t('channelStatus.empty.description'),
    dialogLatency: t('monitorCommon.dialogLatency'),
    endpointPing: t('monitorCommon.endpointPing'),
    availability: t('monitorCommon.availabilityPrefix'),
    extraModelsCount: (count) => t('monitorCommon.extraModelsCount', { n: count }),
    nextUpdateIn: (count) => t('monitorCommon.nextUpdateIn', { n: count }),
    pollEvery: (seconds) => t('monitorCommon.pollEvery', { n: seconds }),
    detailColumns: {
      model: t('channelStatus.detailColumns.model'),
      latestStatus: t('channelStatus.detailColumns.latestStatus'),
      latestLatency: t('channelStatus.detailColumns.latestLatency'),
      availability7d: t('channelStatus.detailColumns.availability7d'),
      availability15d: t('channelStatus.detailColumns.availability15d'),
      availability30d: t('channelStatus.detailColumns.availability30d'),
      avgLatency7d: t('channelStatus.detailColumns.avgLatency7d'),
    },
  },
  onWindowChange: (value) => { void handleWindowChange(value) },
  onRefresh: () => { void manualReload() },
  onAutoRefreshChange: (enabled) => autoRefresh.setEnabled(enabled),
  onIntervalChange: (seconds) => autoRefresh.setInterval(seconds),
  onCardClick: openDetail,
  onLoadDetail: (id) => loadDetail(id),
}))

watch(items, () => {
  void ensureDetailsForWindow()
})

watch(
  () => appStore.cachedPublicSettings?.channel_monitor_enabled,
  (enabled) => {
    if (enabled === false) autoRefresh.stop()
    else if (autoRefresh.enabled.value) autoRefresh.start()
  },
)

onMounted(() => {
  void reload(false)
  if (appStore.cachedPublicSettings?.channel_monitor_enabled !== false) {
    autoRefresh.setEnabled(true)
  }
})

onBeforeUnmount(() => {
  if (abortController) abortController.abort()
})
</script>
