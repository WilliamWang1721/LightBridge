<template>
  <section
    :class="cn(
      'rounded-[calc(var(--ui-radius)+0.25rem)] border border-[hsl(var(--border))]',
      'bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))]',
      'shadow-[0_1px_2px_hsl(var(--foreground)/0.04),0_8px_24px_hsl(var(--foreground)/0.035)]',
      props.class,
    )"
    :aria-labelledby="titleId"
  >
    <div
      v-if="title || description || $slots.header || $slots.actions"
      class="flex flex-col gap-3 border-b border-[hsl(var(--border))] px-5 py-4 sm:flex-row sm:items-start sm:justify-between"
    >
      <div class="min-w-0">
        <slot name="header">
          <h2 v-if="title" :id="titleId" class="text-base font-semibold text-[hsl(var(--foreground))]">
            {{ title }}
          </h2>
          <p v-if="description" class="mt-1 text-sm leading-6 text-[hsl(var(--muted-foreground))]">
            {{ description }}
          </p>
        </slot>
      </div>
      <div v-if="$slots.actions" class="flex shrink-0 flex-wrap items-center gap-2">
        <slot name="actions" />
      </div>
    </div>

    <div :class="cn(padded ? 'p-5' : '', contentClass)">
      <slot />
    </div>

    <div v-if="$slots.footer" class="border-t border-[hsl(var(--border))] px-5 py-4">
      <slot name="footer" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, useId, useSlots, type HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  title?: string
  description?: string
  padded?: boolean
  class?: HTMLAttributes['class']
  contentClass?: HTMLAttributes['class']
}>(), {
  padded: true,
})

const generatedId = useId()
const slots = useSlots()
const titleId = computed(() => props.title && !slots.header ? `page-section-${generatedId}` : undefined)
</script>
