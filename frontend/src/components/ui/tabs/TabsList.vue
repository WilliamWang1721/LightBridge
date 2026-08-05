<template>
  <TabsList
    :loop="loop"
    :class="cn(
      'inline-flex min-h-10 items-center justify-center gap-1 rounded-[var(--ui-radius)]',
      'border border-[hsl(var(--border))] bg-[hsl(var(--muted))] p-1 text-[hsl(var(--muted-foreground))]',
      props.class,
    )"
  >
    <slot />
  </TabsList>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { TabsList } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  loop?: boolean
  class?: HTMLAttributes['class']
}>(), {
  loop: true,
})
</script>
