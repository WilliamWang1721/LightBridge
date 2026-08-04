<template>
  <h3
    :class="cn(
      'text-base font-semibold leading-none tracking-[var(--ui-heading-tracking)] text-[hsl(var(--foreground))]',
      props.class,
    )"
  >
    <slot />
  </h3>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
