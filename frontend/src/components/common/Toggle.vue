<template>
  <Switch :model-value="modelValue" @update:model-value="emit('update:modelValue', $event)" />
</template>

<script setup lang="ts">
import { Switch } from '@/components/ui/switch'

defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()
</script>
