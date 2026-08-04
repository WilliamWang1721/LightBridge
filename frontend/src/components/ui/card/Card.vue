<template>
  <section
    :class="cn(
      'rounded-[calc(var(--ui-radius)+0.25rem)] border border-[hsl(var(--border))]',
      'bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))]',
      'shadow-[0_1px_2px_hsl(var(--foreground)/0.04),0_8px_24px_hsl(var(--foreground)/0.035)]',
      props.class,
    )"
  >
    <slot />
  </section>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
