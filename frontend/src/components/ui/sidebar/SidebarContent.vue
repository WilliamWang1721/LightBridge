<template>
  <nav v-bind="$attrs" data-slot="sidebar-content" :class="cn('min-h-0 flex-1 overflow-y-auto', props.class)">
    <slot />
  </nav>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
