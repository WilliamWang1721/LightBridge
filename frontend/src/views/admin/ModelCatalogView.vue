<template>
  <AppLayout>
    <ReactPageHost
      :load="loadModelCatalogPage"
      :props="pageProps"
      :error-message="t('common.error')"
    >
      <template #fallback>
        <div class="mx-auto w-full max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
          <ModelCatalogPanel
            admin
            :models="models"
            :loading="loading"
            @refresh="loadCatalog"
            @quickSetup="openQuickSetup"
          />
        </div>
      </template>
    </ReactPageHost>

    <!-- 账号选择弹窗 -->
    <BaseDialog
      :show="showAccountPicker"
      :title="t('modelCatalog.selectAccountMonitor', { model: setupModel?.id })"
      width="normal"
      @close="showAccountPicker = false"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ t('modelCatalog.selectAccountHint') }}
        </p>
        <div class="space-y-2">
          <button
            v-for="source in setupModel?.sources"
            :key="source.account_id"
            type="button"
            class="flex w-full items-center justify-between rounded-lg border border-gray-200 p-3 text-left transition-colors hover:bg-gray-50 dark:border-dark-600 dark:hover:bg-dark-700"
            :class="{ 'border-primary-500 bg-primary-50 dark:border-primary-400 dark:bg-primary-900/20': selectedAccountId === source.account_id }"
            @click="selectedAccountId = source.account_id!"
          >
            <div>
              <div class="font-medium text-gray-900 dark:text-white">{{ source.account_name }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ source.platform }} · {{ source.source }}</div>
            </div>
            <Icon v-if="selectedAccountId === source.account_id" name="check" size="md" class="text-primary-500" />
          </button>
        </div>
        <div>
          <label class="input-label">{{ t('modelCatalog.quickMonitorInterval') }}</label>
          <select v-model.number="intervalSeconds" class="input w-full">
            <option :value="30">30s</option>
            <option :value="60">60s</option>
            <option :value="120">120s</option>
            <option :value="300">300s</option>
          </select>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="showAccountPicker = false">
            {{ t('common.cancel') }}
          </button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="!selectedAccountId"
            @click="openMonitorForm"
          >
            {{ t('modelCatalog.quickMonitorSubmit') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- 复用现有的监控表单 -->
    <MonitorFormDialog
      :show="showMonitorForm"
      :monitor="null"
      :prefill="monitorPrefill"
      @close="showMonitorForm = false"
      @saved="onMonitorSaved"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ReactPageHost from '@/console/ReactPageHost.vue'
import ModelCatalogPanel from '@/components/model-catalog/ModelCatalogPanel.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import MonitorFormDialog from '@/components/admin/monitor/MonitorFormDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { getAdminModelCatalog, type ModelCatalogModel } from '@/api/modelCatalog'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { Provider, APIMode } from '@/api/admin/channelMonitor'
import type { ModelCatalogPageProps } from '@/console/react/ModelCatalogPage'

const { t } = useI18n()
const appStore = useAppStore()

const models = ref<ModelCatalogModel[]>([])
const loading = ref(false)
const showAccountPicker = ref(false)
const showMonitorForm = ref(false)
const setupModel = ref<ModelCatalogModel | null>(null)
const selectedAccountId = ref<number | null>(null)
const intervalSeconds = ref(60)

const loadModelCatalogPage = () => import('@/console/react/ModelCatalogPage')

const pageProps = computed<ModelCatalogPageProps>(() => ({
  models: models.value,
  loading: loading.value,
  admin: true,
  copy: {
    refresh: t('common.refresh'),
    searchPlaceholder: t('modelCatalog.searchPlaceholder'),
    allGroups: t('modelCatalog.allGroups'),
    modelCount: (count) => t('modelCatalog.modelCount', { count }),
    sourceCount: (count) => t('modelCatalog.sourceCount', { count }),
    sourceDetails: t('modelCatalog.sourceDetails'),
    unknownAccount: t('modelCatalog.unknownAccount'),
    noGroups: t('modelCatalog.noGroups'),
    noPrice: t('modelCatalog.noPrice'),
    priceTokenRange: (values) => t('modelCatalog.priceTokenRange', values),
    priceRequestRange: (price) => t('modelCatalog.priceRequestRange', { price }),
    usageUnknown: t('modelCatalog.usageUnknown'),
    views: {
      merged: t('modelCatalog.views.merged'),
      by_group: t('modelCatalog.views.by_group'),
      by_channel: t('modelCatalog.views.by_channel'),
      by_account: t('modelCatalog.views.by_account'),
    },
    usageMode: (mode) => t(`modelCatalog.usageModes.${mode}`, mode),
    setupMonitor: t('modelCatalog.setupMonitor'),
    monitorStatus: (status) => t(`modelCatalog.monitorStatus.${status}`, status),
    empty: t('modelCatalog.empty'),
  },
  onRefresh: () => { void loadCatalog() },
  onQuickSetup: openQuickSetup,
}))

const monitorPrefill = ref<{
  name?: string
  provider?: Provider
  api_mode?: APIMode
  endpoint?: string
  api_key?: string
  primary_model?: string
  interval_seconds?: number
}>({})

function openQuickSetup(model: ModelCatalogModel) {
  setupModel.value = model
  selectedAccountId.value = null
  showAccountPicker.value = true
}

function openMonitorForm() {
  if (!setupModel.value || !selectedAccountId.value) return

  const source = setupModel.value.sources?.find(s => s.account_id === selectedAccountId.value)
  if (!source) return

  // 根据平台推断 provider
  let provider: Provider = 'openai'
  const platform = (source.platform || '').toLowerCase()
  if (platform.includes('anthropic') || platform.includes('claude')) provider = 'anthropic'
  else if (platform.includes('gemini') || platform.includes('google')) provider = 'gemini'

  monitorPrefill.value = {
    name: `Monitor: ${setupModel.value.id}`,
    provider,
    primary_model: setupModel.value.id,
    interval_seconds: intervalSeconds.value,
  }

  showAccountPicker.value = false
  showMonitorForm.value = true
}

function onMonitorSaved() {
  showMonitorForm.value = false
  loadCatalog()
}

async function loadCatalog() {
  loading.value = true
  try {
    const data = await getAdminModelCatalog()
    models.value = data.models || []
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadCatalog)
</script>
