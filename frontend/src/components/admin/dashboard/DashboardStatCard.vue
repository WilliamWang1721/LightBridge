<template>
  <Card :class="props.class" :style="props.style">
    <div class="flex items-start gap-3">
      <div class="flex size-10 shrink-0 items-center justify-center rounded-xl bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]">
        <Icon :name="icon" size="md" :stroke-width="1.8" />
      </div>
      <div class="min-w-0">
        <p class="text-xs font-medium text-[hsl(var(--muted-foreground))]">{{ label }}</p>
        <p class="mt-1 text-xl font-semibold tracking-tight text-[hsl(var(--foreground))]">{{ value }}</p>
        <p v-if="hint" class="mt-1 text-xs text-[hsl(var(--muted-foreground))]">{{ hint }}</p>
        <slot name="meta" />
      </div>
    </div>
  </Card>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Card } from '@/components/ui/card'
import Icon from '@/components/icons/Icon.vue'

type DashboardIconName = 'key' | 'server' | 'chart' | 'userPlus' | 'cube' | 'database' | 'bolt' | 'clock' | 'dollar'

const props = defineProps<{
  label: string
  value: string | number
  hint?: string
  icon: DashboardIconName
  class?: HTMLAttributes['class']
  style?: HTMLAttributes['style']
}>()
</script>
