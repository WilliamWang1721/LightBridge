<template>
  <RekaTabsContent
    :value="value"
    :class="cn(
      'mt-4 outline-none',
      'focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] focus-visible:ring-offset-2',
      props.class,
    )"
  >
    <slot />
  </RekaTabsContent>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { TabsContent as RekaTabsContent } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<{
  value: string
  class?: HTMLAttributes['class']
}>()
</script>
