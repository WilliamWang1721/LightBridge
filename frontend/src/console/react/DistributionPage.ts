import { useEffect, useRef, type ChangeEvent, type ReactNode } from 'react'
import type { AdminUser } from '@/types'
import type {
  CreateDistributionRequest,
  DistributionItem,
  DistributionKind,
  DistributionUserFilters,
} from '@/api/distributions'
import { createShadcnElement as h } from './ui/createElement'
import { Input as ShadcnInput, Select as ShadcnSelect } from './ui'

type AudienceMode = 'explicit' | 'advanced' | 'lines' | 'all'
type FilterKey = 'status' | 'role' | 'search' | 'group_name' | 'activity'
type PickerFilterKey = 'status' | 'role' | 'activity' | 'group_name'

export interface DistributionPageCopy {
  title: string
  kind: string
  kinds: Record<DistributionKind, string>
  price: string
  priceHint: string
  content: string
  contentPlaceholder: string
  personalizedContent: string
  personalizedContentPlaceholder: string
  personalizedContentHint: string
  fileHint: string
  audienceMode: string
  audienceExplicit: string
  audienceAdvanced: string
  audienceLines: string
  audienceAll: string
  recipients: string
  recipientsPickerHint: string
  openUserPicker: string
  removeSelectedUser: string
  noUsersSelected: string
  enablePersonalizedContent: string
  filterSearch: string
  filterGroup: string
  filterAnyStatus: string
  filterAnyRole: string
  filterAnyActivity: string
  filterHasActivity: string
  filterHasUsage: string
  filterHasBalanceChange: string
  filterNoActivity: string
  multilineImport: string
  multilinePlaceholder: string
  loading: string
  previewAudience: string
  audienceCount: (count: number) => string
  send: string
  batchJson: string
  batchJsonHint: string
  sendBatch: string
  history: string
  deliveryStats: (item: DistributionItem) => string
  download: string
  delete: string
  empty: string
  userPickerTitle: string
  userPickerSearch: string
  userPickerFilter: string
  selectedUsers: (count: number) => string
  selectPageUsers: string
  clearPageSelection: string
  userPickerEmpty: string
  previous: string
  next: string
  clearSelectedUsers: string
  confirm: string
  close: string
}

export interface DistributionPageActions {
  onTitleChange: (value: string) => void
  onKindChange: (value: DistributionKind) => void
  onPriceChange: (value: number) => void
  onContentChange: (value: string) => void
  onPersonalizedContentChange: (value: string) => void
  onFileChange: (file?: File) => void
  onAudienceModeChange: (value: AudienceMode) => void
  onFilterChange: (key: FilterKey, value: string) => void
  onImportLinesChange: (value: string) => void
  onPersonalizedModeChange: (value: boolean) => void
  onOpenUserPicker: () => void
  onToggleSelectedUser: (user: AdminUser) => void
  onPreviewAudience: () => void
  onSubmit: () => void
  onBatchJsonChange: (value: string) => void
  onSubmitBatch: () => void
  onRemove: (item: DistributionItem) => void
  onDownload: (item: DistributionItem) => void
  onPickerSearchChange: (value: string) => void
  onPickerFilterChange: (key: PickerFilterKey, value: string) => void
  onLoadPickerUsers: (resetPage?: boolean) => void
  onTogglePickerPageSelection: () => void
  onChangePickerPage: (page: number) => void
  onClearSelectedUsers: () => void
  onCloseUserPicker: () => void
}

export interface DistributionPageProps {
  form: CreateDistributionRequest
  filters: DistributionUserFilters
  audienceMode: AudienceMode
  importLines: string
  selectedUsers: AdminUser[]
  selectedUserIds: readonly number[]
  personalizedMode: boolean
  personalizedContent: string
  selectedFileName: string
  audienceCount: number | null
  previewing: boolean
  submitting: boolean
  loading: boolean
  items: DistributionItem[]
  showUserPicker: boolean
  pickerUsers: AdminUser[]
  pickerLoading: boolean
  pickerSearch: string
  pickerPage: number
  pickerPages: number
  pickerFilters: {
    status: '' | 'active' | 'disabled'
    role: '' | 'admin' | 'user'
    activity: '' | 'any' | 'usage' | 'balance_change' | 'none'
    group_name: string
  }
  batchJson: string
  canPersonalize: boolean
  personalizationError: string
  allPickerUsersSelected: boolean
  copy: DistributionPageCopy
  actions: DistributionPageActions
}

function Select({
  value,
  onChange,
  options,
  className = 'input distribution-select',
}: {
  value: string
  onChange: (value: string) => void
  options: Array<{ value: string; label: string }>
  className?: string
}) {
  return h(ShadcnSelect, {
    className,
    value,
    onChange: (event: ChangeEvent<HTMLSelectElement>) => onChange(event.target.value),
  }, options.map((option) => h('option', { key: option.value, value: option.value }, option.label)))
}

function TextInput({
  value,
  onChange,
  placeholder,
  className = 'input',
  type = 'text',
}: {
  value: string | number
  onChange: (value: string) => void
  placeholder?: string
  className?: string
  type?: string
}) {
  return h(ShadcnInput, {
    className,
    type,
    value,
    placeholder,
    onChange: (event: ChangeEvent<HTMLInputElement>) => onChange(event.target.value),
  })
}

function Field({ label, children, hint }: { label: string; children: ReactNode; hint?: string }) {
  return h('label', { className: 'space-y-1' }, [
    h('span', { key: 'label', className: 'text-sm font-medium text-gray-700 dark:text-gray-200' }, label),
    h('span', { key: 'control', className: 'block' }, children),
    hint ? h('p', { key: 'hint', className: 'text-xs text-gray-500 dark:text-gray-400' }, hint) : null,
  ])
}

function FilterSelect({
  value,
  onChange,
  copy,
  kind,
}: {
  value: string
  onChange: (value: string) => void
  copy: DistributionPageCopy
  kind: 'status' | 'role' | 'activity'
}) {
  const options = kind === 'status'
    ? [
        { value: '', label: copy.filterAnyStatus },
        { value: 'active', label: 'active' },
        { value: 'disabled', label: 'disabled' },
      ]
    : kind === 'role'
      ? [
          { value: '', label: copy.filterAnyRole },
          { value: 'user', label: 'user' },
          { value: 'admin', label: 'admin' },
        ]
      : [
          { value: '', label: copy.filterAnyActivity },
          { value: 'any', label: copy.filterHasActivity },
          { value: 'usage', label: copy.filterHasUsage },
          { value: 'balance_change', label: copy.filterHasBalanceChange },
          { value: 'none', label: copy.filterNoActivity },
        ]
  return Select({ value, onChange, options })
}

function UserPickerDialog({ props }: { props: DistributionPageProps }) {
  const { copy, actions } = props
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!props.showUserPicker) return

    const previousActiveElement = document.activeElement as HTMLElement | null
    const wasModalLocked = document.body.classList.contains('modal-open')
    document.body.classList.add('modal-open')

    const focusable = () => Array.from(
      panelRef.current?.querySelectorAll<HTMLElement>('button, input, select, textarea, [tabindex]:not([tabindex="-1"])') ?? [],
    ).filter((element) => !element.hasAttribute('disabled'))

    const handleKeydown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        actions.onCloseUserPicker()
        return
      }
      if (event.key !== 'Tab') return
      const elements = focusable()
      if (elements.length === 0) {
        event.preventDefault()
        panelRef.current?.focus()
        return
      }
      const first = elements[0]
      const last = elements[elements.length - 1]
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault()
        last.focus()
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first.focus()
      }
    }

    document.addEventListener('keydown', handleKeydown)
    window.setTimeout(() => (focusable()[0] || panelRef.current)?.focus(), 0)
    return () => {
      document.removeEventListener('keydown', handleKeydown)
      if (!wasModalLocked) document.body.classList.remove('modal-open')
      if (previousActiveElement?.isConnected) previousActiveElement.focus()
    }
  }, [props.showUserPicker])

  if (!props.showUserPicker) return null

  return h('div', {
    className: 'fixed inset-0 z-[var(--ui-z-dialog)] flex min-h-[100dvh] items-center justify-center overflow-y-auto bg-[hsl(var(--foreground)/0.55)] p-3 backdrop-blur-[4px] sm:p-6',
    role: 'dialog',
    'aria-modal': 'true',
    'aria-labelledby': 'distribution-user-picker-title',
    onClick: actions.onCloseUserPicker,
  }, h('div', {
    ref: panelRef,
    tabIndex: -1,
    className: 'relative flex max-h-[calc(100dvh-1.5rem)] w-full max-w-6xl flex-col overflow-hidden rounded-[calc(var(--ui-radius)+0.375rem)] border border-[hsl(var(--border))] bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))] shadow-[0_24px_80px_hsl(var(--foreground)/0.3)] outline-none sm:max-h-[calc(100dvh-3rem)]',
    onClick: (event: { stopPropagation: () => void }) => event.stopPropagation(),
  }, [
    h('div', { key: 'header', className: 'flex items-center justify-between border-b border-[hsl(var(--border))] px-5 py-4' }, [
      h('h2', { key: 'title', id: 'distribution-user-picker-title', className: 'text-base font-semibold text-[hsl(var(--foreground))]' }, copy.userPickerTitle),
      h('button', {
        key: 'close',
        type: 'button',
        className: 'rounded-[var(--ui-radius)] p-2 text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--accent))] hover:text-[hsl(var(--accent-foreground))]',
        'aria-label': copy.close,
        onClick: actions.onCloseUserPicker,
      }, '×'),
    ]),
    h('div', { key: 'body', className: 'min-h-0 overflow-y-auto p-4 sm:p-5' }, [
      h('div', { key: 'filters', className: 'grid gap-3 md:grid-cols-[minmax(0,1fr)_9rem_9rem_12rem_auto]' }, [
        h(TextInput, {
          key: 'search',
          value: props.pickerSearch,
          placeholder: copy.userPickerSearch,
          onChange: actions.onPickerSearchChange,
        }),
        h(FilterSelect, {
          key: 'status',
          value: props.pickerFilters.status,
          copy,
          kind: 'status',
          onChange: (value: string) => actions.onPickerFilterChange('status', value),
        }),
        h(FilterSelect, {
          key: 'role',
          value: props.pickerFilters.role,
          copy,
          kind: 'role',
          onChange: (value: string) => actions.onPickerFilterChange('role', value),
        }),
        h(FilterSelect, {
          key: 'activity',
          value: props.pickerFilters.activity,
          copy,
          kind: 'activity',
          onChange: (value: string) => actions.onPickerFilterChange('activity', value),
        }),
        h(TextInput, {
          key: 'group',
          value: props.pickerFilters.group_name,
          placeholder: copy.filterGroup,
          onChange: (value: string) => actions.onPickerFilterChange('group_name', value),
        }),
        h('button', {
          key: 'apply',
          type: 'button',
          className: 'btn btn-secondary',
          disabled: props.pickerLoading,
          onClick: () => actions.onLoadPickerUsers(true),
        }, props.pickerLoading ? copy.loading : copy.userPickerFilter),
      ]),
      h('div', { key: 'selection-summary', className: 'mt-4 flex flex-wrap items-center justify-between gap-3 text-sm' }, [
        h('span', { key: 'count', className: 'text-gray-500 dark:text-gray-400' }, copy.selectedUsers(props.selectedUsers.length)),
        h('button', {
          key: 'page-toggle',
          type: 'button',
          className: 'text-sm font-medium text-[hsl(var(--primary))] hover:text-[hsl(var(--accent-foreground))] disabled:opacity-50',
          disabled: props.pickerUsers.length === 0,
          onClick: actions.onTogglePickerPageSelection,
        }, props.allPickerUsersSelected ? copy.clearPageSelection : copy.selectPageUsers),
      ]),
      h('div', { key: 'users', className: 'mt-3 overflow-hidden rounded-[calc(var(--ui-radius)+0.125rem)] border border-[hsl(var(--border))]' },
        props.pickerLoading
          ? h('div', { className: 'p-8 text-center text-sm text-gray-500' }, copy.loading)
          : props.pickerUsers.length === 0
            ? h('div', { className: 'p-8 text-center text-sm text-gray-500' }, copy.userPickerEmpty)
            : h('div', { className: 'divide-y divide-gray-100 dark:divide-dark-700' }, props.pickerUsers.map((user) => h('label', {
                key: user.id,
                className: 'flex cursor-pointer items-center gap-3 px-4 py-3 transition-colors hover:bg-gray-50 dark:hover:bg-dark-800',
              }, [
                h('input', {
                  key: 'checkbox',
                  type: 'checkbox',
                  className: 'h-4 w-4 rounded accent-[hsl(var(--primary))]',
                  checked: props.selectedUserIds.includes(user.id),
                  onChange: () => actions.onToggleSelectedUser(user),
                }),
                h('span', { key: 'body', className: 'min-w-0 flex-1' }, [
                  h('span', { key: 'name', className: 'block truncate text-sm font-medium text-gray-800 dark:text-gray-100' }, user.username || user.email),
                  h('span', { key: 'email', className: 'block truncate text-xs text-gray-500 dark:text-gray-400' }, `#${user.id} · ${user.email}`),
                ]),
                h('span', { key: 'status', className: 'shrink-0 text-xs text-gray-400' }, user.status),
              ]))),
      ),
      props.pickerPages > 1 ? h('div', { key: 'pagination', className: 'mt-4 flex items-center justify-center gap-3' }, [
        h('button', {
          key: 'previous',
          type: 'button',
          className: 'btn btn-secondary',
          disabled: props.pickerPage <= 1 || props.pickerLoading,
          onClick: () => actions.onChangePickerPage(props.pickerPage - 1),
        }, copy.previous),
        h('span', { key: 'page', className: 'text-sm text-gray-500' }, `${props.pickerPage} / ${props.pickerPages}`),
        h('button', {
          key: 'next',
          type: 'button',
          className: 'btn btn-secondary',
          disabled: props.pickerPage >= props.pickerPages || props.pickerLoading,
          onClick: () => actions.onChangePickerPage(props.pickerPage + 1),
        }, copy.next),
      ]) : null,
    ]),
    h('div', { key: 'footer', className: 'flex items-center justify-between gap-3 border-t border-[hsl(var(--border))] bg-[hsl(var(--muted))] px-5 py-4' }, [
      h('button', {
        key: 'clear',
        type: 'button',
        className: 'text-sm text-gray-500 hover:text-red-600',
        onClick: actions.onClearSelectedUsers,
      }, copy.clearSelectedUsers),
      h('button', {
        key: 'confirm',
        type: 'button',
        className: 'btn btn-primary',
        onClick: actions.onCloseUserPicker,
      }, copy.confirm),
    ]),
  ]))
}

function renderAudienceFilters(props: DistributionPageProps) {
  const { copy, actions } = props
  return h('div', { key: 'filters', className: 'mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-5' }, [
    h(TextInput, {
      key: 'search',
      value: props.filters.search || '',
      placeholder: copy.filterSearch,
      onChange: (value: string) => actions.onFilterChange('search', value),
    }),
    h(TextInput, {
      key: 'group',
      value: props.filters.group_name || '',
      placeholder: copy.filterGroup,
      onChange: (value: string) => actions.onFilterChange('group_name', value),
    }),
    h(FilterSelect, {
      key: 'status',
      value: props.filters.status || '',
      copy,
      kind: 'status',
      onChange: (value: string) => actions.onFilterChange('status', value),
    }),
    h(FilterSelect, {
      key: 'role',
      value: props.filters.role || '',
      copy,
      kind: 'role',
      onChange: (value: string) => actions.onFilterChange('role', value),
    }),
    h(FilterSelect, {
      key: 'activity',
      value: props.filters.activity || '',
      copy,
      kind: 'activity',
      onChange: (value: string) => actions.onFilterChange('activity', value),
    }),
  ])
}

export default function DistributionPage(props: DistributionPageProps) {
  const { copy, actions } = props
  const selectedUsers = props.selectedUsers.length > 0
    ? h('div', { key: 'selected-users', className: 'grid max-h-40 grid-cols-[repeat(auto-fit,minmax(min(100%,15rem),1fr))] gap-2 overflow-y-auto' }, props.selectedUsers.map((user) => h('div', {
        key: user.id,
        className: 'flex min-w-0 items-center justify-between gap-3 rounded-[var(--ui-radius)] border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-600 dark:bg-dark-900',
      }, [
        h('div', { key: 'body', className: 'min-w-0' }, [
          h('div', { key: 'name', className: 'truncate text-sm font-medium text-gray-800 dark:text-gray-100' }, user.username || user.email),
          h('div', { key: 'email', className: 'truncate text-xs text-gray-500 dark:text-gray-400' }, `#${user.id} · ${user.email}`),
        ]),
        h('button', {
          key: 'remove',
          type: 'button',
          className: 'shrink-0 rounded-md px-2 py-1 text-xs text-gray-500 hover:bg-white hover:text-red-600 dark:hover:bg-dark-800',
          'aria-label': copy.removeSelectedUser,
          onClick: () => actions.onToggleSelectedUser(user),
        }, '×'),
      ])))
    : h('div', { key: 'selected-users', className: 'rounded-[var(--ui-radius)] border border-dashed border-gray-300 px-4 py-4 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400' }, copy.noUsersSelected)

  return h('div', {}, [
    h('div', { key: 'content', className: 'space-y-6' }, [
    h('section', { key: 'form', className: 'card p-5' }, [
      h('div', { key: 'top-fields', className: 'grid gap-4 lg:grid-cols-2' }, [
        h(Field, {
          key: 'title',
          label: copy.title,
          children: h(TextInput, { value: props.form.title, onChange: actions.onTitleChange }),
        }),
        h(Field, {
          key: 'kind',
          label: copy.kind,
          children: Select({
            value: props.form.kind,
            onChange: (value: string) => actions.onKindChange(value as DistributionKind),
            options: (Object.keys(copy.kinds) as DistributionKind[]).map((value) => ({ value, label: copy.kinds[value] })),
          }),
        }),
        props.form.kind === 'paid' ? h(Field, {
          key: 'price',
          label: copy.price,
          hint: copy.priceHint,
          children: h(TextInput, {
            value: props.form.price ?? 0,
            type: 'number',
            onChange: (value: string) => actions.onPriceChange(Number(value) || 0),
          }),
        }) : null,
      ]),
      h('label', { key: 'content', className: 'mt-4 block space-y-1' }, [
        h('span', { key: 'label', className: 'text-sm font-medium text-gray-700 dark:text-gray-200' }, props.personalizedMode ? copy.personalizedContent : copy.content),
        props.personalizedMode
          ? h('textarea', {
              key: 'personalized',
              className: 'input min-h-36 font-mono text-sm',
              value: props.personalizedContent,
              placeholder: copy.personalizedContentPlaceholder,
              onChange: (event: ChangeEvent<HTMLTextAreaElement>) => actions.onPersonalizedContentChange(event.target.value),
            })
          : h('textarea', {
              key: 'content-input',
              className: 'input min-h-28',
              value: props.form.content || '',
              placeholder: copy.contentPlaceholder,
              onChange: (event: ChangeEvent<HTMLTextAreaElement>) => actions.onContentChange(event.target.value),
            }),
        props.personalizedMode ? h('p', { key: 'hint', className: 'text-xs text-gray-500 dark:text-gray-400' }, copy.personalizedContentHint) : null,
        props.personalizationError ? h('p', { key: 'error', className: 'text-xs text-red-600 dark:text-red-400' }, props.personalizationError) : null,
      ]),
      props.form.kind === 'file' || props.form.kind === 'account_export' ? h('div', { key: 'file', className: 'mt-4 rounded-lg border border-dashed border-gray-300 p-4 dark:border-dark-600' }, [
        h('input', {
          key: 'file-input',
          type: 'file',
          accept: props.form.kind === 'account_export' ? '.json,.zip,application/json,application/zip' : undefined,
          onChange: (event: ChangeEvent<HTMLInputElement>) => actions.onFileChange(event.currentTarget.files?.[0]),
        }),
        h('p', { key: 'hint', className: 'mt-2 text-xs text-gray-500' }, copy.fileHint),
        props.selectedFileName ? h('p', { key: 'name', className: 'mt-1 text-sm text-gray-700 dark:text-gray-200' }, props.selectedFileName) : null,
      ]) : null,
      h('div', { key: 'audience', className: 'mt-5 border-t border-gray-100 pt-5 dark:border-dark-700' }, [
        h('div', { key: 'mode-grid', className: 'grid gap-4 lg:grid-cols-2' }, h(Field, {
          label: copy.audienceMode,
          children: Select({
            value: props.audienceMode,
            onChange: (value: string) => actions.onAudienceModeChange(value as AudienceMode),
            options: [
              { value: 'explicit', label: copy.audienceExplicit },
              { value: 'advanced', label: copy.audienceAdvanced },
              { value: 'lines', label: copy.audienceLines },
              { value: 'all', label: copy.audienceAll },
            ],
          }),
        })),
        props.audienceMode === 'explicit' ? h('div', { key: 'explicit', className: 'mt-4 space-y-3' }, [
          h('div', { key: 'toolbar', className: 'flex flex-wrap items-center justify-between gap-3' }, [
            h('div', { key: 'copy' }, [
              h('span', { key: 'label', className: 'text-sm font-medium text-gray-700 dark:text-gray-200' }, copy.recipients),
              h('p', { key: 'hint', className: 'mt-1 text-xs text-gray-500 dark:text-gray-400' }, copy.recipientsPickerHint),
            ]),
            h('button', { key: 'open', type: 'button', className: 'btn btn-secondary', onClick: actions.onOpenUserPicker }, copy.openUserPicker),
          ]),
          selectedUsers,
          props.canPersonalize ? h('label', { key: 'personalized-toggle', className: 'inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200' }, [
            h('input', {
              key: 'checkbox',
              type: 'checkbox',
              className: 'h-4 w-4 rounded accent-[hsl(var(--primary))]',
              checked: props.personalizedMode,
              onChange: (event: ChangeEvent<HTMLInputElement>) => actions.onPersonalizedModeChange(event.target.checked),
            }),
            h('span', { key: 'label' }, copy.enablePersonalizedContent),
          ]) : null,
        ]) : null,
        props.audienceMode === 'advanced' ? renderAudienceFilters(props) : null,
        props.audienceMode === 'lines' ? h('label', { key: 'lines', className: 'mt-4 block space-y-1' }, [
          h('span', { key: 'label', className: 'text-sm font-medium text-gray-700 dark:text-gray-200' }, copy.multilineImport),
          h('textarea', {
            key: 'textarea',
            className: 'input min-h-36 font-mono text-sm',
            value: props.importLines,
            placeholder: copy.multilinePlaceholder,
            onChange: (event: ChangeEvent<HTMLTextAreaElement>) => actions.onImportLinesChange(event.target.value),
          }),
        ]) : null,
        h('div', { key: 'preview', className: 'mt-4 flex flex-wrap items-center gap-3' }, [
          h('button', {
            key: 'button',
            type: 'button',
            className: 'btn btn-secondary',
            disabled: props.previewing || Boolean(props.personalizationError),
            onClick: actions.onPreviewAudience,
          }, props.previewing ? copy.loading : copy.previewAudience),
          props.audienceCount !== null ? h('span', { key: 'count', className: 'text-sm text-gray-600 dark:text-gray-300' }, copy.audienceCount(props.audienceCount)) : null,
        ]),
      ]),
      h('div', { key: 'submit', className: 'mt-5 flex justify-end' }, h('button', {
        type: 'button',
        className: 'btn btn-primary btn-lg',
        disabled: props.submitting || Boolean(props.personalizationError),
        onClick: actions.onSubmit,
      }, props.submitting ? copy.loading : copy.send)),
    ]),
    h('details', { key: 'batch', className: 'card p-5' }, [
      h('summary', { key: 'summary', className: 'cursor-pointer font-medium text-gray-800 dark:text-gray-100' }, copy.batchJson),
      h('p', { key: 'hint', className: 'mt-2 text-sm text-gray-500' }, copy.batchJsonHint),
      h('textarea', {
        key: 'json',
        className: 'input mt-3 min-h-40 font-mono text-sm',
        value: props.batchJson,
        onChange: (event: ChangeEvent<HTMLTextAreaElement>) => actions.onBatchJsonChange(event.target.value),
      }),
      h('button', { key: 'send', type: 'button', className: 'btn btn-secondary mt-3', onClick: actions.onSubmitBatch }, copy.sendBatch),
    ]),
    h('section', { key: 'history', className: 'card overflow-hidden' }, [
      h('div', { key: 'header', className: 'border-b border-gray-100 px-5 py-4 font-medium text-gray-800 dark:border-dark-700 dark:text-gray-100' }, copy.history),
      props.loading
        ? h('div', { key: 'loading', className: 'p-8 text-center text-gray-500' }, copy.loading)
        : h('div', { key: 'items', className: 'divide-y divide-gray-100 dark:divide-dark-700' }, [
            ...props.items.map((item) => h('div', { key: item.id, className: 'flex flex-col gap-3 p-5 lg:flex-row lg:items-center lg:justify-between' }, [
              h('div', { key: 'copy' }, [
                h('div', { key: 'title', className: 'flex items-center gap-2' }, [
                  h('span', { key: 'name', className: 'font-medium text-gray-900 dark:text-white' }, item.title),
                  h('span', { key: 'kind', className: 'rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300' }, copy.kinds[item.kind]),
                  item.kind === 'paid' ? h('span', { key: 'price', className: 'rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300' }, `$${(item.price || 0).toFixed(2)}`) : null,
                ]),
                h('div', { key: 'stats', className: 'mt-1 text-sm text-gray-500' }, copy.deliveryStats(item)),
              ]),
              h('div', { key: 'actions', className: 'flex gap-2' }, [
                item.has_attachment ? h('button', { key: 'download', type: 'button', className: 'rounded-lg border px-3 py-2 text-sm dark:border-dark-600', onClick: () => actions.onDownload(item) }, copy.download) : null,
                h('button', { key: 'delete', type: 'button', className: 'rounded-lg border border-red-200 px-3 py-2 text-sm text-red-600 hover:bg-red-50 dark:border-red-900', onClick: () => actions.onRemove(item) }, copy.delete),
              ]),
            ])),
            props.items.length === 0 ? h('div', { key: 'empty', className: 'p-8 text-center text-gray-500' }, copy.empty) : null,
          ]),
    ]),
    ]),
    h(UserPickerDialog, { key: 'picker', props }),
  ])
}
