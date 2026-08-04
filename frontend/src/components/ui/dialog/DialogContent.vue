<template>
  <DialogPortal to="#lightbridge-ui-portal">
    <DialogOverlay
      class="fixed inset-0 z-[var(--ui-z-dialog)] bg-black/45 backdrop-blur-[2px] data-[state=open]:animate-fade-in"
    />
    <DialogContent
      v-bind="forwarded"
      :class="cn(
        'fixed left-1/2 top-1/2 z-[calc(var(--ui-z-dialog)+1)] grid w-[calc(100%-2rem)] max-w-lg',
        '-translate-x-1/2 -translate-y-1/2 gap-4 border border-[hsl(var(--border))]',
        'rounded-[calc(var(--ui-radius)+0.25rem)] bg-[hsl(var(--card))] p-6 text-[hsl(var(--card-foreground))]',
        'shadow-2xl outline-none data-[state=open]:animate-scale-in',
        props.class,
      )"
    >
      <slot />

      <DialogClose
        v-if="showClose"
        class="absolute right-4 top-4 rounded-[calc(var(--ui-radius)-2px)] p-1.5 text-[hsl(var(--muted-foreground))] transition-colors hover:bg-[hsl(var(--accent))] hover:text-[hsl(var(--accent-foreground))] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))]"
        :aria-label="closeLabel"
      >
        <AppIcon name="close" size="sm" />
        <span class="sr-only">{{ closeLabel }}</span>
      </DialogClose>
    </DialogContent>
  </DialogPortal>
</template>

<script setup lang="ts">
import { computed, type HTMLAttributes } from 'vue'
import { AppIcon } from '@/components/ui/icon'
import {
  DialogClose,
  DialogContent,
  DialogOverlay,
  DialogPortal,
  type DialogContentEmits,
  type DialogContentProps,
  useForwardPropsEmits,
} from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<DialogContentProps & {
  class?: HTMLAttributes['class']
  showClose?: boolean
  closeLabel?: string
}>(), {
  showClose: true,
  closeLabel: 'Close dialog',
})

const emits = defineEmits<DialogContentEmits>()
const delegatedProps = computed(() => {
  const { class: _class, showClose: _showClose, closeLabel: _closeLabel, ...delegated } = props
  return delegated
})
const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>
