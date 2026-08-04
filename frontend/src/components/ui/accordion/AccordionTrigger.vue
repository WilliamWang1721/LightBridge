<template>
  <AccordionHeader class="flex">
    <RekaAccordionTrigger
      :class="cn(
        'group flex flex-1 items-center justify-between gap-3 py-4 text-left text-sm font-medium',
        'text-[hsl(var(--foreground))] transition-colors hover:text-[hsl(var(--primary))]',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] focus-visible:ring-offset-2',
        props.class,
      )"
    >
      <slot />
      <ChevronDown
        class="h-4 w-4 shrink-0 text-[hsl(var(--muted-foreground))] transition-transform duration-200 group-data-[state=open]:rotate-180"
        aria-hidden="true"
      />
    </RekaAccordionTrigger>
  </AccordionHeader>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { ChevronDown } from '@lucide/vue'
import { AccordionHeader, AccordionTrigger as RekaAccordionTrigger } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
