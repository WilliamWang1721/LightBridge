<template>
  <div class="space-y-4">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex flex-1 flex-wrap items-center gap-3">
        <div class="relative w-full sm:w-80">
          <Icon
            name="search"
            size="md"
            class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
          />
          <input
            v-model="searchQuery"
            type="text"
            class="input pl-10"
            :placeholder="t('modelCatalog.searchPlaceholder')"
          />
        </div>

        <div class="inline-flex overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
          <button
            v-for="mode in visibleViewModes"
            :key="mode"
            type="button"
            :class="[
              'px-3 py-2 text-sm transition-colors',
              activeView === mode
                ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
                : 'text-gray-600 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-700'
            ]"
            @click="activeView = mode"
          >
            {{ t(`modelCatalog.views.${mode}`) }}
          </button>
        </div>
      </div>

      <button
        type="button"
        class="btn btn-secondary"
        :disabled="loading"
        :title="t('common.refresh', 'Refresh')"
        @click="$emit('refresh')"
      >
        <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
      </button>
    </div>

    <div v-if="loading" class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="i in 6"
        :key="i"
        class="h-44 animate-pulse rounded-lg border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800"
      />
    </div>

    <div
      v-else-if="sections.length === 0"
      class="rounded-lg border border-dashed border-gray-300 bg-white px-6 py-12 text-center dark:border-dark-600 dark:bg-dark-800"
    >
      <Icon name="database" size="xl" class="mx-auto mb-3 text-gray-300 dark:text-gray-600" />
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('modelCatalog.empty') }}</p>
    </div>

    <div v-else class="space-y-6">
      <section v-for="section in sections" :key="section.key" class="space-y-3">
        <div v-if="activeView !== 'merged'" class="flex items-center gap-2">
          <h2 class="text-sm font-semibold text-gray-800 dark:text-gray-100">{{ section.title }}</h2>
          <span class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-400">
            {{ t('modelCatalog.modelCount', { count: section.models.length }) }}
          </span>
        </div>

        <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          <article
            v-for="model in section.models"
            :key="`${section.key}:${model.id}`"
            class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800"
          >
            <div class="mb-3 flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="flex items-center gap-2">
                  <ModelIcon :model="model.id" size="20px" />
                  <h3 class="truncate text-sm font-semibold text-gray-900 dark:text-white">
                    {{ model.display_name || model.id }}
                  </h3>
                </div>
                <p class="mt-1 truncate font-mono text-xs text-gray-500 dark:text-gray-400">{{ model.id }}</p>
              </div>
              <span class="shrink-0 rounded-full bg-emerald-50 px-2 py-1 text-xs text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
                {{ t('modelCatalog.sourceCount', { count: model.source_count }) }}
              </span>
            </div>

            <div class="mb-3 flex flex-wrap gap-1.5">
              <span
                v-for="mode in (model.usage_modes || [])"
                :key="mode"
                class="rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
              >
                {{ formatUsageMode(mode) }}
              </span>
              <span
                v-if="!(model.usage_modes || []).length"
                class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-400"
              >
                {{ t('modelCatalog.usageUnknown') }}
              </span>
            </div>

            <div class="space-y-2 text-xs text-gray-500 dark:text-gray-400">
              <div class="flex items-start gap-2">
                <Icon name="dollar" size="sm" class="mt-0.5 shrink-0" />
                <span>{{ formatPriceRange(model.price_range) }}</span>
              </div>
              <div class="flex items-start gap-2">
                <Icon name="grid" size="sm" class="mt-0.5 shrink-0" />
                <span class="line-clamp-2">{{ formatGroups(model.groups) }}</span>
              </div>
            </div>

            <div v-if="admin && model.sources?.length" class="mt-4 border-t border-gray-100 pt-3 dark:border-dark-700">
              <div class="mb-2 text-xs font-medium text-gray-500 dark:text-gray-400">
                {{ t('modelCatalog.sourceDetails') }}
              </div>
              <div class="space-y-1.5">
                <div
                  v-for="source in model.sources"
                  :key="`${source.account_id}:${source.platform}:${source.source}`"
                  class="flex items-center justify-between gap-2 rounded bg-gray-50 px-2 py-1.5 text-xs dark:bg-dark-700"
                >
                  <span class="min-w-0 truncate text-gray-700 dark:text-gray-200">
                    {{ source.account_name || t('modelCatalog.unknownAccount') }}
                  </span>
                  <span class="shrink-0 text-gray-500 dark:text-gray-400">
                    {{ formatSourceChannels(source) }}
                  </span>
                </div>
              </div>
            </div>
          </article>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import type {
  ModelCatalogGroup,
  ModelCatalogModel,
  ModelCatalogPriceRange
} from '@/api/modelCatalog'

type CatalogViewMode = 'merged' | 'by_group' | 'by_channel' | 'by_account'

const props = withDefaults(defineProps<{
  models: ModelCatalogModel[]
  loading?: boolean
  admin?: boolean
}>(), {
  models: () => [],
  loading: false,
  admin: false
})

defineEmits<{
  refresh: []
}>()

const { t } = useI18n()

const searchQuery = ref('')
const activeView = ref<CatalogViewMode>('merged')

const visibleViewModes = computed<CatalogViewMode[]>(() =>
  props.admin ? ['merged', 'by_group', 'by_channel', 'by_account'] : ['merged', 'by_group']
)

const filteredModels = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return props.models
  return props.models.filter((model) => {
    const groupHit = (model.groups || []).some((group) => group.name.toLowerCase().includes(query))
    const sourceHit = props.admin && model.sources?.some((source) =>
      [source.account_name, source.platform, source.source].some((value) =>
        String(value || '').toLowerCase().includes(query)
      )
    )
    return (
      model.id.toLowerCase().includes(query) ||
      (model.display_name || '').toLowerCase().includes(query) ||
      model.platform.toLowerCase().includes(query) ||
      groupHit ||
      Boolean(sourceHit)
    )
  })
})

const sections = computed(() => {
  const models = filteredModels.value
  switch (activeView.value) {
    case 'by_group':
      return groupByModelGroups(models)
    case 'by_channel':
      return props.admin ? groupBySourcePlatform(models) : groupMerged(models)
    case 'by_account':
      return props.admin ? groupBySourceAccount(models) : groupMerged(models)
    case 'merged':
    default:
      return groupMerged(models)
  }
})

function groupMerged(models: ModelCatalogModel[]) {
  return [{ key: 'merged', title: t('modelCatalog.views.merged'), models }]
}

function groupByModelGroups(models: ModelCatalogModel[]) {
  const sections = new Map<string, { key: string; title: string; models: ModelCatalogModel[] }>()
  for (const model of models) {
    for (const group of (model.groups || [])) {
      const key = String(group.id)
      if (!sections.has(key)) {
        sections.set(key, { key, title: group.name, models: [] })
      }
      sections.get(key)?.models.push(model)
    }
  }
  return Array.from(sections.values()).sort((a, b) => a.title.localeCompare(b.title))
}

function groupBySourcePlatform(models: ModelCatalogModel[]) {
  const sections = new Map<string, { key: string; title: string; models: ModelCatalogModel[] }>()
  for (const model of models) {
    const channels = new Map<string, string>()
    for (const source of model.sources || []) {
      for (const channel of source.channels || []) {
        const key = String(channel.id || channel.name)
        if (key) channels.set(key, channel.name || key)
      }
    }
    if (channels.size === 0) {
      for (const source of model.sources || []) {
        if (source.platform) channels.set(source.platform, source.platform)
      }
    }
    for (const [key, title] of channels) {
      if (!sections.has(key)) {
        sections.set(key, { key, title, models: [] })
      }
      sections.get(key)?.models.push(model)
    }
  }
  return Array.from(sections.values()).sort((a, b) => a.title.localeCompare(b.title))
}

function groupBySourceAccount(models: ModelCatalogModel[]) {
  const sections = new Map<string, { key: string; title: string; models: ModelCatalogModel[] }>()
  for (const model of models) {
    for (const source of model.sources || []) {
      const key = String(source.account_id || source.account_name || source.platform)
      if (!sections.has(key)) {
        sections.set(key, { key, title: source.account_name || t('modelCatalog.unknownAccount'), models: [] })
      }
      sections.get(key)?.models.push(model)
    }
  }
  return Array.from(sections.values()).sort((a, b) => a.title.localeCompare(b.title))
}

function formatUsageMode(mode: string) {
  return t(`modelCatalog.usageModes.${mode}`, mode)
}

function formatGroups(groups: ModelCatalogGroup[]) {
  const list = groups || []
  if (!list.length) return t('modelCatalog.noGroups')
  return list.map((group) => group.name).join(', ')
}

function formatSourceChannels(source: NonNullable<ModelCatalogModel['sources']>[number]) {
  const channels = (source.channels || []).map((channel) => channel.name).filter(Boolean)
  if (channels.length > 0) return channels.join(', ')
  return source.platform
}

function formatPriceRange(range?: ModelCatalogPriceRange | null) {
  if (!range) return t('modelCatalog.noPrice')
  const input = formatMinMax(range.min_input_price, range.max_input_price)
  const output = formatMinMax(range.min_output_price, range.max_output_price)
  const request = formatMinMax(range.min_per_request_price, range.max_per_request_price)
  if (input || output) {
    return t('modelCatalog.priceTokenRange', {
      input: input || '-',
      output: output || '-'
    })
  }
  if (request) {
    return t('modelCatalog.priceRequestRange', { price: request })
  }
  return t('modelCatalog.noPrice')
}

function formatMinMax(min?: number | null, max?: number | null) {
  if (min == null && max == null) return ''
  if (min == null) return formatMoney(max)
  if (max == null || min === max) return formatMoney(min)
  return `${formatMoney(min)} - ${formatMoney(max)}`
}

function formatMoney(value?: number | null) {
  if (value == null) return '-'
  return `$${Number(value).toFixed(4)}`
}
</script>
