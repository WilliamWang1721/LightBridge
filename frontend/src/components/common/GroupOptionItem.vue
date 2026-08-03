<template>
  <div class="flex min-w-0 flex-1 items-start justify-between gap-3">
    <!-- Left: name + description -->
    <div
      class="flex min-w-0 flex-1 flex-col items-start"
      :title="description || undefined"
    >
      <!-- Row 1: group name and upstream protocols -->
      <GroupBadge
        :name="name"
        :icon="icon"
        :color="color"
        :upstream-platforms="upstreamPlatforms"
        :upstream-protocols="upstreamProtocols"
        :subscription-type="subscriptionType"
        :show-rate="false"
        class="groupOptionItemBadge"
      />
      <GroupUpstreamBadges
        v-if="upstreamPlatforms?.length || upstreamProtocols?.length"
        class="mt-1"
        :upstream-platforms="upstreamPlatforms"
        :upstream-protocols="upstreamProtocols"
      />
      <!-- Row 2: description with top spacing -->
      <span
        v-if="description"
        class="mt-1.5 w-full text-left text-xs leading-relaxed text-gray-500 dark:text-gray-400 line-clamp-2"
      >
        {{ description }}
      </span>
    </div>

    <!-- Right: rate pill + checkmark (vertically centered to first row) -->
    <div class="flex shrink-0 items-center gap-2 pt-0.5">
      <!-- Rate pill -->
      <span
        v-if="rateMultiplier !== undefined"
        :class="['inline-flex items-center whitespace-nowrap rounded-full px-3 py-1 text-xs font-semibold', ratePillClass]"
        :style="ratePillStyle"
      >
        <template v-if="hasCustomRate">
          <span class="mr-1 line-through opacity-50">{{ rateMultiplier }}x</span>
          <span class="font-bold">{{ userRateMultiplier }}x</span>
        </template>
        <template v-else>
          {{ rateMultiplier }}x 倍率
        </template>
      </span>
      <!-- Checkmark -->
      <svg
        v-if="showCheckmark && selected"
        class="h-4 w-4 shrink-0 text-primary-600 dark:text-primary-400"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        stroke-width="2"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CSSProperties } from 'vue'
import GroupBadge from './GroupBadge.vue'
import type { GroupIcon, SubscriptionType, GroupPlatform, GroupUpstreamProtocol } from '@/types'
import GroupUpstreamBadges from './GroupUpstreamBadges.vue'
import { normalizeGroupColor } from '@/utils/groupUpstreams'

interface Props {
  name: string
  icon?: GroupIcon | string | null
  color?: string | null
  upstreamPlatforms?: GroupPlatform[]
  upstreamProtocols?: GroupUpstreamProtocol[]
  subscriptionType?: SubscriptionType
  rateMultiplier?: number
  userRateMultiplier?: number | null
  description?: string | null
  selected?: boolean
  showCheckmark?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  subscriptionType: 'standard',
  selected: false,
  showCheckmark: true,
  userRateMultiplier: null
})

// Whether user has a custom rate different from default
const hasCustomRate = computed(() => {
  return (
    props.userRateMultiplier !== null &&
    props.userRateMultiplier !== undefined &&
    props.rateMultiplier !== undefined &&
    props.userRateMultiplier !== props.rateMultiplier
  )
})

const normalizedColor = computed(() => normalizeGroupColor(props.color))

const ratePillClass = computed(() => {
  return normalizedColor.value
    ? ''
    : 'bg-gray-100 text-gray-700 dark:bg-dark-600 dark:text-gray-300'
})

const ratePillStyle = computed<CSSProperties | undefined>(() => {
  if (!normalizedColor.value) return undefined
  return {
    color: normalizedColor.value,
    backgroundColor: `${normalizedColor.value}14`,
  }
})
</script>

<style scoped>
/* Bold the group name inside GroupBadge when used in dropdown option */
.groupOptionItemBadge :deep(span.truncate) {
  font-weight: 600;
}
</style>
