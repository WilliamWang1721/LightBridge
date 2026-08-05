<template>
  <div
    aria-hidden="true"
    :class="cn('animate-pulse rounded-[var(--ui-radius)] bg-[hsl(var(--muted))]', props.class)"
  />
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
