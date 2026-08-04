<template>
  <DropdownMenuPortal to="#lightbridge-ui-portal">
    <RekaDropdownMenuContent
      :side="side"
      :align="align"
      :side-offset="sideOffset"
      :collision-padding="collisionPadding"
      :class="cn(
        'z-[var(--ui-z-dropdown)] min-w-44 overflow-hidden rounded-[var(--ui-radius)]',
        'border border-[hsl(var(--border))] bg-[hsl(var(--popover))] p-1',
        'text-[hsl(var(--popover-foreground))] shadow-xl outline-none data-[state=open]:animate-scale-in',
        props.class,
      )"
    >
      <slot />
    </RekaDropdownMenuContent>
  </DropdownMenuPortal>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import {
  DropdownMenuContent as RekaDropdownMenuContent,
  DropdownMenuPortal,
} from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  side?: 'top' | 'right' | 'bottom' | 'left'
  align?: 'start' | 'center' | 'end'
  sideOffset?: number
  collisionPadding?: number
  class?: HTMLAttributes['class']
}>(), {
  side: 'bottom',
  align: 'end',
  sideOffset: 6,
  collisionPadding: 12,
})
</script>
