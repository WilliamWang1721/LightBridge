<template>
  <CheckboxRoot
    v-bind="$attrs"
    :model-value="modelValue"
    :disabled="disabled"
    :class="cn(
      'peer h-4 w-4 shrink-0 rounded-[calc(var(--ui-radius)-0.25rem)] border border-[hsl(var(--primary))]',
      'bg-[hsl(var(--background))] shadow-sm',
      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] focus-visible:ring-offset-2',
      'focus-visible:ring-offset-[hsl(var(--background))]',
      'disabled:cursor-not-allowed disabled:opacity-50',
      'data-[state=checked]:bg-[hsl(var(--primary))] data-[state=checked]:text-[hsl(var(--primary-foreground))]',
      'data-[state=indeterminate]:bg-[hsl(var(--primary))] data-[state=indeterminate]:text-[hsl(var(--primary-foreground))]',
      props.class,
    )"
    @update:model-value="onUpdate"
  >
    <CheckboxIndicator class="grid h-full w-full place-items-center text-current">
      <Minus v-if="modelValue === 'indeterminate'" class="h-3 w-3" aria-hidden="true" />
      <Check v-else class="h-3 w-3" aria-hidden="true" />
    </CheckboxIndicator>
  </CheckboxRoot>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Check, Minus } from '@lucide/vue'
import { CheckboxIndicator, CheckboxRoot } from 'reka-ui'
import { cn } from '@/lib/utils'

type CheckboxValue = boolean | 'indeterminate'

const props = withDefaults(defineProps<{
  modelValue?: CheckboxValue
  disabled?: boolean
  class?: HTMLAttributes['class']
}>(), {
  modelValue: false,
  disabled: false,
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: CheckboxValue): void
}>()

function onUpdate(value: CheckboxValue) {
  emit('update:modelValue', value)
}
</script>
