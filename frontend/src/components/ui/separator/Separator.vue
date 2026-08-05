<template>
  <SeparatorRoot
    :orientation="orientation"
    :decorative="decorative"
    :class="cn(
      'shrink-0 bg-[hsl(var(--border))]',
      orientation === 'horizontal' ? 'h-px w-full' : 'h-full w-px',
      props.class,
    )"
  />
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Separator as SeparatorRoot } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  orientation?: 'horizontal' | 'vertical'
  decorative?: boolean
  class?: HTMLAttributes['class']
}>(), {
  orientation: 'horizontal',
  decorative: true,
})
</script>
