import * as React from 'react'
import { cn } from '../lib/utils'

const Textarea = React.forwardRef<HTMLTextAreaElement, React.TextareaHTMLAttributes<HTMLTextAreaElement>>(({ className, ...props }, ref) => {
  const cleaned = className?.replace(/\binput(?:-[a-z-]+)?\b/g, ' ')
  return React.createElement('textarea', {
    ...props,
    ref,
    className: cn(
      'flex min-h-24 w-full resize-y rounded-[var(--ui-radius)] border border-[hsl(var(--input))] bg-[hsl(var(--background))] px-3 py-2 text-sm text-[hsl(var(--foreground))] shadow-sm outline-none transition-colors placeholder:text-[hsl(var(--muted-foreground))] focus-visible:border-[hsl(var(--ring))] focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring)/.18)] disabled:cursor-not-allowed disabled:opacity-50',
      cleaned,
    ),
  })
})

Textarea.displayName = 'Textarea'

export { Textarea }
