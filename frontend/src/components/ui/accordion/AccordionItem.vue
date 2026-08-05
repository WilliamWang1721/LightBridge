<template>
  <RekaAccordionItem
    :value="value"
    :disabled="disabled"
    :class="cn('border-b border-[hsl(var(--border))]', props.class)"
  >
    <slot />
  </RekaAccordionItem>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { AccordionItem as RekaAccordionItem } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  value: string
  disabled?: boolean
  class?: HTMLAttributes['class']
}>(), {
  disabled: false,
})
</script>
