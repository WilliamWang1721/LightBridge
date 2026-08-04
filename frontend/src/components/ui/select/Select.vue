<template>
  <SelectRoot v-bind="forwarded">
    <SelectTrigger
      :class="cn(
        'flex h-[var(--ui-control-height)] w-full items-center justify-between gap-2',
        'rounded-[var(--ui-radius)] border border-[hsl(var(--input))] bg-[hsl(var(--background))] px-3 py-2',
        'text-sm text-[hsl(var(--foreground))] shadow-sm',
        'focus:outline-none focus:ring-2 focus:ring-[hsl(var(--ring))] focus:ring-offset-2',
        'focus:ring-offset-[hsl(var(--background))] disabled:cursor-not-allowed disabled:opacity-50',
        'data-[placeholder]:text-[hsl(var(--muted-foreground))]',
        triggerClass,
      )"
    >
      <SelectValue :placeholder="placeholder" />
      <SelectIcon as-child>
        <ChevronDown class="h-4 w-4 shrink-0 opacity-60" aria-hidden="true" />
      </SelectIcon>
    </SelectTrigger>

    <SelectPortal to="#lightbridge-ui-portal">
      <SelectContent
        position="popper"
        :side-offset="4"
        class="z-[var(--ui-z-dropdown)] max-h-[var(--reka-select-content-available-height)] min-w-[var(--reka-select-trigger-width)] overflow-hidden rounded-[var(--ui-radius)] border border-[hsl(var(--border))] bg-[hsl(var(--popover))] text-[hsl(var(--popover-foreground))] shadow-xl data-[state=open]:animate-scale-in"
      >
        <SelectScrollUpButton class="flex h-7 cursor-default items-center justify-center">
          <ChevronUp class="h-4 w-4" aria-hidden="true" />
        </SelectScrollUpButton>
        <SelectViewport class="p-1">
          <slot />
        </SelectViewport>
        <SelectScrollDownButton class="flex h-7 cursor-default items-center justify-center">
          <ChevronDown class="h-4 w-4" aria-hidden="true" />
        </SelectScrollDownButton>
      </SelectContent>
    </SelectPortal>
  </SelectRoot>
</template>

<script setup lang="ts">
import { computed, type HTMLAttributes } from 'vue'
import { ChevronDown, ChevronUp } from '@lucide/vue'
import {
  SelectContent,
  SelectIcon,
  SelectPortal,
  SelectRoot,
  type SelectRootEmits,
  type SelectRootProps,
  SelectScrollDownButton,
  SelectScrollUpButton,
  SelectTrigger,
  SelectValue,
  SelectViewport,
  useForwardPropsEmits,
} from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<SelectRootProps & {
  placeholder?: string
  triggerClass?: HTMLAttributes['class']
}>()
const emits = defineEmits<SelectRootEmits>()
const delegatedProps = computed(() => {
  const { placeholder: _placeholder, triggerClass: _triggerClass, ...delegated } = props
  return delegated
})
const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>
