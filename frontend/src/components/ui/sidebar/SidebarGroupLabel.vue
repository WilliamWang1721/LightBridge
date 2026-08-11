<template>
  <div v-bind="$attrs" data-slot="sidebar-group-label" :class="cn('flex h-8 shrink-0 items-center px-2 text-xs font-medium text-[hsl(var(--muted-foreground))]', props.class)">
    <slot />
  </div>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
