<template>
  <PopoverPortal to="#lightbridge-ui-portal">
    <RekaPopoverContent
      :side="side"
      :align="align"
      :side-offset="sideOffset"
      :collision-padding="collisionPadding"
      :class="cn(
        'z-[var(--ui-z-dropdown)] w-80 rounded-[var(--ui-radius)] border border-[hsl(var(--border))]',
        'bg-[hsl(var(--popover))] p-4 text-[hsl(var(--popover-foreground))] shadow-xl',
        'outline-none data-[state=open]:animate-scale-in',
        props.class,
      )"
    >
      <slot />
    </RekaPopoverContent>
  </PopoverPortal>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { PopoverContent as RekaPopoverContent, PopoverPortal } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  side?: 'top' | 'right' | 'bottom' | 'left'
  align?: 'start' | 'center' | 'end'
  sideOffset?: number
  collisionPadding?: number
  class?: HTMLAttributes['class']
}>(), {
  side: 'bottom',
  align: 'center',
  sideOffset: 6,
  collisionPadding: 12,
})
</script>
