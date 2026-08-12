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
            :models="models"
            :loading="loading"
            @refresh="loadCatalog"
          />
        </div>
      </template>
    </ReactPageHost>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import ReactPageHost from '@/console/ReactPageHost.vue'
import ModelCatalogPanel from '@/components/model-catalog/ModelCatalogPanel.vue'
import { getUserModelCatalog, type ModelCatalogModel } from '@/api/modelCatalog'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useI18n } from 'vue-i18n'
import type { ModelCatalogPageProps } from '@/console/react/ModelCatalogPage'

const { t } = useI18n()
const appStore = useAppStore()

const models = ref<ModelCatalogModel[]>([])
const loading = ref(false)

const loadModelCatalogPage = () => import('@/console/react/ModelCatalogPage')

const pageProps = computed<ModelCatalogPageProps>(() => ({
  models: models.value,
  loading: loading.value,
  admin: false,
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
}))

async function loadCatalog() {
  loading.value = true
  try {
    const data = await getUserModelCatalog()
    models.value = data.models || []
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadCatalog)
</script>
