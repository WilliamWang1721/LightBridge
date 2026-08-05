<template>
  <ProgressRoot
    :model-value="value"
    :max="max"
    :class="cn(
      'relative h-2 w-full overflow-hidden rounded-full bg-[hsl(var(--secondary))]',
      props.class,
    )"
  >
    <ProgressIndicator
      class="h-full w-full rounded-full bg-[hsl(var(--primary))] transition-transform duration-300"
      :style="{ transform: `translateX(-${100 - percentage}%)` }"
    />
  </ProgressRoot>
</template>

<script setup lang="ts">
import { computed, type HTMLAttributes } from 'vue'
import { ProgressIndicator, ProgressRoot } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  value?: number
  max?: number
  class?: HTMLAttributes['class']
}>(), {
  value: 0,
  max: 100,
})

const percentage = computed(() => {
  if (props.max <= 0) return 0
  return Math.min(100, Math.max(0, (props.value / props.max) * 100))
})
</script>
