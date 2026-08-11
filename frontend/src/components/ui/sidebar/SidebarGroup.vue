<template>
  <section v-bind="$attrs" data-slot="sidebar-group" :class="cn('relative flex w-full min-w-0 flex-col p-2', props.class)">
    <slot />
  </section>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
