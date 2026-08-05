<template>
  <RekaAccordionContent
    :class="cn(
      'overflow-hidden text-sm text-[hsl(var(--muted-foreground))]',
      'data-[state=closed]:animate-accordion-up data-[state=open]:animate-accordion-down',
      props.class,
    )"
  >
    <div class="pb-4 pt-0">
      <slot />
    </div>
  </RekaAccordionContent>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { AccordionContent as RekaAccordionContent } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
