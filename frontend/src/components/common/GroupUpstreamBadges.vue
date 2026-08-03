<template>
  <div class="flex flex-wrap items-center gap-1.5">
    <span
      v-for="platform in normalizedPlatforms"
      :key="`platform-${platform}`"
      :class="[
        'inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium dark:bg-dark-600',
        groupUpstreamPlatformTextClass(platform),
      ]"
    >
      <PlatformIcon :platform="platform" size="xs" />
      {{ t(`admin.groups.upstreamPlatforms.${platform}`) }}
    </span>
    <span
      v-for="protocol in normalizedProtocols"
      :key="`protocol-${protocol}`"
      :class="groupUpstreamProtocolBadgeClass(protocol)"
    >
      {{ t(`admin.groups.upstreamProtocols.${protocol}`) }}
    </span>
    <span
      v-if="showEmpty && normalizedPlatforms.length === 0 && normalizedProtocols.length === 0"
      class="text-xs text-gray-400 dark:text-gray-500"
    >
      {{ t('admin.groups.upstreamProtocols.none') }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GroupPlatform, GroupUpstreamProtocol } from '@/types'
import PlatformIcon from './PlatformIcon.vue'
import {
  groupUpstreamPlatformTextClass,
  groupUpstreamProtocolBadgeClass,
  normalizeGroupUpstreamPlatforms,
  normalizeGroupUpstreamProtocols,
} from '@/utils/groupUpstreams'

const props = withDefaults(defineProps<{
  upstreamPlatforms?: GroupPlatform[] | string[]
  upstreamProtocols?: GroupUpstreamProtocol[] | string[]
  showEmpty?: boolean
}>(), {
  upstreamPlatforms: () => [],
  upstreamProtocols: () => [],
  showEmpty: false,
})

const { t } = useI18n()

const normalizedPlatforms = computed(() =>
  normalizeGroupUpstreamPlatforms(props.upstreamPlatforms),
)
const normalizedProtocols = computed(() =>
  normalizeGroupUpstreamProtocols(props.upstreamProtocols),
)
</script>
