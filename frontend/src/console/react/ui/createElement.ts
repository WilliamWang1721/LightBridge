import { createElement, type ElementType, type ReactNode } from 'react'
import { Button } from './button'
import { Card } from './card'
import { Input } from './input'
import { Label } from './label'
import { Select } from './select'
import { Table, TableBody, TableCell, TableFooter, TableHead, TableHeader, TableRow } from './table'
import { Textarea } from './textarea'

const primitiveMap: Record<string, ElementType> = {
  button: Button,
  input: Input,
  label: Label,
  select: Select,
  table: Table,
  tbody: TableBody,
  td: TableCell,
  tfoot: TableFooter,
  th: TableHead,
  thead: TableHeader,
  textarea: Textarea,
  tr: TableRow,
}

export function createShadcnElement<K extends keyof React.JSX.IntrinsicElements>(
  type: K,
  props?: Record<string, unknown> & React.Attributes | null,
  ...children: ReactNode[]
): React.ReactElement | null
export function createShadcnElement<P>(
  type: React.JSXElementConstructor<P>,
  props?: P & React.Attributes,
  ...children: ReactNode[]
): React.ReactElement | null
export function createShadcnElement(type: ElementType | string, props?: unknown, ...children: ReactNode[]) {
  const className = props && typeof props === 'object' && 'className' in props && typeof props.className === 'string' ? props.className : ''
  const isCard = ['div', 'section', 'article'].includes(type as string) && /(^|\s)card(?:\s|$)/.test(className)
  return createElement(isCard ? Card : primitiveMap[type as string] || type, props as never, ...children)
}
