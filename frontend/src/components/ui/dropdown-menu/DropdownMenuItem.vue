<template>
  <RekaDropdownMenuItem
    :disabled="disabled"
    :class="cn(
      'relative flex min-h-9 cursor-default select-none items-center gap-2 rounded-[calc(var(--ui-radius)-0.125rem)] px-2.5 py-2',
      'text-sm outline-none transition-colors',
      'focus:bg-[hsl(var(--accent))] focus:text-[hsl(var(--accent-foreground))]',
      'data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
      destructive ? 'text-[hsl(var(--destructive))] focus:bg-[hsl(var(--destructive)/0.1)]' : '',
      inset ? 'pl-8' : '',
      props.class,
    )"
  >
    <slot />
  </RekaDropdownMenuItem>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { DropdownMenuItem as RekaDropdownMenuItem } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  disabled?: boolean
  destructive?: boolean
  inset?: boolean
  class?: HTMLAttributes['class']
}>(), {
  disabled: false,
  destructive: false,
  inset: false,
})
</script>
