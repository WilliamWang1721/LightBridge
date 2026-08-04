<template>
  <main
    :class="cn(
      'app-main w-full space-y-[var(--ui-page-gap)]',
      bleed ? 'max-w-none px-0' : '',
      props.class,
    )"
    :data-page-shell="surface"
  >
    <slot />
  </main>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  surface?: 'admin' | 'user' | 'auth' | 'public' | 'setup' | 'payment'
  bleed?: boolean
  class?: HTMLAttributes['class']
}>(), {
  surface: 'user',
  bleed: false,
})
</script>
