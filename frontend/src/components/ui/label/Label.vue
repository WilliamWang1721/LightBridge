<template>
  <RekaLabel
    :for="forId"
    :class="cn(
      'text-sm font-medium leading-none text-[hsl(var(--foreground))]',
      'peer-disabled:cursor-not-allowed peer-disabled:opacity-70',
      props.class,
    )"
  >
    <slot />
  </RekaLabel>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Label as RekaLabel } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<{
  forId?: string
  class?: HTMLAttributes['class']
}>()
</script>
