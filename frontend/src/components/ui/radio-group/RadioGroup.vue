<template>
  <RadioGroupRoot
    v-bind="$attrs"
    :model-value="modelValue"
    :disabled="disabled"
    :orientation="orientation"
    :class="cn(orientation === 'horizontal' ? 'flex flex-wrap gap-4' : 'grid gap-3', props.class)"
    @update:model-value="onUpdate"
  >
    <slot />
  </RadioGroupRoot>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { RadioGroupRoot } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  modelValue?: string
  disabled?: boolean
  orientation?: 'horizontal' | 'vertical'
  class?: HTMLAttributes['class']
}>(), {
  modelValue: '',
  disabled: false,
  orientation: 'vertical',
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
}>()

function onUpdate(value: string) {
  emit('update:modelValue', value)
}
</script>
