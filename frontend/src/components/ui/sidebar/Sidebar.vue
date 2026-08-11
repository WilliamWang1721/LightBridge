<template>
  <aside
    v-bind="$attrs"
    data-slot="sidebar"
    :class="cn(
      'flex min-h-0 flex-col bg-[hsl(var(--card))] text-[hsl(var(--foreground))]',
      props.class,
    )"
  >
    <slot />
  </aside>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
