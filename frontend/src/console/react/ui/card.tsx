import * as React from 'react'
import { cn } from '../lib/utils'

const Card = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, ...props }, ref) => React.createElement('div', {
  ...props,
  ref,
  className: cn('rounded-[calc(var(--ui-radius)+0.125rem)] border border-[hsl(var(--border))] bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))] shadow-sm', className),
}))
Card.displayName = 'Card'

const CardHeader = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, ...props }, ref) => React.createElement('div', { ...props, ref, className: cn('flex flex-col space-y-1.5 p-6', className) }))
CardHeader.displayName = 'CardHeader'

const CardTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(({ className, ...props }, ref) => React.createElement('h3', { ...props, ref, className: cn('text-base font-semibold leading-none tracking-tight', className) }))
CardTitle.displayName = 'CardTitle'

const CardDescription = React.forwardRef<HTMLParagraphElement, React.HTMLAttributes<HTMLParagraphElement>>(({ className, ...props }, ref) => React.createElement('p', { ...props, ref, className: cn('text-sm text-[hsl(var(--muted-foreground))]', className) }))
CardDescription.displayName = 'CardDescription'

const CardContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, ...props }, ref) => React.createElement('div', { ...props, ref, className: cn('p-6 pt-0', className) }))
CardContent.displayName = 'CardContent'

const CardFooter = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, ...props }, ref) => React.createElement('div', { ...props, ref, className: cn('flex items-center p-6 pt-0', className) }))
CardFooter.displayName = 'CardFooter'

export { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter }
