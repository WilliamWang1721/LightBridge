<template>
  <li v-bind="$attrs" data-slot="sidebar-menu-item" :class="cn('group/menu-item relative min-w-0', props.class)">
    <slot />
  </li>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
