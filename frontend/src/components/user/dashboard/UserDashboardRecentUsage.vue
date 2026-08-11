<template>
  <Card>
    <div class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('dashboard.recentUsage') }}</h2>
      <span class="badge badge-gray">{{ t('dashboard.last7Days') }}</span>
    </div>
    <div class="p-6">
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner size="lg" />
      </div>
      <div v-else-if="normalizedData.length === 0" class="py-8">
        <EmptyState :title="t('dashboard.noUsageRecords')" :description="t('dashboard.startUsingApi')" />
      </div>
      <div v-else class="space-y-3">
        <div v-for="(log, index) in normalizedData" :key="log.id ?? index" class="flex min-w-0 flex-col items-stretch gap-3 rounded-xl bg-gray-50 p-4 transition-colors hover:bg-gray-100 sm:flex-row sm:items-center sm:justify-between dark:bg-dark-800/50 dark:hover:bg-dark-800">
          <div class="flex min-w-0 items-center gap-4">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]">
              <Icon name="beaker" size="md" />
            </div>
            <div class="min-w-0">
              <p class="truncate text-sm font-medium text-gray-900 dark:text-white" :title="log.model">{{ log.model }}</p>
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ formatDateTime(log.created_at) }}</p>
            </div>
          </div>
          <div class="min-w-0 text-left sm:shrink-0 sm:text-right">
            <p class="break-words text-sm font-semibold">
              <span class="text-[hsl(var(--foreground))]" :title="t('dashboard.actual')">${{ formatCost(log.actual_cost) }}</span>
              <span class="font-normal text-gray-400 dark:text-gray-500" :title="t('dashboard.standard')"> / ${{ formatCost(log.total_cost) }}</span>
            </p>
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ log.total_tokens.toLocaleString() }} tokens</p>
          </div>
        </div>

        <router-link to="/usage" class="flex items-center justify-center gap-2 py-3 text-sm font-medium text-[hsl(var(--foreground))] transition-colors hover:text-[hsl(var(--muted-foreground))]">
          {{ t('dashboard.viewAllUsage') }}
          <Icon name="arrowRight" size="sm" />
        </router-link>
      </div>
    </div>
  </Card>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import { Card } from '@/components/ui/card'
import { formatDateTime } from '@/utils/format'
import type { UsageLog } from '@/types'
import { computed } from 'vue'
import { readUsageNumber } from '@/utils/usageDisplay'

const props = defineProps<{
  data: UsageLog[] | null
  loading: boolean
}>()
const { t } = useI18n()
const formatCost = (c: number) => c.toFixed(4)

const normalizedData = computed(() => (props.data || []).map((log) => {
  const values = log as unknown as Record<string, unknown>
  const inputTokens = readUsageNumber(log, ['input_tokens', 'prompt_tokens'])
  const outputTokens = readUsageNumber(log, ['output_tokens', 'completion_tokens'])
  const cacheCreationTokens = readUsageNumber(log, ['cache_creation_tokens', 'cache_write_tokens'])
  const cacheReadTokens = readUsageNumber(log, ['cache_read_tokens', 'cached_tokens'])
  return {
    id: values.id as number | string | undefined,
    model: String(values.model ?? values.requested_model ?? values.upstream_model ?? '-'),
    created_at: String(values.created_at ?? values.createdAt ?? ''),
    total_tokens: readUsageNumber(
      log,
      ['total_tokens', 'tokens'],
      inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens,
    ),
    actual_cost: readUsageNumber(log, ['actual_cost', 'user_cost']),
    total_cost: readUsageNumber(log, ['total_cost', 'cost', 'standard_cost']),
  }
}))
</script>
