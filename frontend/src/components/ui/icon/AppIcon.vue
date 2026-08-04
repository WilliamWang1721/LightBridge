<template>
  <component
    :is="icon"
    :class="cn(sizeClasses[size], props.class)"
    :aria-hidden="label ? undefined : 'true'"
    :aria-label="label"
  />
</template>

<script setup lang="ts">
import { computed, type HTMLAttributes } from 'vue'
import { useUIPlatform } from '@/composables/useUIPlatform'
import { resolveAppIcon, type AppIconName } from '@/ui-platform/icons'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  name: AppIconName
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  label?: string
  class?: HTMLAttributes['class']
}>(), {
  size: 'md',
})

const { profile } = useUIPlatform()
const icon = computed(() => resolveAppIcon(props.name, profile.value.iconLibrary))

const sizeClasses = {
  xs: 'h-3 w-3',
  sm: 'h-4 w-4',
  md: 'h-5 w-5',
  lg: 'h-6 w-6',
  xl: 'h-8 w-8',
}
</script>
