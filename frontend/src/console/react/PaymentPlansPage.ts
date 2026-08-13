import { useEffect, useState, type ReactNode } from 'react'
import type { AdminGroup } from '@/types'
import type { SubscriptionPlan } from '@/types/payment'
import { createShadcnElement as h } from './ui/createElement'
import { AppIcon } from './ui/app-icon'
import { Button as ShadcnButton, Dialog as ShadcnDialog, DialogClose, DialogContent, DialogFooter, DialogHeader, DialogTitle, Switch as ShadcnSwitch } from './ui'

export interface PaymentPlanForm {
  name: string
  group_id: number | null
  description: string
  price: number
  original_price: number
  validity_days: number
  validity_unit: string
  sort_order: number
  for_sale: boolean
  features: string
}

export interface PaymentPlansPageCopy {
  refresh: string
  createPlan: string
  editPlan: string
  planName: string
  group: string
  groupMissing: string
  price: string
  originalPrice: string
  validityDays: string
  validityUnit: string
  days: string
  weeks: string
  months: string
  forSale: string
  sortOrder: string
  actions: string
  edit: string
  delete: string
  selectGroup: string
  planDescription: string
  dailyLimit: string
  weeklyLimit: string
  monthlyLimit: string
  unlimited: string
  features: string
  featuresPlaceholder: string
  featuresHint: string
  cancel: string
  save: string
  saving: string
  deletePlan: string
  deletePlanConfirm: string
  confirm: string
  noPlans: string
  groupRequired: string
  priceRequired: string
  validityDaysRequired: string
}

export interface PaymentPlansPageProps {
  plans: SubscriptionPlan[]
  groups: AdminGroup[]
  loading: boolean
  saving: boolean
  editingPlan: SubscriptionPlan | null
  showPlanDialog: boolean
  showDeleteDialog: boolean
  deletingPlanId: number | null
  copy: PaymentPlansPageCopy
  onRefresh: () => void
  onOpenEdit: (plan: SubscriptionPlan | null) => void
  onCloseEdit: () => void
  onToggleForSale: (plan: SubscriptionPlan) => void
  onOpenDelete: (plan: SubscriptionPlan) => void
  onCloseDelete: () => void
  onDelete: () => void
  onSavePlan: (form: PaymentPlanForm) => void
}

function Icon({ name }: { name: 'refresh' | 'edit' | 'trash' | 'x' }): ReactNode {
  return h(AppIcon, { name })
}

function Button({ children, className = '', ...props }: { children?: ReactNode; className?: string; type?: 'button' | 'submit'; disabled?: boolean; onClick?: () => void; title?: string }) {
  return h(ShadcnButton, { type: props.type || 'button', disabled: props.disabled, onClick: props.onClick, title: props.title, className }, children)
}

function Modal({ open, title, onClose, children, footer, wide = false }: { open: boolean; title: string; onClose: () => void; children?: ReactNode; footer?: ReactNode; wide?: boolean }): ReactNode {
  if (!open) return null
  return h(ShadcnDialog, { open, onOpenChange: (nextOpen) => { if (!nextOpen) onClose() } }, h(DialogContent, { className: wide ? 'max-w-4xl' : 'max-w-lg' }, [
    h(DialogHeader, { key: 'header', className: 'flex-row items-center justify-between' }, [h(DialogTitle, { key: 'title' }, title), h(DialogClose, { key: 'close', 'aria-label': 'Close' }, h(Icon, { name: 'x' }))]),
    h('div', { key: 'body', className: 'min-h-0 max-h-[calc(100dvh-12rem)] overflow-y-auto p-5' }, children),
    footer ? h(DialogFooter, { key: 'footer' }, footer) : null,
  ]))
}

function groupFor(groups: AdminGroup[], id: number) {
  return groups.find((group) => group.id === id)
}

function PlanFormDialog({ props }: { props: PaymentPlansPageProps }): ReactNode {
  const { copy, editingPlan } = props
  const [form, setForm] = useState<PaymentPlanForm>({ name: '', group_id: null, description: '', price: 0, original_price: 0, validity_days: 30, validity_unit: 'days', sort_order: 0, for_sale: true, features: '' })
  useEffect(() => {
    setForm(editingPlan ? { name: editingPlan.name, group_id: editingPlan.group_id, description: editingPlan.description, price: editingPlan.price, original_price: editingPlan.original_price || 0, validity_days: editingPlan.validity_days, validity_unit: editingPlan.validity_unit || 'days', sort_order: editingPlan.sort_order || 0, for_sale: editingPlan.for_sale, features: (editingPlan.features || []).join('\n') } : { name: '', group_id: null, description: '', price: 0, original_price: 0, validity_days: 30, validity_unit: 'days', sort_order: 0, for_sale: true, features: '' })
  }, [props.showPlanDialog, editingPlan])

  const update = <K extends keyof PaymentPlanForm>(key: K, value: PaymentPlanForm[K]) => setForm((current) => ({ ...current, [key]: value }))
  const selectedGroup = form.group_id ? groupFor(props.groups, form.group_id) : undefined
  const subscriptionGroups = props.groups.filter((group) => group.subscription_type === 'subscription')
  return h(Modal, { open: props.showPlanDialog, title: editingPlan ? copy.editPlan : copy.createPlan, onClose: props.onCloseEdit, wide: true, footer: [
    h(Button, { key: 'cancel', className: 'border border-[hsl(var(--border))] text-[hsl(var(--foreground))] hover:bg-[hsl(var(--muted))]', onClick: props.onCloseEdit }, copy.cancel),
    h(Button, { key: 'save', type: 'submit', disabled: props.saving, className: 'bg-[hsl(var(--foreground))] text-[hsl(var(--background))] hover:opacity-85', onClick: () => props.onSavePlan(form) }, props.saving ? copy.saving : copy.save),
  ] }, h('form', { className: 'space-y-4', onSubmit: (event: { preventDefault: () => void }) => { event.preventDefault(); props.onSavePlan(form) } }, [
    h('div', { key: 'first-row', className: 'grid grid-cols-1 gap-4 sm:grid-cols-2' }, [
      h('label', { key: 'name', className: 'block' }, [h('span', { key: 'label', className: 'input-label' }, copy.planName), h('input', { key: 'input', value: form.name, onChange: (event: { target: { value: string } }) => update('name', event.target.value), className: 'input mt-1', required: true })]),
      h('label', { key: 'group', className: 'block' }, [h('span', { key: 'label', className: 'input-label' }, copy.group), h('select', { key: 'input', value: form.group_id ?? '', onChange: (event: { target: { value: string } }) => update('group_id', event.target.value ? Number(event.target.value) : null), className: 'input mt-1 h-10', required: true }, [h('option', { key: 'empty', value: '' }, copy.selectGroup), ...subscriptionGroups.map((group) => h('option', { key: group.id, value: group.id }, `${group.name} (${group.rate_multiplier}x)`))])]),
    ]),
    selectedGroup ? h('div', { key: 'group-info', className: 'rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--muted)/.45)] p-3' }, [h('p', { key: 'name', className: 'font-medium text-[hsl(var(--foreground))]' }, selectedGroup.name), h('div', { key: 'limits', className: 'mt-2 grid grid-cols-1 gap-2 text-xs text-[hsl(var(--muted-foreground))] sm:grid-cols-3' }, [h('span', { key: 'daily' }, `${copy.dailyLimit}: ${selectedGroup.daily_limit_usd == null ? copy.unlimited : `$${selectedGroup.daily_limit_usd}`}`), h('span', { key: 'weekly' }, `${copy.weeklyLimit}: ${selectedGroup.weekly_limit_usd == null ? copy.unlimited : `$${selectedGroup.weekly_limit_usd}`}`), h('span', { key: 'monthly' }, `${copy.monthlyLimit}: ${selectedGroup.monthly_limit_usd == null ? copy.unlimited : `$${selectedGroup.monthly_limit_usd}`}`)])]) : null,
    h('label', { key: 'description', className: 'block' }, [h('span', { key: 'label', className: 'input-label' }, copy.planDescription), h('textarea', { key: 'input', rows: 2, value: form.description, onChange: (event: { target: { value: string } }) => update('description', event.target.value), className: 'input mt-1 resize-y', required: true })]),
    h('div', { key: 'price', className: 'grid grid-cols-1 gap-4 sm:grid-cols-2' }, [h('label', { key: 'current', className: 'block' }, [h('span', { key: 'label', className: 'input-label' }, copy.price), h('input', { key: 'input', type: 'number', min: '0.01', step: '0.01', value: form.price, onChange: (event: { target: { value: string } }) => update('price', Number(event.target.value)), className: 'input mt-1', required: true })]), h('label', { key: 'original', className: 'block' }, [h('span', { key: 'label', className: 'input-label' }, copy.originalPrice), h('input', { key: 'input', type: 'number', min: '0', step: '0.01', value: form.original_price, onChange: (event: { target: { value: string } }) => update('original_price', Number(event.target.value)), className: 'input mt-1' })])]),
    h('div', { key: 'validity', className: 'grid grid-cols-1 gap-4 sm:grid-cols-2' }, [h('label', { key: 'days', className: 'block' }, [h('span', { key: 'label', className: 'input-label' }, copy.validityDays), h('input', { key: 'input', type: 'number', min: '1', value: form.validity_days, onChange: (event: { target: { value: string } }) => update('validity_days', Number(event.target.value)), className: 'input mt-1', required: true })]), h('label', { key: 'unit', className: 'block' }, [h('span', { key: 'label', className: 'input-label' }, copy.validityUnit), h('select', { key: 'input', value: form.validity_unit, onChange: (event: { target: { value: string } }) => update('validity_unit', event.target.value), className: 'input mt-1 h-10' }, [h('option', { key: 'days', value: 'days' }, copy.days), h('option', { key: 'weeks', value: 'weeks' }, copy.weeks), h('option', { key: 'months', value: 'months' }, copy.months)])])]),
    h('label', { key: 'sort', className: 'block sm:max-w-[calc(50%-0.5rem)]' }, [h('span', { key: 'label', className: 'input-label' }, copy.sortOrder), h('input', { key: 'input', type: 'number', min: '0', value: form.sort_order, onChange: (event: { target: { value: string } }) => update('sort_order', Number(event.target.value)), className: 'input mt-1' })]),
    h('label', { key: 'features', className: 'block' }, [h('span', { key: 'label', className: 'input-label' }, copy.features), h('textarea', { key: 'input', rows: 3, value: form.features, onChange: (event: { target: { value: string } }) => update('features', event.target.value), placeholder: copy.featuresPlaceholder, className: 'input mt-1 resize-y' }), h('span', { key: 'hint', className: 'mt-1 block text-xs text-[hsl(var(--muted-foreground))]' }, copy.featuresHint)]),
    h('label', { key: 'sale', className: 'flex items-center gap-3 text-sm text-[hsl(var(--foreground))]' }, [h('span', { key: 'label' }, copy.forSale), h(ShadcnSwitch, { key: 'toggle', checked: form.for_sale, onCheckedChange: (value) => update('for_sale', value) })]),
  ]))
}

export default function PaymentPlansPage(props: PaymentPlansPageProps) {
  return h('div', { className: 'space-y-4' }, [
    h('div', { key: 'actions', className: 'flex items-center justify-end gap-2' }, [h(Button, { key: 'refresh', className: 'btn btn-secondary', disabled: props.loading, title: props.copy.refresh, onClick: props.onRefresh }, h(Icon, { name: 'refresh' })), h(Button, { key: 'create', className: 'bg-[hsl(var(--foreground))] text-[hsl(var(--background))] hover:opacity-85', onClick: () => props.onOpenEdit(null) }, props.copy.createPlan)]),
    h('section', { key: 'table', className: 'card overflow-hidden' }, h('div', { className: 'overflow-x-auto' }, h('table', { className: 'min-w-[900px] w-full text-left' }, [
      h('thead', { key: 'head', className: 'border-b border-[hsl(var(--border))] bg-[hsl(var(--muted)/.45)]' }, h('tr', null, [props.copy.planName, props.copy.group, props.copy.price, props.copy.validityDays, props.copy.forSale, props.copy.sortOrder, props.copy.actions].map((label) => h('th', { key: label, scope: 'col', className: 'px-4 py-3 text-xs font-semibold text-[hsl(var(--muted-foreground))]' }, label)))),
      h('tbody', { key: 'body', className: 'divide-y divide-[hsl(var(--border))]' }, props.loading ? h('tr', null, h('td', { colSpan: 7, className: 'px-4 py-12 text-center' }, h('span', { className: 'mx-auto block h-7 w-7 animate-spin rounded-full border-2 border-[hsl(var(--border))] border-t-[hsl(var(--foreground))]' }))) : props.plans.length ? props.plans.map((plan) => { const group = groupFor(props.groups, plan.group_id); return h('tr', { key: plan.id, className: 'transition-colors hover:bg-[hsl(var(--muted)/.35)]' }, [h('td', { key: 'name', className: 'px-4 py-3 text-sm font-medium text-[hsl(var(--foreground))]' }, [plan.name, h('div', { key: 'id', className: 'text-xs font-normal text-[hsl(var(--muted-foreground))]' }, `#${plan.id}`)]), h('td', { key: 'group', className: 'px-4 py-3 text-sm' }, group ? h('span', { className: 'text-[hsl(var(--foreground))]' }, group.name) : h('span', { className: 'text-[hsl(var(--muted-foreground))]' }, `#${plan.group_id} · ${props.copy.groupMissing}`)), h('td', { key: 'price', className: 'px-4 py-3 text-sm text-[hsl(var(--foreground))]' }, [h('span', { key: 'current', className: 'font-medium' }, `$${(plan.price || 0).toFixed(2)}`), plan.original_price ? h('span', { key: 'original', className: 'ml-1 text-xs text-[hsl(var(--muted-foreground))] line-through' }, `$${plan.original_price.toFixed(2)}`) : null]), h('td', { key: 'validity', className: 'px-4 py-3 text-sm text-[hsl(var(--foreground))]' }, `${plan.validity_days} ${plan.validity_unit === 'weeks' ? props.copy.weeks : plan.validity_unit === 'months' ? props.copy.months : props.copy.days}`), h('td', { key: 'sale', className: 'px-4 py-3' }, h(ShadcnSwitch, { checked: plan.for_sale, onCheckedChange: () => props.onToggleForSale(plan), 'aria-label': props.copy.forSale })), h('td', { key: 'sort', className: 'px-4 py-3 text-sm text-[hsl(var(--muted-foreground))]' }, plan.sort_order), h('td', { key: 'actions', className: 'px-4 py-3' }, h('div', { className: 'flex items-center gap-1' }, [h(Button, { key: 'edit', className: 'text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))]', title: props.copy.edit, onClick: () => props.onOpenEdit(plan) }, [h(Icon, { key: 'icon', name: 'edit' }), props.copy.edit]), h(Button, { key: 'delete', className: 'text-[hsl(var(--destructive))] hover:bg-[hsl(var(--destructive)/.1)]', title: props.copy.delete, onClick: () => props.onOpenDelete(plan) }, [h(Icon, { key: 'icon', name: 'trash' }), props.copy.delete])]))]) }) : h('tr', null, h('td', { colSpan: 7, className: 'px-4 py-12 text-center text-sm text-[hsl(var(--muted-foreground))]' }, props.copy.noPlans))),
    ]))),
    h(PlanFormDialog, { key: 'edit', props }),
    h(Modal, { key: 'delete', open: props.showDeleteDialog, title: props.copy.deletePlan, onClose: props.onCloseDelete, footer: [h(Button, { key: 'cancel', className: 'border border-[hsl(var(--border))] text-[hsl(var(--foreground))] hover:bg-[hsl(var(--muted))]', onClick: props.onCloseDelete }, props.copy.cancel), h(Button, { key: 'confirm', className: 'bg-[hsl(var(--destructive))] text-[hsl(var(--destructive-foreground))] hover:opacity-85', onClick: props.onDelete }, props.copy.confirm)] }, h('p', { className: 'text-sm text-[hsl(var(--muted-foreground))]' }, `${props.copy.deletePlanConfirm} #${props.deletingPlanId ?? ''}`)),
  ])
}
