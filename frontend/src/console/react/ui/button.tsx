import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '../lib/utils'

const buttonVariants = cva(
  'inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-[var(--ui-radius)] text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:shrink-0',
  {
    variants: {
      variant: {
        default: 'bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))] hover:bg-[hsl(var(--primary)/.88)]',
        destructive: 'bg-[hsl(var(--destructive))] text-[hsl(var(--destructive-foreground))] hover:bg-[hsl(var(--destructive)/.88)]',
        outline: 'border border-[hsl(var(--border))] bg-[hsl(var(--background))] text-[hsl(var(--foreground))] hover:bg-[hsl(var(--muted))]',
        secondary: 'bg-[hsl(var(--secondary))] text-[hsl(var(--secondary-foreground))] hover:bg-[hsl(var(--secondary)/.78)]',
        ghost: 'text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]',
        link: 'text-[hsl(var(--foreground))] underline-offset-4 hover:underline',
      },
      size: {
        default: 'h-10 px-4 py-2',
        sm: 'h-9 rounded-lg px-3',
        lg: 'h-11 rounded-xl px-6',
        icon: 'h-10 w-10',
      },
    },
    defaultVariants: { variant: 'default', size: 'default' },
  },
)

type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & VariantProps<typeof buttonVariants>

const legacyVariant = (className?: string): VariantProps<typeof buttonVariants>['variant'] => {
  if (className?.includes('btn-danger')) return 'destructive'
  if (className?.includes('btn-secondary')) return 'outline'
  if (className?.includes('btn-ghost')) return 'ghost'
  if (className?.includes('btn-primary')) return 'default'
  return undefined
}

const legacySize = (className?: string): VariantProps<typeof buttonVariants>['size'] => {
  if (className?.includes('btn-icon')) return 'icon'
  if (className?.includes('btn-sm')) return 'sm'
  if (className?.includes('btn-lg')) return 'lg'
  return undefined
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(({ className, variant, size, type = 'button', ...props }, ref) => {
  const cleaned = className?.replace(/\bbtn(?:-[a-z-]+)?\b/g, ' ')
  return React.createElement('button', {
    ...props,
    ref,
    type,
    className: cn(buttonVariants({ variant: variant || legacyVariant(className), size: size || legacySize(className) }), cleaned),
  })
})

Button.displayName = 'Button'

export { Button, buttonVariants }
export type { ButtonProps }
