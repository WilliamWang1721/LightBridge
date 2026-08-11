<template>
  <header v-bind="$attrs" data-slot="sidebar-header" :class="cn('shrink-0', props.class)">
    <slot />
  </header>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
