<template>
  <TooltipProvider :delay-duration="delayDuration" :skip-delay-duration="skipDelayDuration">
    <TooltipRoot>
      <TooltipTrigger as-child>
        <slot />
      </TooltipTrigger>
      <TooltipPortal to="#lightbridge-ui-portal">
        <TooltipContent
          :side="side"
          :side-offset="sideOffset"
          :class="cn(
            'z-[var(--ui-z-dropdown)] max-w-xs rounded-[calc(var(--ui-radius)-0.125rem)]',
            'bg-[hsl(var(--foreground))] px-2.5 py-1.5 text-xs text-[hsl(var(--background))] shadow-lg',
            'data-[state=delayed-open]:animate-fade-in',
            contentClass,
          )"
        >
          <slot name="content">{{ text }}</slot>
          <TooltipArrow class="fill-[hsl(var(--foreground))]" />
        </TooltipContent>
      </TooltipPortal>
    </TooltipRoot>
  </TooltipProvider>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import {
  TooltipArrow,
  TooltipContent,
  TooltipPortal,
  TooltipProvider,
  TooltipRoot,
  TooltipTrigger,
} from 'reka-ui'
import { cn } from '@/lib/utils'

withDefaults(defineProps<{
  text?: string
  side?: 'top' | 'right' | 'bottom' | 'left'
  sideOffset?: number
  delayDuration?: number
  skipDelayDuration?: number
  contentClass?: HTMLAttributes['class']
}>(), {
  text: '',
  side: 'top',
  sideOffset: 6,
  delayDuration: 350,
  skipDelayDuration: 300,
})
</script>
