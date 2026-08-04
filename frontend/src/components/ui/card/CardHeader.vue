<template>
  <header :class="cn('flex flex-col gap-1.5 p-5', props.class)">
    <slot />
  </header>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
