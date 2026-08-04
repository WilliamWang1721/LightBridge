<template>
  <div v-if="fields.length" class="grid gap-4 md:grid-cols-2">
    <div v-for="field in fields" :key="field.key" class="space-y-2">
      <label :for="fieldId(field.key)" class="input-label">
        {{ field.label }}
      </label>

      <div v-if="field.type === 'color'" class="flex gap-2">
        <input
          :id="`${fieldId(field.key)}-picker`"
          type="color"
          class="h-10 w-12 cursor-pointer rounded-lg border border-gray-200 bg-white p-1 dark:border-dark-600 dark:bg-dark-800"
          :value="String(modelValue[field.key] || '#000000')"
          @input="update(field.key, ($event.target as HTMLInputElement).value)"
        >
        <input
          :id="fieldId(field.key)"
          type="text"
          class="input flex-1 font-mono"
          :placeholder="field.placeholder || '#2563eb'"
          :value="String(modelValue[field.key] || '')"
          @input="update(field.key, ($event.target as HTMLInputElement).value)"
        >
      </div>

      <input
        v-else-if="field.type === 'text'"
        :id="fieldId(field.key)"
        type="text"
        class="input"
        :placeholder="field.placeholder"
        :value="String(modelValue[field.key] || '')"
        @input="update(field.key, ($event.target as HTMLInputElement).value)"
      >

      <select
        v-else-if="field.type === 'select'"
        :id="fieldId(field.key)"
        class="input"
        :value="String(modelValue[field.key] || '')"
        @change="update(field.key, ($event.target as HTMLSelectElement).value)"
      >
        <option v-for="option in field.options || []" :key="option" :value="option">
          {{ option }}
        </option>
      </select>

      <input
        v-else-if="field.type === 'number'"
        :id="fieldId(field.key)"
        type="number"
        class="input"
        :min="field.min"
        :max="field.max"
        :step="field.step || 'any'"
        :value="Number(modelValue[field.key] ?? 0)"
        @input="update(field.key, Number(($event.target as HTMLInputElement).value))"
      >

      <label
        v-else-if="field.type === 'boolean'"
        :for="fieldId(field.key)"
        class="flex min-h-10 cursor-pointer items-center justify-between rounded-xl border border-gray-200 bg-white px-4 py-2 dark:border-dark-600 dark:bg-dark-800"
      >
        <span class="text-sm text-gray-600 dark:text-gray-300">
          {{ Boolean(modelValue[field.key]) ? onLabel : offLabel }}
        </span>
        <input
          :id="fieldId(field.key)"
          type="checkbox"
          class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          :checked="Boolean(modelValue[field.key])"
          @change="update(field.key, ($event.target as HTMLInputElement).checked)"
        >
      </label>

      <p v-if="field.description" class="input-hint">{{ field.description }}</p>
      <p v-if="errors[field.key]" class="input-error-text">{{ errors[field.key] }}</p>
    </div>
  </div>

  <div v-else class="rounded-xl border border-dashed border-gray-200 px-4 py-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
    {{ emptyLabel }}
  </div>
</template>

<script setup lang="ts">
import type { UIThemeConfigField } from '@/ui-platform/themePackages'

const props = withDefaults(defineProps<{
  themeId: string
  fields: UIThemeConfigField[]
  modelValue: Record<string, unknown>
  errors?: Record<string, string>
  emptyLabel?: string
  onLabel?: string
  offLabel?: string
}>(), {
  errors: () => ({}),
  emptyLabel: 'This UI package does not expose configurable parameters.',
  onLabel: 'Enabled',
  offLabel: 'Disabled',
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: Record<string, unknown>): void
}>()

function fieldId(key: string) {
  return `ui-theme-${props.themeId}-${key}`
}

function update(key: string, value: unknown) {
  emit('update:modelValue', {
    ...props.modelValue,
    [key]: value,
  })
}
</script>
