<template>
  <DropdownMenuLabel
    :class="cn('px-2.5 py-2 text-xs font-semibold text-[hsl(var(--muted-foreground))]', inset ? 'pl-8' : '', props.class)"
  >
    <slot />
  </DropdownMenuLabel>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { DropdownMenuLabel } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  inset?: boolean
  class?: HTMLAttributes['class']
}>(), {
  inset: false,
})
</script>
