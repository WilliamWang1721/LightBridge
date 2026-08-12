import * as React from 'react'
import { cn } from '../lib/utils'

interface SwitchProps extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'onChange'> {
  checked?: boolean
  onCheckedChange?: (checked: boolean) => void
  onChange?: (checked: boolean) => void
}

const Switch = React.forwardRef<HTMLButtonElement, SwitchProps>(({ className, checked = false, onCheckedChange, onChange, role = 'switch', type = 'button', ...props }, ref) => React.createElement('button', {
  ...props,
  ref,
  type,
  role,
  'aria-checked': checked,
  onClick: () => {
    onCheckedChange?.(!checked)
    onChange?.(!checked)
  },
  className: cn(
    'relative inline-flex h-6 w-11 shrink-0 items-center rounded-full border border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[hsl(var(--ring))] disabled:cursor-not-allowed disabled:opacity-50',
    checked ? 'bg-[hsl(var(--primary))]' : 'bg-[hsl(var(--muted))]',
    className,
  ),
}, React.createElement('span', {
  className: cn('pointer-events-none block h-5 w-5 rounded-full bg-[hsl(var(--background))] shadow-sm transition-transform', checked ? 'translate-x-5' : 'translate-x-0.5'),
})) )

Switch.displayName = 'Switch'

export { Switch }
export type { SwitchProps }
