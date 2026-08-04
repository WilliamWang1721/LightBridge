<template>
  <RadioGroupItem
    :id="id"
    :value="value"
    :disabled="disabled"
    :class="cn(
      'aspect-square h-4 w-4 rounded-full border border-[hsl(var(--primary))] bg-[hsl(var(--background))] text-[hsl(var(--primary))] shadow-sm',
      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] focus-visible:ring-offset-2',
      'focus-visible:ring-offset-[hsl(var(--background))] disabled:cursor-not-allowed disabled:opacity-50',
      props.class,
    )"
  >
    <RadioGroupIndicator class="grid h-full w-full place-items-center">
      <span class="h-2 w-2 rounded-full bg-current" />
    </RadioGroupIndicator>
  </RadioGroupItem>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { RadioGroupIndicator, RadioGroupItem } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  id?: string
  value: string
  disabled?: boolean
  class?: HTMLAttributes['class']
}>(), {
  disabled: false,
})
</script>
