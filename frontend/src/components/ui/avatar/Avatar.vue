<template>
  <AvatarRoot
    :class="cn(
      'relative flex shrink-0 overflow-hidden rounded-full bg-[hsl(var(--muted))]',
      sizeClasses[size],
      props.class,
    )"
  >
    <AvatarImage
      v-if="src"
      :src="src"
      :alt="alt"
      class="aspect-square h-full w-full object-cover"
    />
    <AvatarFallback
      :delay-ms="delayMs"
      class="flex h-full w-full items-center justify-center rounded-full bg-[hsl(var(--muted))] text-xs font-medium text-[hsl(var(--muted-foreground))]"
    >
      <slot>{{ fallback }}</slot>
    </AvatarFallback>
  </AvatarRoot>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { AvatarFallback, AvatarImage, AvatarRoot } from 'reka-ui'
import { cn } from '@/lib/utils'

const sizeClasses = {
  sm: 'h-7 w-7',
  default: 'h-9 w-9',
  lg: 'h-12 w-12',
  xl: 'h-16 w-16',
} as const

const props = withDefaults(defineProps<{
  src?: string
  alt?: string
  fallback?: string
  delayMs?: number
  size?: keyof typeof sizeClasses
  class?: HTMLAttributes['class']
}>(), {
  src: undefined,
  alt: '',
  fallback: '?',
  delayMs: 0,
  size: 'default',
})
</script>
