<template>
  <div
    :class="cn(
      'toolbar flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between',
      props.class,
    )"
    role="toolbar"
    :aria-label="label"
  >
    <div class="flex min-w-0 flex-1 flex-wrap items-center gap-2">
      <slot />
    </div>
    <div v-if="$slots.actions" class="flex shrink-0 flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  label?: string
  class?: HTMLAttributes['class']
}>(), {
  label: 'Page actions',
})
</script>
