<template>
  <section
    :class="cn(
      'overflow-hidden rounded-[calc(var(--ui-radius)+0.25rem)] border border-[hsl(var(--border))]',
      'bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))] shadow-[0_1px_2px_hsl(var(--foreground)/0.04)]',
      props.class,
    )"
    :aria-busy="loading || undefined"
  >
    <header
      v-if="$slots.toolbar || $slots.actions"
      class="flex flex-col gap-3 border-b border-[hsl(var(--border))] p-3 sm:flex-row sm:items-center sm:justify-between"
    >
      <div class="min-w-0 flex-1">
        <slot name="toolbar" />
      </div>
      <div v-if="$slots.actions" class="flex shrink-0 flex-wrap items-center gap-2">
        <slot name="actions" />
      </div>
    </header>

    <div v-if="loading" class="space-y-3 p-4" role="status" aria-label="Loading table">
      <Skeleton v-for="index in skeletonRows" :key="index" class="h-10 w-full" />
    </div>

    <PageState
      v-else-if="empty"
      :title="emptyTitle"
      :description="emptyDescription"
      kind="empty"
      class="m-4"
    >
      <template v-if="$slots.emptyIcon" #icon>
        <slot name="emptyIcon" />
      </template>
      <template v-if="$slots.emptyActions" #actions>
        <slot name="emptyActions" />
      </template>
    </PageState>

    <div v-else class="table-wrapper overflow-x-auto">
      <slot />
    </div>

    <footer
      v-if="$slots.footer"
      class="flex flex-col gap-3 border-t border-[hsl(var(--border))] p-3 sm:flex-row sm:items-center sm:justify-between"
    >
      <slot name="footer" />
    </footer>
  </section>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Skeleton } from '@/components/ui/skeleton'
import { PageState } from '@/components/ui/page'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  loading?: boolean
  empty?: boolean
  emptyTitle?: string
  emptyDescription?: string
  skeletonRows?: number
  class?: HTMLAttributes['class']
}>(), {
  loading: false,
  empty: false,
  emptyTitle: 'No data',
  emptyDescription: 'There are no records to display.',
  skeletonRows: 6,
})
</script>
