<template>
  <span
    :class="[
      'inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs font-medium transition-colors',
      badgeClass
    ]"
    :style="badgeStyle"
  >
    <Icon :name="normalizedIcon" size="xs" class="shrink-0" />
    <!-- Group name -->
    <span class="truncate">{{ name }}</span>
    <!-- Right side label -->
    <span v-if="showLabel" :class="labelClass">
      <template v-if="hasCustomRate">
        <!-- 原倍率删除线 + 专属倍率高亮 -->
        <span class="line-through opacity-50 mr-0.5">{{ rateMultiplier }}x</span>
        <span class="font-bold">{{ userRateMultiplier }}x</span>
      </template>
      <template v-else>
        {{ labelText }}
      </template>
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CSSProperties } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GroupIcon, SubscriptionType, GroupPlatform, GroupUpstreamProtocol } from '@/types'
import Icon from '@/components/icons/Icon.vue'
import { normalizeGroupColor, normalizeGroupIcon } from '@/utils/groupUpstreams'

interface Props {
  name: string
  icon?: GroupIcon | string | null
  color?: string | null
  upstreamPlatforms?: GroupPlatform[] | string[]
  upstreamProtocols?: GroupUpstreamProtocol[] | string[]
  subscriptionType?: SubscriptionType
  rateMultiplier?: number
  userRateMultiplier?: number | null // 用户专属倍率
  showRate?: boolean
  daysRemaining?: number | null // 剩余天数（订阅类型时使用）
  /**
   * 订阅分组默认在右侧 label 展示"订阅"或剩余天数；
   * 开启后订阅分组也改为显示倍率（保留订阅主题色 label，配合可用渠道这类
   * 只关心费率、不关心有效期的场景）。
   */
  alwaysShowRate?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  subscriptionType: 'standard',
  showRate: true,
  daysRemaining: null,
  userRateMultiplier: null,
  alwaysShowRate: false
})

const { t } = useI18n()

const isSubscription = computed(() => props.subscriptionType === 'subscription')
const normalizedIcon = computed(() => normalizeGroupIcon(props.icon))
const normalizedColor = computed(() => normalizeGroupColor(props.color))
const badgeStyle = computed<CSSProperties | undefined>(() => {
  if (!normalizedColor.value) return undefined
  return {
    color: normalizedColor.value,
    borderColor: `${normalizedColor.value}55`,
    backgroundColor: `${normalizedColor.value}14`,
  }
})

// 是否有专属倍率（且与默认倍率不同）
const hasCustomRate = computed(() => {
  return (
    props.userRateMultiplier !== null &&
    props.userRateMultiplier !== undefined &&
    props.rateMultiplier !== undefined &&
    props.userRateMultiplier !== props.rateMultiplier
  )
})

// 是否显示右侧标签
const showLabel = computed(() => {
  if (!props.showRate) return false
  // 订阅类型：显示天数或"订阅"
  if (isSubscription.value) return true
  // 标准类型：显示倍率（包括专属倍率）
  return props.rateMultiplier !== undefined || hasCustomRate.value
})

// Label text
const labelText = computed(() => {
  const rateLabel = props.rateMultiplier !== undefined ? `${props.rateMultiplier}x` : ''
  if (isSubscription.value && !props.alwaysShowRate) {
    // 如果有剩余天数，显示天数
    if (props.daysRemaining !== null && props.daysRemaining !== undefined) {
      if (props.daysRemaining <= 0) {
        return t('admin.users.expired')
      }
      return t('admin.users.daysRemaining', { days: props.daysRemaining })
    }
    // 否则显示"订阅"
    return t('groups.subscription')
  }
  return rateLabel
})

// Label style based on type and days remaining
const labelClass = computed(() => {
  const base = 'px-1.5 py-0.5 rounded text-[10px] font-semibold'

  if (!isSubscription.value) {
    // Standard: subtle background (不再为专属倍率使用不同的背景色)
    return `${base} bg-black/10 dark:bg-white/10`
  }

  // 订阅类型：根据剩余天数显示不同颜色
  if (props.daysRemaining !== null && props.daysRemaining !== undefined) {
    if (props.daysRemaining <= 0 || props.daysRemaining <= 3) {
      // 已过期或紧急（<=3天）：红色
      return `${base} bg-red-200/80 text-red-800 dark:bg-red-800/50 dark:text-red-300`
    }
    if (props.daysRemaining <= 7) {
      // 警告（<=7天）：橙色
      return `${base} bg-amber-200/80 text-amber-800 dark:bg-amber-800/50 dark:text-amber-300`
    }
  }

  return `${base} bg-black/10 dark:bg-white/10`
})

// Group appearance is independent from upstream brands. Empty values stay neutral.
const badgeClass = computed(() => {
  if (normalizedColor.value) return ''
  return isSubscription.value
    ? 'border-violet-200 bg-violet-100 text-violet-700 dark:border-violet-900/50 dark:bg-violet-900/30 dark:text-violet-400'
    : 'border-gray-200 bg-gray-100 text-gray-700 dark:border-dark-500 dark:bg-dark-600 dark:text-gray-300'
})
</script>
