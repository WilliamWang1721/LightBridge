<template>
  <footer v-bind="$attrs" data-slot="sidebar-footer" :class="cn('mt-auto shrink-0', props.class)">
    <slot />
  </footer>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
