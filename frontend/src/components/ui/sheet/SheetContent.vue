<template>
  <DialogPortal to="#lightbridge-ui-portal">
    <DialogOverlay
      class="fixed inset-0 z-[var(--ui-z-dialog)] bg-black/50 backdrop-blur-[2px] data-[state=open]:animate-fade-in"
    />
    <DialogContent
      v-bind="$attrs"
      :class="cn(
        'fixed z-[var(--ui-z-nested-dialog)] flex flex-col gap-4 border border-[hsl(var(--border))]',
        'bg-[hsl(var(--background))] p-5 text-[hsl(var(--foreground))] shadow-2xl outline-none',
        'data-[state=open]:animate-slide-in-right',
        sideClasses[side],
        props.class,
      )"
    >
      <slot />
      <DialogClose
        class="absolute right-4 top-4 rounded-[calc(var(--ui-radius)-0.125rem)] p-1.5 text-[hsl(var(--muted-foreground))] transition-colors hover:bg-[hsl(var(--accent))] hover:text-[hsl(var(--accent-foreground))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))]"
      >
        <X class="h-4 w-4" aria-hidden="true" />
        <span class="sr-only">Close</span>
      </DialogClose>
    </DialogContent>
  </DialogPortal>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { X } from '@lucide/vue'
import { DialogClose, DialogContent, DialogOverlay, DialogPortal } from 'reka-ui'
import { cn } from '@/lib/utils'

const sideClasses = {
  top: 'inset-x-0 top-0 max-h-[85dvh] rounded-b-[calc(var(--ui-radius)+0.25rem)] border-t-0',
  right: 'inset-y-0 right-0 h-full w-[min(92vw,32rem)] rounded-l-[calc(var(--ui-radius)+0.25rem)] border-r-0',
  bottom: 'inset-x-0 bottom-0 max-h-[85dvh] rounded-t-[calc(var(--ui-radius)+0.25rem)] border-b-0',
  left: 'inset-y-0 left-0 h-full w-[min(92vw,32rem)] rounded-r-[calc(var(--ui-radius)+0.25rem)] border-l-0',
} as const

const props = withDefaults(defineProps<{
  side?: keyof typeof sideClasses
  class?: HTMLAttributes['class']
}>(), {
  side: 'right',
})
</script>
