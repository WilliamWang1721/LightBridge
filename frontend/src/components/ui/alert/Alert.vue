<template>
  <div role="alert" :class="cn(alertVariants({ variant }), props.class)">
    <slot />
  </div>
</template>

<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const alertVariants = cva(
  [
    'relative w-full rounded-[var(--ui-radius)] border p-4',
    '[&>svg~*]:pl-7 [&>svg]:absolute [&>svg]:left-4 [&>svg]:top-4 [&>svg]:h-4 [&>svg]:w-4',
  ],
  {
    variants: {
      variant: {
        default: 'border-[hsl(var(--border))] bg-[hsl(var(--background))] text-[hsl(var(--foreground))]',
        info: 'border-blue-500/25 bg-blue-500/10 text-blue-900 dark:text-blue-100',
        success: 'border-emerald-500/25 bg-emerald-500/10 text-emerald-900 dark:text-emerald-100',
        warning: 'border-amber-500/25 bg-amber-500/10 text-amber-950 dark:text-amber-100',
        destructive:
          'border-[hsl(var(--destructive)/0.3)] bg-[hsl(var(--destructive)/0.08)] text-[hsl(var(--destructive))]',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  },
)

type AlertVariants = VariantProps<typeof alertVariants>

const props = withDefaults(defineProps<{
  variant?: AlertVariants['variant']
  class?: HTMLAttributes['class']
}>(), {
  variant: 'default',
})
</script>
