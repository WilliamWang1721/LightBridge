<template>
  <RekaSelectItem
    v-bind="delegatedProps"
    :class="cn(
      'relative flex w-full cursor-default select-none items-center rounded-[calc(var(--ui-radius)-2px)] py-2 pl-8 pr-2 text-sm outline-none',
      'focus:bg-[hsl(var(--accent))] focus:text-[hsl(var(--accent-foreground))]',
      'data-[highlighted]:bg-[hsl(var(--accent))] data-[highlighted]:text-[hsl(var(--accent-foreground))]',
      'data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
      props.class,
    )"
  >
    <span class="absolute left-2 flex h-4 w-4 items-center justify-center">
      <SelectItemIndicator>
        <AppIcon name="check" size="sm" />
      </SelectItemIndicator>
    </span>
    <SelectItemText>
      <slot />
    </SelectItemText>
  </RekaSelectItem>
</template>

<script setup lang="ts">
import { computed, type HTMLAttributes } from 'vue'
import { AppIcon } from '@/components/ui/icon'
import {
  SelectItem as RekaSelectItem,
  SelectItemIndicator,
  SelectItemText,
  type SelectItemProps,
} from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<SelectItemProps & {
  class?: HTMLAttributes['class']
}>()
const delegatedProps = computed(() => {
  const { class: _class, ...delegated } = props
  return delegated
})
</script>
