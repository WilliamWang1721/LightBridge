import * as React from 'react'
import { cn } from '../lib/utils'

const Table = React.forwardRef<HTMLTableElement, React.TableHTMLAttributes<HTMLTableElement>>(({ className, ...props }, ref) => React.createElement('table', { ...props, ref, className: cn('w-full caption-bottom text-sm', className) }))
Table.displayName = 'Table'
const TableHeader = React.forwardRef<HTMLTableSectionElement, React.HTMLAttributes<HTMLTableSectionElement>>(({ className, ...props }, ref) => React.createElement('thead', { ...props, ref, className: cn('[&_tr]:border-b', className) }))
TableHeader.displayName = 'TableHeader'
const TableBody = React.forwardRef<HTMLTableSectionElement, React.HTMLAttributes<HTMLTableSectionElement>>(({ className, ...props }, ref) => React.createElement('tbody', { ...props, ref, className: cn('[&_tr:last-child]:border-0', className) }))
TableBody.displayName = 'TableBody'
const TableFooter = React.forwardRef<HTMLTableSectionElement, React.HTMLAttributes<HTMLTableSectionElement>>(({ className, ...props }, ref) => React.createElement('tfoot', { ...props, ref, className: cn('border-t bg-[hsl(var(--muted)/.45)] font-medium', className) }))
TableFooter.displayName = 'TableFooter'
const TableRow = React.forwardRef<HTMLTableRowElement, React.HTMLAttributes<HTMLTableRowElement>>(({ className, ...props }, ref) => React.createElement('tr', { ...props, ref, className: cn('border-b transition-colors hover:bg-[hsl(var(--muted)/.35)]', className) }))
TableRow.displayName = 'TableRow'
const TableHead = React.forwardRef<HTMLTableCellElement, React.ThHTMLAttributes<HTMLTableCellElement>>(({ className, ...props }, ref) => React.createElement('th', { ...props, ref, className: cn('h-10 px-4 text-left align-middle text-xs font-semibold text-[hsl(var(--muted-foreground))]', className) }))
TableHead.displayName = 'TableHead'
const TableCell = React.forwardRef<HTMLTableCellElement, React.TdHTMLAttributes<HTMLTableCellElement>>(({ className, ...props }, ref) => React.createElement('td', { ...props, ref, className: cn('p-4 align-middle', className) }))
TableCell.displayName = 'TableCell'

export { Table, TableBody, TableCell, TableFooter, TableHead, TableHeader, TableRow }
