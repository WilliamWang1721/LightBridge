<template>
  <textarea
    ref="textareaRef"
    v-bind="$attrs"
    :value="modelValue"
    :class="cn(
      'flex min-h-24 w-full rounded-[var(--ui-radius)] border border-[hsl(var(--input))]',
      'bg-[hsl(var(--background))] px-3 py-2 text-sm text-[hsl(var(--foreground))] shadow-sm',
      'placeholder:text-[hsl(var(--muted-foreground))]',
      'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] focus-visible:ring-offset-2',
      'focus-visible:ring-offset-[hsl(var(--background))]',
      'disabled:cursor-not-allowed disabled:opacity-50',
      'aria-[invalid=true]:border-[hsl(var(--destructive))] aria-[invalid=true]:ring-[hsl(var(--destructive))]',
      props.class,
    )"
    @input="onInput"
  />
</template>

<script setup lang="ts">
import { ref, type HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  modelValue?: string | number | null
  class?: HTMLAttributes['class']
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
}>()

const textareaRef = ref<HTMLTextAreaElement | null>(null)

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLTextAreaElement).value)
}

defineExpose({
  focus: () => textareaRef.value?.focus(),
  select: () => textareaRef.value?.select(),
  element: textareaRef,
})
</script>
