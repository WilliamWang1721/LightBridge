<template>
  <component
    :is="as"
    v-bind="$attrs"
    data-slot="sidebar-menu-button"
    :class="cn(
      'flex w-full min-w-0 items-center gap-2 overflow-hidden rounded-[var(--ui-radius)] px-3 py-2 text-left text-sm font-medium',
      'text-[hsl(var(--muted-foreground))] transition-colors hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]',
      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))]',
      props.class,
    )"
  >
    <slot />
  </component>
</template>

<script setup lang="ts">
import type { Component, HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{
  as?: string | Component
  class?: HTMLAttributes['class']
}>(), {
  as: 'button',
})

const { as } = props
</script>
