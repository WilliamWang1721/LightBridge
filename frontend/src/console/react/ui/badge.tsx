import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../lib/utils'

const badgeVariants = cva('inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors', {
  variants: {
    variant: {
      default: 'border-transparent bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))]',
      secondary: 'border-transparent bg-[hsl(var(--secondary))] text-[hsl(var(--secondary-foreground))]',
      outline: 'border-[hsl(var(--border))] bg-[hsl(var(--background))] text-[hsl(var(--foreground))]',
      destructive: 'border-transparent bg-[hsl(var(--destructive)/.1)] text-[hsl(var(--destructive))]',
    },
  },
  defaultVariants: { variant: 'default' },
})

type BadgeProps = React.HTMLAttributes<HTMLDivElement> & VariantProps<typeof badgeVariants>

function Badge({ className, variant, ...props }: BadgeProps) {
  return React.createElement('span', { ...props, className: cn(badgeVariants({ variant }), className) })
}

export { Badge, badgeVariants }
export type { BadgeProps }
