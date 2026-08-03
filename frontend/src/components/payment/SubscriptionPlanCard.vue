<template>
  <div
    class="group relative flex flex-col overflow-hidden rounded-2xl border border-gray-200 bg-white transition-all hover:-translate-y-0.5 hover:shadow-xl dark:border-dark-600 dark:bg-dark-800"
    :style="cardStyle"
  >
    <!-- Colored top accent bar -->
    <div class="h-1.5 bg-primary-500" :style="accentStyle" />

    <div class="flex flex-1 flex-col p-4">
      <!-- Header: name + badge + price -->
      <div class="mb-3 flex items-start justify-between gap-2">
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <h3 class="truncate text-base font-bold text-gray-900 dark:text-white">{{ plan.name }}</h3>
            <GroupBadge
              v-if="plan.group_name"
              :name="plan.group_name"
              :icon="plan.group_icon"
              :color="plan.group_color"
              :upstream-platforms="plan.upstream_platforms"
              :upstream-protocols="plan.upstream_protocols"
              :show-rate="false"
              :title="plan.available_ingress_protocols?.join(', ') || ''"
            />
          </div>
          <GroupUpstreamBadges
            v-if="plan.upstream_platforms?.length || plan.upstream_protocols?.length"
            class="mt-1.5"
            :upstream-platforms="plan.upstream_platforms"
            :upstream-protocols="plan.upstream_protocols"
          />
          <p v-if="plan.description" class="mt-0.5 text-xs leading-relaxed text-gray-500 dark:text-dark-400 line-clamp-2">
            {{ plan.description }}
          </p>
        </div>
        <div class="shrink-0 text-right">
          <div class="flex items-baseline gap-1">
            <span class="text-xs text-gray-400 dark:text-dark-500">$</span>
            <span class="text-2xl font-extrabold tracking-tight text-primary-600 dark:text-primary-400" :style="accentTextStyle">{{ plan.price }}</span>
          </div>
          <span class="text-[11px] text-gray-400 dark:text-dark-500">/ {{ validitySuffix }}</span>
          <div v-if="plan.original_price" class="mt-0.5 flex items-center justify-end gap-1.5">
            <span class="text-xs text-gray-400 line-through dark:text-dark-500">${{ plan.original_price }}</span>
            <span class="rounded bg-gray-100 px-1 py-0.5 text-[10px] font-semibold text-gray-600 dark:bg-dark-600 dark:text-gray-300">{{ discountText }}</span>
          </div>
        </div>
      </div>

      <!-- Group quota info (compact) -->
      <div class="mb-3 grid grid-cols-2 gap-x-3 gap-y-1 rounded-lg bg-gray-50 px-3 py-2 text-xs dark:bg-dark-700/50">
        <div class="flex items-center justify-between">
          <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.rate') }}</span>
          <span class="font-medium text-gray-700 dark:text-gray-300">{{ rateDisplay }}</span>
        </div>
        <div v-if="plan.daily_limit_usd != null" class="flex items-center justify-between">
          <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.dailyLimit') }}</span>
          <span class="font-medium text-gray-700 dark:text-gray-300">${{ plan.daily_limit_usd }}</span>
        </div>
        <div v-if="plan.weekly_limit_usd != null" class="flex items-center justify-between">
          <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.weeklyLimit') }}</span>
          <span class="font-medium text-gray-700 dark:text-gray-300">${{ plan.weekly_limit_usd }}</span>
        </div>
        <div v-if="plan.monthly_limit_usd != null" class="flex items-center justify-between">
          <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.monthlyLimit') }}</span>
          <span class="font-medium text-gray-700 dark:text-gray-300">${{ plan.monthly_limit_usd }}</span>
        </div>
        <div v-if="plan.daily_limit_usd == null && plan.weekly_limit_usd == null && plan.monthly_limit_usd == null" class="flex items-center justify-between">
          <span class="text-gray-400 dark:text-dark-500">{{ t('payment.planCard.quota') }}</span>
          <span class="font-medium text-gray-700 dark:text-gray-300">{{ t('payment.planCard.unlimited') }}</span>
        </div>
      </div>

      <!-- Features list (compact) -->
      <div v-if="plan.features.length > 0" class="mb-3 space-y-1">
        <div v-for="feature in plan.features" :key="feature" class="flex items-start gap-1.5">
          <svg class="mt-0.5 h-3.5 w-3.5 flex-shrink-0 text-primary-500" :style="accentTextStyle" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
          </svg>
          <span class="text-xs text-gray-600 dark:text-gray-300">{{ feature }}</span>
        </div>
      </div>

      <div class="flex-1" />

      <!-- Subscribe Button -->
      <button
        type="button"
        class="w-full rounded-xl bg-primary-600 py-2.5 text-sm font-semibold text-white transition-all hover:bg-primary-700 active:scale-[0.98]"
        :style="buttonStyle"
        @click="emit('select', plan)"
      >
        {{ isRenewal ? t('payment.renewNow') : t('payment.subscribeNow') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionPlan } from '@/types/payment'
import type { UserSubscription } from '@/types'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupUpstreamBadges from '@/components/common/GroupUpstreamBadges.vue'
import { normalizeGroupColor } from '@/utils/groupUpstreams'

const props = defineProps<{ plan: SubscriptionPlan; activeSubscriptions?: UserSubscription[] }>()
const emit = defineEmits<{ select: [plan: SubscriptionPlan] }>()
const { t } = useI18n()

const isRenewal = computed(() =>
  props.activeSubscriptions?.some(s => s.group_id === props.plan.group_id && s.status === 'active') ?? false
)

const groupColor = computed(() => normalizeGroupColor(props.plan.group_color))
const cardStyle = computed<CSSProperties | undefined>(() =>
  groupColor.value ? { borderColor: `${groupColor.value}55` } : undefined,
)
const accentStyle = computed<CSSProperties | undefined>(() =>
  groupColor.value ? { backgroundColor: groupColor.value } : undefined,
)
const accentTextStyle = computed<CSSProperties | undefined>(() =>
  groupColor.value ? { color: groupColor.value } : undefined,
)
const buttonStyle = computed<CSSProperties | undefined>(() =>
  groupColor.value ? { backgroundColor: groupColor.value } : undefined,
)

const discountText = computed(() => {
  if (!props.plan.original_price || props.plan.original_price <= 0) return ''
  const pct = Math.round((1 - props.plan.price / props.plan.original_price) * 100)
  return pct > 0 ? `-${pct}%` : ''
})

const rateDisplay = computed(() => {
  const rate = props.plan.rate_multiplier ?? 1
  return `×${Number(rate.toPrecision(10))}`
})

const validitySuffix = computed(() => {
  const u = props.plan.validity_unit || 'day'
  if (u === 'month') return t('payment.perMonth')
  if (u === 'year') return t('payment.perYear')
  return `${props.plan.validity_days}${t('payment.days')}`
})
</script>
