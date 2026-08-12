import { useState, type ReactNode } from 'react'
import type { PaymentOrder } from '@/types/payment'
import { createShadcnElement as h } from './ui/createElement'
import { Badge as ShadcnBadge, Dialog as ShadcnDialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle } from './ui'

export interface UserOrdersPageProps {
  orders: PaymentOrder[]
  loading: boolean
  actionLoading: boolean
  currentFilter: string
  statusFilters: Array<{ value: string; label: string }>
  page: number
  pageSize: number
  total: number
  refundEligible: (order: PaymentOrder) => boolean
  copy: {
    all: string
    refresh: string
    recharge: string
    orderId: string
    orderNo: string
    payAmount: string
    paymentMethod: string
    status: string
    createdAt: string
    actions: string
    cancel: string
    requestRefund: string
    confirmCancel: string
    refundReason: string
    refundReasonPlaceholder: string
    orderAmount: string
    close: string
    processing: string
    save: string
    noOrders: string
    previous: string
    next: string
    pageOf: (page: number, total: number) => string
  }
  onFilterChange: (value: string) => void
  onRefresh: () => void
  onRecharge: () => void
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
  onCancel: (id: number) => void
  onRefund: (order: PaymentOrder, reason: string) => void
}

const date = (value: string) => new Date(value).toLocaleString()
const currency = (order: PaymentOrder) => `${order.currency || '¥'}${order.pay_amount.toFixed(2)}`

function Status({ value }: { value: string }): ReactNode {
  return h(ShadcnBadge, { variant: 'outline' }, value)
}

function Dialog({ title, children, footer, onClose }: { title: string; children?: ReactNode; footer: ReactNode; onClose: () => void }): ReactNode {
  return h(ShadcnDialog, { open: true, onOpenChange: (open) => { if (!open) onClose() } }, h(DialogContent, { className: 'max-w-lg' }, [
    h(DialogHeader, { key: 'heading', className: 'flex-row items-center justify-between' }, [h(DialogTitle, { key: 'title' }, title), h(DialogClose, { key: 'close', 'aria-label': title }, '×')]),
    h('div', { key: 'body', className: 'p-5' }, children),
    h(DialogFooter, { key: 'footer' }, footer),
  ]))
}

function Pagination({ page, pageSize, total, copy, onPageChange, onPageSizeChange }: Pick<UserOrdersPageProps, 'page' | 'pageSize' | 'total' | 'copy' | 'onPageChange' | 'onPageSizeChange'>): ReactNode {
  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  return h('div', { className: 'flex flex-wrap items-center justify-between gap-3 border-t border-[hsl(var(--border))] px-4 py-3 text-sm text-[hsl(var(--muted-foreground))]' }, [
    h('span', { key: 'info' }, copy.pageOf(page, totalPages)),
    h('div', { key: 'controls', className: 'flex items-center gap-2' }, [
      h('select', { key: 'size', value: pageSize, className: 'input h-9 w-20', 'aria-label': 'Page size', onChange: (event: { target: { value: string } }) => onPageSizeChange(Number(event.target.value)) }, [10, 20, 50].map((size) => h('option', { key: size, value: size }, size))),
      h('button', { key: 'previous', type: 'button', className: 'btn btn-secondary h-9', disabled: page <= 1, onClick: () => onPageChange(page - 1) }, copy.previous),
      h('button', { key: 'next', type: 'button', className: 'btn btn-secondary h-9', disabled: page >= totalPages, onClick: () => onPageChange(page + 1) }, copy.next),
    ]),
  ])
}

export default function UserOrdersPage({ orders, loading, actionLoading, currentFilter, statusFilters, page, pageSize, total, refundEligible, copy, onFilterChange, onRefresh, onRecharge, onPageChange, onPageSizeChange, onCancel, onRefund }: UserOrdersPageProps) {
  const [cancelTarget, setCancelTarget] = useState<PaymentOrder | null>(null)
  const [refundTarget, setRefundTarget] = useState<PaymentOrder | null>(null)
  const [refundReason, setRefundReason] = useState('')
  const table = loading
    ? h('div', { className: 'flex min-h-64 items-center justify-center text-sm text-[hsl(var(--muted-foreground))]', role: 'status' }, copy.refresh)
    : orders.length
      ? h('div', { className: 'overflow-x-auto' }, h('table', { className: 'w-full min-w-[920px] border-collapse text-sm' }, [
        h('thead', { key: 'head' }, h('tr', { className: 'border-b border-[hsl(var(--border))] bg-[hsl(var(--muted)/.45)] text-left text-xs font-medium uppercase tracking-wide text-[hsl(var(--muted-foreground))]' }, [copy.orderId, copy.orderNo, copy.payAmount, copy.paymentMethod, copy.status, copy.createdAt, copy.actions].map((label) => h('th', { key: label, className: 'px-4 py-3' }, label)))),
        h('tbody', { key: 'body' }, orders.map((order) => h('tr', { key: order.id, className: 'border-b border-[hsl(var(--border))] last:border-0' }, [
          h('td', { key: 'id', className: 'px-4 py-4 font-mono' }, `#${order.id}`),
          h('td', { key: 'no', className: 'px-4 py-4 text-[hsl(var(--foreground))]' }, order.out_trade_no),
          h('td', { key: 'amount', className: 'px-4 py-4' }, [h('span', { key: 'pay', className: 'font-medium text-[hsl(var(--foreground))]' }, currency(order)), order.amount !== order.pay_amount ? h('span', { key: 'credited', className: 'mt-1 block text-xs text-[hsl(var(--muted-foreground))]' }, `${copy.orderAmount}: ${order.order_type === 'balance' ? '$' : '¥'}${order.amount.toFixed(2)}`) : null]),
          h('td', { key: 'method', className: 'px-4 py-4 text-[hsl(var(--muted-foreground))]' }, order.payment_type),
          h('td', { key: 'status', className: 'px-4 py-4' }, h(Status, { value: order.status })),
          h('td', { key: 'created', className: 'px-4 py-4 text-xs text-[hsl(var(--muted-foreground))]' }, date(order.created_at)),
          h('td', { key: 'actions', className: 'px-4 py-4' }, h('div', { className: 'flex flex-wrap gap-2' }, [order.status === 'PENDING' ? h('button', { key: 'cancel', type: 'button', className: 'btn btn-secondary h-8 px-2 text-xs', onClick: () => setCancelTarget(order) }, copy.cancel) : null, refundEligible(order) ? h('button', { key: 'refund', type: 'button', className: 'btn btn-secondary h-8 px-2 text-xs', onClick: () => { setRefundTarget(order); setRefundReason('') } }, copy.requestRefund) : null])),
        ]))),
      ]))
      : h('div', { className: 'flex min-h-64 items-center justify-center p-8 text-sm text-[hsl(var(--muted-foreground))]' }, copy.noOrders)
  const cancelDialog = cancelTarget ? h(Dialog, { key: 'cancel-dialog', title: copy.cancel, onClose: () => setCancelTarget(null), footer: [
    h('button', { key: 'close', type: 'button', className: 'btn btn-secondary', onClick: () => setCancelTarget(null) }, copy.close),
    h('button', { key: 'confirm', type: 'button', className: 'btn btn-primary', disabled: actionLoading, onClick: () => { onCancel(cancelTarget.id); setCancelTarget(null) } }, actionLoading ? copy.processing : copy.cancel),
  ] }, h('p', { className: 'text-sm text-[hsl(var(--muted-foreground))]' }, copy.confirmCancel)) : null
  const refundDialog = refundTarget ? h(Dialog, { key: 'refund-dialog', title: copy.requestRefund, onClose: () => setRefundTarget(null), footer: [
    h('button', { key: 'close', type: 'button', className: 'btn btn-secondary', onClick: () => setRefundTarget(null) }, copy.close),
    h('button', { key: 'confirm', type: 'button', className: 'btn btn-primary', disabled: actionLoading || !refundReason.trim(), onClick: () => { onRefund(refundTarget, refundReason.trim()); setRefundTarget(null) } }, actionLoading ? copy.processing : copy.save),
  ] }, [
    h('div', { key: 'order', className: 'rounded-xl bg-[hsl(var(--muted)/.55)] p-4 text-sm' }, [
      h('div', { key: 'id', className: 'flex justify-between gap-3' }, [h('span', { className: 'text-[hsl(var(--muted-foreground))]' }, copy.orderId), h('span', { className: 'font-mono' }, `#${refundTarget.id}`)]),
      h('div', { key: 'amount', className: 'mt-2 flex justify-between gap-3' }, [h('span', { className: 'text-[hsl(var(--muted-foreground))]' }, copy.payAmount), h('span', {}, currency(refundTarget))]),
    ]),
    h('label', { key: 'reason', className: 'mt-4 block' }, [h('span', { className: 'mb-1 block text-sm font-medium' }, copy.refundReason), h('textarea', { value: refundReason, rows: 4, className: 'input w-full', placeholder: copy.refundReasonPlaceholder, onChange: (event: { target: { value: string } }) => setRefundReason(event.target.value) })]),
  ]) : null
  return h('div', { className: 'space-y-4' }, [
    h('div', { key: 'filters', className: 'card flex flex-wrap items-center gap-3 p-4' }, [
      h('select', { key: 'filter', value: currentFilter, className: 'input h-10 w-40', 'aria-label': copy.status, onChange: (event: { target: { value: string } }) => onFilterChange(event.target.value) }, statusFilters.map((option) => h('option', { key: option.value, value: option.value }, option.label))),
      h('div', { key: 'actions', className: 'ml-auto flex flex-wrap items-center justify-end gap-2' }, [h('button', { key: 'refresh', type: 'button', className: 'btn btn-secondary h-10', disabled: loading, onClick: onRefresh }, copy.refresh), h('button', { key: 'recharge', type: 'button', className: 'btn btn-primary h-10', onClick: onRecharge }, copy.recharge)]),
    ]),
    h('section', { key: 'table', className: 'card overflow-hidden' }, table),
    total > 0 ? h(Pagination, { key: 'pagination', page, pageSize, total, copy, onPageChange, onPageSizeChange }) : null,
    cancelDialog,
    refundDialog,
  ])
}
