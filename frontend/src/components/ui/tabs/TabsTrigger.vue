<template>
  <RekaTabsTrigger
    :value="value"
    :disabled="disabled"
    :class="cn(
      'inline-flex min-h-8 items-center justify-center whitespace-nowrap rounded-[calc(var(--ui-radius)-0.125rem)] px-3 py-1.5',
      'text-sm font-medium transition-all',
      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] focus-visible:ring-offset-2',
      'disabled:pointer-events-none disabled:opacity-50',
      'data-[state=active]:bg-[hsl(var(--background))] data-[state=active]:text-[hsl(var(--foreground))]',
      'data-[state=active]:shadow-sm',
      props.class,
    )"
  >
    <slot />
  </RekaTabsTrigger>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { TabsTrigger as RekaTabsTrigger } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  value: string
  disabled?: boolean
  class?: HTMLAttributes['class']
}>(), {
  disabled: false,
})
</script>
