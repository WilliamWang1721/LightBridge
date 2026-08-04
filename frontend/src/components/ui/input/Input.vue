<template>
  <input
    v-bind="$attrs"
    :value="modelValue"
    :class="cn(
      'flex h-[var(--ui-control-height)] w-full rounded-[var(--ui-radius)]',
      'border border-[hsl(var(--input))] bg-[hsl(var(--background))] px-3 py-2 text-sm',
      'text-[hsl(var(--foreground))] shadow-sm transition-[border-color,box-shadow,background-color]',
      'placeholder:text-[hsl(var(--muted-foreground))]',
      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] focus-visible:ring-offset-2',
      'focus-visible:ring-offset-[hsl(var(--background))]',
      'disabled:cursor-not-allowed disabled:opacity-50',
      'aria-[invalid=true]:border-[hsl(var(--destructive))] aria-[invalid=true]:ring-[hsl(var(--destructive))]',
      props.class,
    )"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  >
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  modelValue?: string | number | null
  class?: HTMLAttributes['class']
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
}>()
</script>
