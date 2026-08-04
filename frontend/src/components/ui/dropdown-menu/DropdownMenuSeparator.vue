<template>
  <DropdownMenuSeparator :class="cn('-mx-1 my-1 h-px bg-[hsl(var(--border))]', props.class)" />
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { DropdownMenuSeparator } from 'reka-ui'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>
