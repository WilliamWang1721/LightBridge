<template>
  <div class="w-full">
    <label v-if="label" :for="id" class="input-label mb-1.5 block">
      {{ label }}
      <span v-if="required" class="text-red-500">*</span>
    </label>
    <div class="relative">
      <LumaTextarea
        :id="id"
        ref="textAreaRef"
        :model-value="modelValue"
        :disabled="disabled"
        :required="required"
        :placeholder="placeholderText"
        :readonly="readonly"
        :rows="rows"
        :aria-invalid="error ? 'true' : undefined"
        :class="[
          'min-h-[80px] resize-y',
          disabled ? 'cursor-not-allowed opacity-60' : ''
        ]"
        @update:model-value="emit('update:modelValue', String($event ?? ''))"
        @change="$emit('change', ($event.target as HTMLTextAreaElement).value)"
        @blur="$emit('blur', $event)"
        @focus="$emit('focus', $event)"
      />
    </div>
    <!-- Hint / Error Text -->
    <p v-if="error" class="input-error-text mt-1.5">
      {{ error }}
    </p>
    <p v-else-if="hint" class="input-hint mt-1.5">
      {{ hint }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { Textarea as LumaTextarea } from '@/components/ui/textarea'

interface Props {
  modelValue: string | null | undefined
  label?: string
  placeholder?: string
  disabled?: boolean
  required?: boolean
  readonly?: boolean
  error?: string
  hint?: string
  id?: string
  rows?: number | string
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  required: false,
  readonly: false,
  rows: 3
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
  (e: 'blur', event: FocusEvent): void
  (e: 'focus', event: FocusEvent): void
}>()

const textAreaRef = ref<InstanceType<typeof LumaTextarea> | null>(null)
const placeholderText = computed(() => props.placeholder || '')

// Expose focus method
defineExpose({
  focus: () => textAreaRef.value?.focus(),
  select: () => textAreaRef.value?.select()
})
</script>
