import * as React from 'react'
import { cn } from '../lib/utils'

const Select = React.forwardRef<HTMLSelectElement, React.SelectHTMLAttributes<HTMLSelectElement>>(({ className, ...props }, ref) => {
  const cleaned = className?.replace(/\binput(?:-[a-z-]+)?\b/g, ' ')
  return React.createElement('select', {
    ...props,
    ref,
    className: cn(
      'flex h-10 w-full appearance-none rounded-[var(--ui-radius)] border border-[hsl(var(--input))] bg-[hsl(var(--background))] px-3 py-2 pr-8 text-sm text-[hsl(var(--foreground))] shadow-sm outline-none transition-colors focus-visible:border-[hsl(var(--ring))] focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring)/.18)] disabled:cursor-not-allowed disabled:opacity-50',
      cleaned,
    ),
  })
})

Select.displayName = 'Select'

export { Select }
