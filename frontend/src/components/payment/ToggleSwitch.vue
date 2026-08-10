<template>
  <div class="flex flex-col items-center gap-0.5">
    <span class="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">{{ label }}</span>
    <Switch
      :model-value="checked"
      :aria-label="label"
      @update:model-value="emit('toggle')"
    />
  </div>
</template>

<script setup lang="ts">
import { Switch } from '@/components/ui/switch'

defineProps<{ label: string; checked: boolean }>()
const emit = defineEmits<{ toggle: [] }>()
</script>
