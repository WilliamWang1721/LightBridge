import * as React from 'react'
import { cn } from '../lib/utils'

const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(({ className, type = 'text', ...props }, ref) => {
  const isChoice = type === 'checkbox' || type === 'radio'
  const cleaned = className?.replace(/\binput(?:-[a-z-]+)?\b/g, ' ')
  return React.createElement('input', {
    ...props,
    ref,
    type,
    className: cn(
      isChoice
        ? 'h-4 w-4 shrink-0 rounded border-[hsl(var(--border))] accent-[hsl(var(--primary))]'
        : 'flex h-10 w-full rounded-[var(--ui-radius)] border border-[hsl(var(--input))] bg-[hsl(var(--background))] px-3 py-2 text-sm text-[hsl(var(--foreground))] shadow-sm outline-none transition-colors placeholder:text-[hsl(var(--muted-foreground))] focus-visible:border-[hsl(var(--ring))] focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring)/.18)] disabled:cursor-not-allowed disabled:opacity-50',
      cleaned,
    ),
  })
})

Input.displayName = 'Input'

export { Input }
