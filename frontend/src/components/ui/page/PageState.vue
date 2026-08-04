<template>
  <div
    :class="cn(
      'empty-state flex min-h-56 flex-col items-center justify-center gap-3 px-5 py-10 text-center',
      props.class,
    )"
    :role="kind === 'error' ? 'alert' : 'status'"
  >
    <div
      v-if="$slots.icon"
      :class="cn(
        'grid h-12 w-12 place-items-center rounded-full',
        kind === 'error'
          ? 'bg-[hsl(var(--destructive)/0.1)] text-[hsl(var(--destructive))]'
          : 'bg-[hsl(var(--muted))] text-[hsl(var(--muted-foreground))]',
      )"
    >
      <slot name="icon" />
    </div>
    <div class="space-y-1">
      <h3 class="empty-state-title text-base font-semibold">{{ title }}</h3>
      <p v-if="description" class="empty-state-description text-sm leading-6">
        {{ description }}
      </p>
    </div>
    <div v-if="$slots.actions" class="mt-1 flex flex-wrap items-center justify-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  title: string
  description?: string
  kind?: 'empty' | 'loading' | 'error' | 'permission'
  class?: HTMLAttributes['class']
}>(), {
  kind: 'empty',
})
</script>
