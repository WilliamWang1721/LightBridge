import * as React from 'react'
import { cn } from '../lib/utils'

const Label = React.forwardRef<HTMLLabelElement, React.LabelHTMLAttributes<HTMLLabelElement>>(({ className, ...props }, ref) => React.createElement('label', {
  ...props,
  ref,
  className: cn('text-sm font-medium leading-none text-[hsl(var(--foreground))] peer-disabled:cursor-not-allowed peer-disabled:opacity-70', className),
}))

Label.displayName = 'Label'

export { Label }
