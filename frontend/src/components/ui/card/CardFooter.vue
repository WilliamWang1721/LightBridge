<template>
  <footer :class="cn('flex items-center gap-2 p-5 pt-0', props.class)">
    <slot />
  </footer>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
