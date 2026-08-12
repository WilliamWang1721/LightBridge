import { useEffect, useState, type ReactNode } from 'react'
import type { OrderStatus, PaymentOrder } from '@/types/payment'
import { createShadcnElement as h } from './ui/createElement'
import { Badge as ShadcnBadge, Button as ShadcnButton, Dialog as ShadcnDialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle, Select as ShadcnSelect } from './ui'

export type PaymentOrderRow = PaymentOrder & {
  user_email?: string
  user_name?: string
  user_notes?: string
}

export interface PaymentOrderAuditLog {
  id: number
  action: string
  detail: string | null
  operator: string | null
  created_at: string
}

export interface PaymentOrdersPageCopy {
  searchOrders: string
  allStatuses: string
  allPaymentTypes: string
  allOrderTypes: string
  balanceOrder: string
  subscriptionOrder: string
  orderId: string
  orderNo: string
  user: string
  payAmount: string
  paymentMethod: string
  status: string
  createdAt: string
  actions: string
  view: string
  cancel: string
  retry: string
  approveRefund: string
  retryRefund: string
  refund: string
  orderDetail: string
  amount: string
  feeRate: string
  expiresAt: string
  paidAt: string
  refundAmount: string
  refundReason: string
  refundRequestInfo: string
  refundRequestedAt: string
  refundRequestedBy: string
  refundRequestReason: string
  auditLogs: string
  operator: string
  refundOrder: string
  creditedAmount: string
  deductBalance: string
  deductBalanceHint: string
  maxRefundable: string
  refundReasonPlaceholder: string
  forceRefund: string
  confirmRefund: string
  processing: string
  cancelAction: string
  previous: string
  next: string
  showing: string
  to: string
  of: string
  results: string
  perPage: string
  pageOf: (values: { page: number; total: number }) => string
  noOrders: string
  noData: string
  paymentMethodLabel: (type: string) => string
  statusLabel: (status: string) => string
}

export interface PaymentOrdersPageProps {
  orders: PaymentOrderRow[]
  loading: boolean
  search: string
  filters: { status: string; paymentType: string; orderType: string }
  statusOptions: Array<{ value: string; label: string }>
  paymentTypeOptions: Array<{ value: string; label: string }>
  orderTypeOptions: Array<{ value: string; label: string }>
  page: number
  pageSize: number
  total: number
  selectedOrder: PaymentOrder | null
  auditLogs: PaymentOrderAuditLog[]
  showDetailDialog: boolean
  showRefundDialog: boolean
  refundSubmitting: boolean
  copy: PaymentOrdersPageCopy
  formatDate: (value: string) => string
  onSearch: (value: string) => void
  onFilterChange: (key: 'status' | 'paymentType' | 'orderType', value: string) => void
  onRefresh: () => void
  onViewOrder: (order: PaymentOrder) => void
  onCancelOrder: (order: PaymentOrder) => void
  onRetryOrder: (order: PaymentOrder) => void
  onOpenRefund: (order: PaymentOrder) => void
  onCloseDetail: () => void
  onCloseRefund: () => void
  onRefund: (data: { amount: number; reason: string; deduct_balance: boolean; force: boolean }) => void
  onPageChange: (page: number) => void
  onPageSizeChange: (size: number) => void
}

function Icon({ name }: { name: 'eye' | 'x' | 'refresh' | 'check' | 'dollar' | 'chevronLeft' | 'chevronRight' }): ReactNode {
  const paths = {
    eye: 'M2.25 12s3.75-6.75 9.75-6.75S21.75 12 21.75 12 18 18.75 12 18.75 2.25 12 2.25 12Z M12 15.25a3.25 3.25 0 1 0 0-6.5 3.25 3.25 0 0 0 0 6.5Z',
    x: 'm6.75 6.75 10.5 10.5m0-10.5-10.5 10.5',
    refresh: 'M20.25 12a8.25 8.25 0 1 1-2.42-5.83M20.25 4.5v5.25H15',
    check: 'm5.25 12.75 4.5 4.5 9-10.5',
    dollar: 'M12 6.75v10.5m3-8.25c-.62-.77-1.56-1.25-2.63-1.25h-.74A2.63 2.63 0 0 0 9 10.38c0 1.45 1.18 2.62 2.63 2.62h.74A2.63 2.63 0 0 1 15 15.62 2.63 2.63 0 0 1 12.37 18h-.74A3.38 3.38 0 0 1 9 16.75',
    chevronLeft: 'm14.25 18-6-6 6-6',
    chevronRight: 'm9.75 18 6-6-6-6',
  }[name]
  return h('svg', {
    className: 'h-4 w-4',
    fill: 'none',
    viewBox: '0 0 24 24',
    stroke: 'currentColor',
    strokeWidth: 1.7,
    strokeLinecap: 'round',
    strokeLinejoin: 'round',
    'aria-hidden': 'true',
  }, h('path', { d: paths }))
}

function Button({
  children,
  className = '',
  ...props
}: { children?: ReactNode; className?: string; type?: 'button' | 'submit'; disabled?: boolean; onClick?: () => void; title?: string }) {
  return h(ShadcnButton, {
    type: props.type || 'button',
    disabled: props.disabled,
    onClick: props.onClick,
    title: props.title,
    size: 'sm',
    className,
  }, children)
}

function StatusBadge({ status, copy }: { status: OrderStatus; copy: PaymentOrdersPageCopy }): ReactNode {
  const muted = ['EXPIRED', 'CANCELLED'].includes(status)
  const destructive = ['FAILED', 'REFUND_FAILED'].includes(status)
  const emphasis = ['REFUND_REQUESTED', 'REFUNDING'].includes(status)
  const className = destructive
    ? 'border-[hsl(var(--destructive)/.25)] bg-[hsl(var(--destructive)/.1)] text-[hsl(var(--destructive))]'
    : emphasis
      ? 'border-[hsl(var(--foreground)/.2)] bg-[hsl(var(--muted))] text-[hsl(var(--foreground))]'
      : muted
        ? 'border-[hsl(var(--border))] bg-[hsl(var(--muted))] text-[hsl(var(--muted-foreground))]'
        : 'border-[hsl(var(--border))] bg-[hsl(var(--background))] text-[hsl(var(--foreground))]'
  return h(ShadcnBadge, { variant: 'outline', className }, copy.statusLabel(status))
}

function SelectField({
  value,
  options,
  onChange,
}: {
  value: string
  options: Array<{ value: string; label: string }>
  onChange: (value: string) => void
}): ReactNode {
  return h(ShadcnSelect, {
    value,
    onChange: (event: { target: { value: string } }) => onChange(event.target.value),
    className: 'input h-10 min-w-32 appearance-none',
  }, options.map((option) => h('option', { key: option.value, value: option.value }, option.label)))
}

function Modal({
  open,
  title,
  onClose,
  children,
  footer,
  wide = false,
}: {
  open: boolean
  title: string
  onClose: () => void
  children?: ReactNode
  footer?: ReactNode
  wide?: boolean
}): ReactNode {
  if (!open) return null
  return h(ShadcnDialog, { open, onOpenChange: (nextOpen) => { if (!nextOpen) onClose() } }, h(DialogContent, { className: wide ? 'max-w-4xl' : 'max-w-lg' }, [
    h(DialogHeader, { key: 'header', className: 'flex-row items-center justify-between' }, [
      h(DialogTitle, { key: 'title' }, title),
      h(DialogClose, { key: 'close', 'aria-label': 'Close' }, h(Icon, { name: 'x' })),
    ]),
    h('div', { key: 'body', className: 'min-h-0 max-h-[calc(100dvh-12rem)] overflow-y-auto p-5' }, children),
    footer ? h(DialogFooter, { key: 'footer' }, footer) : null,
  ]))
}

function OrdersTable({ props }: { props: PaymentOrdersPageProps }): ReactNode {
  const { orders, loading, copy } = props
  const headers = [copy.orderId, copy.orderNo, copy.user, copy.payAmount, copy.paymentMethod, copy.status, copy.createdAt, copy.actions]
  return h('section', { className: 'card overflow-hidden' }, [
    h('div', { key: 'table', className: 'overflow-x-auto' }, h('table', { className: 'min-w-[980px] w-full text-left' }, [
      h('thead', { key: 'head', className: 'border-b border-[hsl(var(--border))] bg-[hsl(var(--muted)/.45)]' }, h('tr', null, headers.map((label) => h('th', { key: label, scope: 'col', className: 'px-4 py-3 text-xs font-semibold text-[hsl(var(--muted-foreground))]' }, label)))),
      h('tbody', { key: 'body', className: 'divide-y divide-[hsl(var(--border))]' }, loading
        ? h('tr', null, h('td', { colSpan: headers.length, className: 'px-4 py-12 text-center text-sm text-[hsl(var(--muted-foreground))]' }, h('span', { className: 'mx-auto block h-7 w-7 animate-spin rounded-full border-2 border-[hsl(var(--border))] border-t-[hsl(var(--foreground))]' })))
        : orders.length
          ? orders.map((order) => h('tr', { key: order.id, className: 'align-middle transition-colors hover:bg-[hsl(var(--muted)/.35)]' }, [
            h('td', { key: 'id', className: 'px-4 py-3 font-mono text-sm text-[hsl(var(--foreground))]' }, `#${order.id}`),
            h('td', { key: 'no', className: 'px-4 py-3 text-sm text-[hsl(var(--foreground))]' }, order.out_trade_no),
            h('td', { key: 'user', className: 'px-4 py-3 text-sm text-[hsl(var(--foreground))]' }, [order.user_email || order.user_name || `#${order.user_id}`, order.user_notes ? h('span', { key: 'notes', className: 'ml-1 text-xs text-[hsl(var(--muted-foreground))]' }, `(${order.user_notes})`) : null]),
            h('td', { key: 'pay', className: 'px-4 py-3 text-sm text-[hsl(var(--foreground))]' }, [h('span', { key: 'amount', className: 'font-medium' }, `¥${order.pay_amount.toFixed(2)}`), order.fee_rate > 0 ? h('span', { key: 'fee', className: 'ml-1 text-xs text-[hsl(var(--muted-foreground))]' }, `(${copy.feeRate} ${order.fee_rate}%)`) : null, order.amount !== order.pay_amount ? h('div', { key: 'credited', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, `${copy.creditedAmount}: ${order.order_type === 'balance' ? '$' : '¥'}${order.amount.toFixed(2)}`) : null]),
            h('td', { key: 'method', className: 'px-4 py-3 text-sm text-[hsl(var(--muted-foreground))]' }, copy.paymentMethodLabel(order.payment_type)),
            h('td', { key: 'status', className: 'px-4 py-3' }, h(StatusBadge, { status: order.status, copy })),
            h('td', { key: 'created', className: 'px-4 py-3 text-xs text-[hsl(var(--muted-foreground))]' }, props.formatDate(order.created_at)),
            h('td', { key: 'actions', className: 'px-4 py-3' }, h('div', { className: 'flex items-center gap-1' }, [
              h(Button, { key: 'view', className: 'text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]', title: copy.view, onClick: () => props.onViewOrder(order) }, [h(Icon, { key: 'icon', name: 'eye' }), copy.view]),
              order.status === 'PENDING' ? h(Button, { key: 'cancel', className: 'text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]', title: copy.cancel, onClick: () => props.onCancelOrder(order) }, [h(Icon, { key: 'icon', name: 'x' }), copy.cancel]) : null,
              order.status === 'FAILED' ? h(Button, { key: 'retry', className: 'text-[hsl(var(--foreground))] hover:bg-[hsl(var(--muted))]', title: copy.retry, onClick: () => props.onRetryOrder(order) }, [h(Icon, { key: 'icon', name: 'refresh' }), copy.retry]) : null,
              order.status === 'REFUND_REQUESTED' ? h(Button, { key: 'approve', className: 'text-[hsl(var(--foreground))] hover:bg-[hsl(var(--muted))]', title: copy.approveRefund, onClick: () => props.onOpenRefund(order) }, [h(Icon, { key: 'icon', name: 'check' }), copy.approveRefund]) : null,
              order.status === 'REFUND_FAILED' ? h(Button, { key: 'retry-refund', className: 'text-[hsl(var(--foreground))] hover:bg-[hsl(var(--muted))]', title: copy.retryRefund, onClick: () => props.onOpenRefund(order) }, [h(Icon, { key: 'icon', name: 'refresh' }), copy.retryRefund]) : null,
              ['COMPLETED', 'PARTIALLY_REFUNDED'].includes(order.status) ? h(Button, { key: 'refund', className: 'text-[hsl(var(--destructive))] hover:bg-[hsl(var(--destructive)/.1)]', title: copy.refund, onClick: () => props.onOpenRefund(order) }, [h(Icon, { key: 'icon', name: 'dollar' }), copy.refund]) : null,
            ])),
          ]))
          : h('tr', null, h('td', { colSpan: headers.length, className: 'px-4 py-12 text-center text-sm text-[hsl(var(--muted-foreground))]' }, copy.noOrders)),
      ),
    ])),
  ])
}

function DetailDialog({ props }: { props: PaymentOrdersPageProps }): ReactNode {
  const { selectedOrder: order, copy } = props
  if (!order) return null
  const field = (label: string, value: ReactNode, key: string) => h('div', { key }, [h('p', { key: 'label', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, label), h('div', { key: 'value', className: 'mt-1 text-sm text-[hsl(var(--foreground))]' }, value)])
  return h(Modal, { open: props.showDetailDialog, title: copy.orderDetail, onClose: props.onCloseDetail, wide: true }, [
    h('div', { key: 'fields', className: 'grid grid-cols-1 gap-4 sm:grid-cols-2' }, [
      field(copy.orderId, h('span', { className: 'font-mono font-medium' }, `#${order.id}`), 'id'),
      field(copy.orderNo, h('span', { className: 'font-medium' }, order.out_trade_no), 'no'),
      field(copy.status, h(StatusBadge, { status: order.status, copy }), 'status'),
      field(copy.amount, `${order.order_type === 'balance' ? '$' : '¥'}${order.amount.toFixed(2)}`, 'amount'),
      field(copy.payAmount, `¥${order.pay_amount.toFixed(2)}`, 'pay'),
      field(copy.paymentMethod, copy.paymentMethodLabel(order.payment_type), 'method'),
      field(copy.feeRate, `${order.fee_rate}%`, 'fee'),
      field(copy.createdAt, props.formatDate(order.created_at), 'created'),
      field(copy.expiresAt, props.formatDate(order.expires_at), 'expires'),
      order.paid_at ? field(copy.paidAt, props.formatDate(order.paid_at), 'paid') : null,
      order.refund_amount ? field(copy.refundAmount, `${order.order_type === 'balance' ? '$' : '¥'}${order.refund_amount.toFixed(2)}`, 'refund-amount') : null,
      order.refund_reason ? field(copy.refundReason, order.refund_reason, 'refund-reason') : null,
      order.refund_requested_at ? h('div', { key: 'refund-request', className: 'col-span-full border-t border-[hsl(var(--border))] pt-4' }, [
        h('p', { key: 'title', className: 'mb-3 text-xs font-semibold text-[hsl(var(--foreground))]' }, copy.refundRequestInfo),
        h('div', { key: 'content', className: 'grid grid-cols-1 gap-3 sm:grid-cols-2' }, [
          field(copy.refundRequestedAt, props.formatDate(order.refund_requested_at), 'requested-at'),
          field(copy.refundRequestedBy, `#${order.refund_requested_by ?? '-'}`, 'requested-by'),
          order.refund_request_reason ? field(copy.refundRequestReason, order.refund_request_reason, 'request-reason') : null,
        ]),
      ]) : null,
    ]),
    props.auditLogs.length ? h('div', { key: 'logs', className: 'mt-6 border-t border-[hsl(var(--border))] pt-4' }, [
      h('p', { key: 'title', className: 'mb-2 text-xs font-semibold text-[hsl(var(--muted-foreground))]' }, copy.auditLogs),
      h('div', { key: 'items', className: 'max-h-48 space-y-2 overflow-y-auto' }, props.auditLogs.map((log) => h('div', { key: log.id, className: 'rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--muted)/.45)] p-3' }, [
        h('div', { key: 'meta', className: 'flex items-center justify-between gap-3' }, [h('span', { key: 'action', className: 'text-xs font-medium text-[hsl(var(--foreground))]' }, log.action), h('span', { key: 'date', className: 'text-xs text-[hsl(var(--muted-foreground))]' }, props.formatDate(log.created_at))]),
        log.detail ? h('div', { key: 'detail', className: 'mt-1 break-all text-xs text-[hsl(var(--muted-foreground))]' }, log.detail) : null,
        log.operator ? h('div', { key: 'operator', className: 'mt-1 text-xs text-[hsl(var(--muted-foreground))]' }, `${copy.operator}: ${log.operator}`) : null,
      ]))),
    ]) : null,
  ])
}

function RefundDialog({ props }: { props: PaymentOrdersPageProps }): ReactNode {
  const order = props.selectedOrder
  const [amount, setAmount] = useState(0)
  const [reason, setReason] = useState('')
  const [deductBalance, setDeductBalance] = useState(true)
  const [force, setForce] = useState(false)

  useEffect(() => {
    if (!props.showRefundDialog || !order) return
    const nextAmount = order.status === 'REFUND_REQUESTED' && order.refund_amount ? order.refund_amount : order.amount
    setAmount(nextAmount)
    setReason(order.refund_request_reason || '')
    setDeductBalance(true)
    setForce(false)
  }, [props.showRefundDialog, order])

  if (!order) return null
  const maxRefundable = order.amount - (['PARTIALLY_REFUNDED', 'REFUNDED'].includes(order.status) ? order.refund_amount || 0 : 0)
  const valid = amount > 0 && amount <= maxRefundable && reason.trim().length > 0
  return h(Modal, {
    open: props.showRefundDialog,
    title: props.copy.refundOrder,
    onClose: props.onCloseRefund,
    footer: [
      h(Button, { key: 'cancel', className: 'border border-[hsl(var(--border))] text-[hsl(var(--foreground))] hover:bg-[hsl(var(--muted))]', onClick: props.onCloseRefund }, props.copy.cancelAction),
      h(Button, { key: 'confirm', type: 'submit', disabled: props.refundSubmitting || !valid, className: 'bg-[hsl(var(--foreground))] text-[hsl(var(--background))] hover:opacity-85', onClick: () => props.onRefund({ amount, reason: reason.trim(), deduct_balance: deductBalance, force }) }, props.refundSubmitting ? props.copy.processing : props.copy.confirmRefund),
    ],
  }, h('form', { className: 'space-y-4', onSubmit: (event: { preventDefault: () => void }) => { event.preventDefault(); if (valid) props.onRefund({ amount, reason: reason.trim(), deduct_balance: deductBalance, force }) } }, [
    order.refund_requested_at || order.refund_request_reason ? h('div', { key: 'request', className: 'rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--muted)/.55)] p-3 text-sm' }, [h('p', { key: 'title', className: 'font-medium text-[hsl(var(--foreground))]' }, props.copy.refundRequestInfo), order.refund_requested_at ? h('p', { key: 'date', className: 'mt-1 text-[hsl(var(--muted-foreground))]' }, `${props.copy.refundRequestedAt}: ${props.formatDate(order.refund_requested_at)}`) : null, order.refund_request_reason ? h('p', { key: 'reason', className: 'mt-1 text-[hsl(var(--muted-foreground))]' }, `${props.copy.refundRequestReason}: ${order.refund_request_reason}`) : null]) : null,
    h('div', { key: 'summary', className: 'rounded-xl bg-[hsl(var(--muted)/.6)] p-3 text-sm' }, [h('div', { key: 'id', className: 'flex justify-between gap-4' }, [h('span', { key: 'label', className: 'text-[hsl(var(--muted-foreground))]' }, props.copy.orderId), h('span', { key: 'value', className: 'font-mono text-[hsl(var(--foreground))]' }, `#${order.id}`)]), h('div', { key: 'credited', className: 'mt-1 flex justify-between gap-4' }, [h('span', { key: 'label', className: 'text-[hsl(var(--muted-foreground))]' }, props.copy.creditedAmount), h('span', { key: 'value', className: 'font-medium text-[hsl(var(--foreground))]' }, `${order.order_type === 'balance' ? '$' : '¥'}${order.amount.toFixed(2)}`)]), h('div', { key: 'pay', className: 'mt-1 flex justify-between gap-4' }, [h('span', { key: 'label', className: 'text-[hsl(var(--muted-foreground))]' }, props.copy.payAmount), h('span', { key: 'value', className: 'font-medium text-[hsl(var(--foreground))]' }, `¥${order.pay_amount.toFixed(2)}`)])]),
    h('label', { key: 'deduct', className: 'flex items-start gap-2 text-sm text-[hsl(var(--foreground))]' }, [h('input', { key: 'input', type: 'checkbox', checked: deductBalance, onChange: (event: { target: { checked: boolean } }) => setDeductBalance(event.target.checked), className: 'mt-0.5 h-4 w-4 rounded border-[hsl(var(--border))]' }), h('span', { key: 'copy' }, [props.copy.deductBalance, h('span', { key: 'hint', className: 'ml-1 text-xs text-[hsl(var(--muted-foreground))]' }, props.copy.deductBalanceHint)])]),
    h('label', { key: 'amount', className: 'block' }, [h('span', { key: 'label', className: 'input-label' }, props.copy.refundAmount), h('input', { key: 'input', type: 'number', min: '0.01', max: String(maxRefundable), step: '0.01', value: amount, onChange: (event: { target: { value: string } }) => setAmount(Number(event.target.value)), className: 'input mt-1', required: true }), h('span', { key: 'hint', className: 'mt-1 block text-xs text-[hsl(var(--muted-foreground))]' }, `${props.copy.maxRefundable}: ${order.order_type === 'balance' ? '$' : '¥'}${maxRefundable.toFixed(2)}`)]),
    h('label', { key: 'reason', className: 'block' }, [h('span', { key: 'label', className: 'input-label' }, props.copy.refundReason), h('textarea', { key: 'input', rows: 3, value: reason, onChange: (event: { target: { value: string } }) => setReason(event.target.value), placeholder: props.copy.refundReasonPlaceholder, className: 'input mt-1 resize-y', required: true })]),
    force ? h('label', { key: 'force', className: 'flex items-center gap-2 text-sm font-medium text-[hsl(var(--destructive))]' }, [h('input', { key: 'input', type: 'checkbox', checked: force, onChange: (event: { target: { checked: boolean } }) => setForce(event.target.checked), className: 'h-4 w-4 rounded border-[hsl(var(--border))]' }), props.copy.forceRefund]) : null,
  ]))
}

function Pagination({ props }: { props: PaymentOrdersPageProps }): ReactNode {
  const totalPages = Math.max(1, Math.ceil(props.total / props.pageSize))
  if (props.total <= 0) return null
  return h('div', { className: 'flex flex-col gap-3 border-t border-[hsl(var(--border))] px-4 py-3 text-sm text-[hsl(var(--muted-foreground))] sm:flex-row sm:items-center sm:justify-between' }, [
    h('p', { key: 'summary' }, `${props.copy.showing} ${(props.page - 1) * props.pageSize + 1} ${props.copy.to} ${Math.min(props.page * props.pageSize, props.total)} ${props.copy.of} ${props.total} ${props.copy.results}`),
    h('div', { key: 'controls', className: 'flex items-center gap-2' }, [
      h('span', { key: 'size-label', className: 'hidden sm:inline' }, `${props.copy.perPage}:`),
      h('select', { key: 'size', value: props.pageSize, onChange: (event: { target: { value: string } }) => props.onPageSizeChange(Number(event.target.value)), className: 'input h-9 w-20' }, [10, 20, 50, 100].map((size) => h('option', { key: size, value: size }, size))),
      h('span', { key: 'page', className: 'px-1' }, props.copy.pageOf({ page: props.page, total: totalPages })),
      h(Button, { key: 'previous', className: 'border border-[hsl(var(--border))] text-[hsl(var(--foreground))] hover:bg-[hsl(var(--muted))]', disabled: props.page <= 1, title: props.copy.previous, onClick: () => props.onPageChange(props.page - 1) }, h(Icon, { name: 'chevronLeft' })),
      h(Button, { key: 'next', className: 'border border-[hsl(var(--border))] text-[hsl(var(--foreground))] hover:bg-[hsl(var(--muted))]', disabled: props.page >= totalPages, title: props.copy.next, onClick: () => props.onPageChange(props.page + 1) }, h(Icon, { name: 'chevronRight' })),
    ]),
  ])
}

export default function PaymentOrdersPage(props: PaymentOrdersPageProps) {
  return h('div', { className: 'space-y-4' }, [
    h('section', { key: 'filters', className: 'card p-4' }, h('div', { className: 'flex flex-wrap items-center gap-3' }, [
      h('input', { key: 'search', value: props.search, onChange: (event: { target: { value: string } }) => props.onSearch(event.target.value), type: 'search', placeholder: props.copy.searchOrders, className: 'input min-w-48 flex-1 sm:max-w-64' }),
      h(SelectField, { key: 'status', value: props.filters.status, options: props.statusOptions, onChange: (value) => props.onFilterChange('status', value) }),
      h(SelectField, { key: 'payment', value: props.filters.paymentType, options: props.paymentTypeOptions, onChange: (value) => props.onFilterChange('paymentType', value) }),
      h(SelectField, { key: 'order', value: props.filters.orderType, options: props.orderTypeOptions, onChange: (value) => props.onFilterChange('orderType', value) }),
      h('div', { key: 'refresh', className: 'ml-auto' }, h(Button, { className: 'btn btn-secondary', title: 'Refresh', disabled: props.loading, onClick: props.onRefresh }, h(Icon, { name: 'refresh' }))),
    ])),
    h(OrdersTable, { key: 'orders', props }),
    h(Pagination, { key: 'pagination', props }),
    h(DetailDialog, { key: 'detail', props }),
    h(RefundDialog, { key: 'refund', props }),
  ])
}
