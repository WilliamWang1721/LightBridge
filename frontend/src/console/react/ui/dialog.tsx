import * as React from 'react'
import { cn } from '../lib/utils'

interface DialogContextValue {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const DialogContext = React.createContext<DialogContextValue | null>(null)

function useDialogContext() {
  const context = React.useContext(DialogContext)
  if (!context) throw new Error('Dialog components must be used inside Dialog')
  return context
}

function Dialog({ open, onOpenChange, children }: { open: boolean; onOpenChange: (open: boolean) => void; children?: React.ReactNode }) {
  return React.createElement(DialogContext.Provider, { value: { open, onOpenChange } }, children)
}

const DialogContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, children, onClick, ...props }, ref) => {
  const { open, onOpenChange } = useDialogContext()
  React.useEffect(() => {
    if (!open) return undefined
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onOpenChange(false)
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [open, onOpenChange])

  if (!open) return null
  return React.createElement('div', {
    className: 'fixed inset-0 z-[var(--ui-z-dialog,100)] flex min-h-[100dvh] items-center justify-center overflow-y-auto bg-[hsl(var(--foreground)/.55)] p-3 backdrop-blur-[4px] sm:p-6',
    role: 'dialog',
    'aria-modal': 'true',
    onMouseDown: (event: React.MouseEvent<HTMLDivElement>) => {
      if (event.target === event.currentTarget) onOpenChange(false)
    },
  }, React.createElement('div', {
    ...props,
    ref,
    tabIndex: -1,
    className: cn('relative max-h-[calc(100dvh-1.5rem)] w-full max-w-lg overflow-y-auto rounded-[calc(var(--ui-radius)+.375rem)] border border-[hsl(var(--border))] bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))] shadow-[0_24px_80px_hsl(var(--foreground)/.3)] outline-none sm:max-h-[calc(100dvh-3rem)]', className),
    onClick: (event: React.MouseEvent<HTMLDivElement>) => {
      event.stopPropagation()
      onClick?.(event)
    },
  }, children))
})
DialogContent.displayName = 'DialogContent'

const DialogHeader = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, ...props }, ref) => React.createElement('div', { ...props, ref, className: cn('flex flex-col space-y-1.5 border-b border-[hsl(var(--border))] px-5 py-4', className) }))
DialogHeader.displayName = 'DialogHeader'

const DialogTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(({ className, ...props }, ref) => React.createElement('h2', { ...props, ref, className: cn('text-base font-semibold text-[hsl(var(--foreground))]', className) }))
DialogTitle.displayName = 'DialogTitle'

const DialogDescription = React.forwardRef<HTMLParagraphElement, React.HTMLAttributes<HTMLParagraphElement>>(({ className, ...props }, ref) => React.createElement('p', { ...props, ref, className: cn('text-sm text-[hsl(var(--muted-foreground))]', className) }))
DialogDescription.displayName = 'DialogDescription'

const DialogFooter = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, ...props }, ref) => React.createElement('div', { ...props, ref, className: cn('flex flex-col-reverse gap-2 border-t border-[hsl(var(--border))] px-5 py-4 sm:flex-row sm:justify-end', className) }))
DialogFooter.displayName = 'DialogFooter'

const DialogClose = React.forwardRef<HTMLButtonElement, React.ButtonHTMLAttributes<HTMLButtonElement>>(({ className, onClick, type = 'button', ...props }, ref) => {
  const { onOpenChange } = useDialogContext()
  return React.createElement('button', { ...props, ref, type, className: cn('rounded-[var(--ui-radius)] p-2 text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]', className), onClick: (event: React.MouseEvent<HTMLButtonElement>) => { onClick?.(event); if (!event.defaultPrevented) onOpenChange(false) } })
})
DialogClose.displayName = 'DialogClose'

export { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle }
