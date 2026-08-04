<template>
  <header :class="cn('page-header', props.class)" data-page-header>
    <div class="min-w-0">
      <div v-if="$slots.eyebrow" class="mb-2 text-xs font-semibold uppercase tracking-[0.12em] text-[hsl(var(--muted-foreground))]">
        <slot name="eyebrow" />
      </div>
      <div class="flex min-w-0 items-center gap-3">
        <slot name="icon" />
        <h1 class="page-title truncate">{{ title }}</h1>
      </div>
      <p v-if="description" class="page-description">{{ description }}</p>
      <slot name="description" />
    </div>

    <div v-if="$slots.actions" class="page-actions shrink-0">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  title: string
  description?: string
  class?: HTMLAttributes['class']
}>()
</script>
