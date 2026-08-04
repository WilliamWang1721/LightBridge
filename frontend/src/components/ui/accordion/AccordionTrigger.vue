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
      <AppIcon
        name="chevron-down"
        size="sm"
        class="shrink-0 text-[hsl(var(--muted-foreground))] transition-transform duration-200 group-data-[state=open]:rotate-180"
      />
    </RekaAccordionTrigger>
  </AccordionHeader>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { AppIcon } from '@/components/ui/icon'
import { AccordionHeader, AccordionTrigger as RekaAccordionTrigger } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
