<template>
  <AppLayout>
    <ReactPageHost
      :load="loadDistributionPage"
      :props="pageProps"
      :error-message="t('common.error')"
    >
      <template #fallback>
        <div class="space-y-6">
    <section class="card p-5">
      <div class="grid gap-4 lg:grid-cols-2">
        <label class="space-y-1">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('distributions.title') }}</span>
          <input v-model="form.title" class="input" maxlength="200" />
        </label>
        <label class="space-y-1">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('distributions.kind') }}</span>
          <select v-model="form.kind" class="input distribution-select">
            <option value="text">{{ t('distributions.kinds.text') }}</option>
            <option value="message">{{ t('distributions.kinds.message') }}</option>
            <option value="file">{{ t('distributions.kinds.file') }}</option>
            <option value="account_export">{{ t('distributions.kinds.account_export') }}</option>
            <option value="paid">{{ t('distributions.kinds.paid') }}</option>
          </select>
        </label>
        <label v-if="form.kind === 'paid'" class="space-y-1">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('distributions.price') }}</span>
          <input v-model.number="form.price" class="input" type="number" min="0.01" step="0.01" inputmode="decimal" />
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('distributions.priceHint') }}</p>
        </label>
      </div>

      <label class="mt-4 block space-y-1">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-200">
          {{ personalizedMode ? t('distributions.personalizedContent') : t('distributions.content') }}
        </span>
        <textarea
          v-if="personalizedMode"
          v-model="personalizedContent"
          class="input min-h-36 font-mono text-sm"
          :placeholder="t('distributions.personalizedContentPlaceholder')"
        />
        <textarea
          v-else
          v-model="form.content"
          class="input min-h-28"
          :placeholder="t('distributions.contentPlaceholder')"
        />
        <p v-if="personalizedMode" class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('distributions.personalizedContentHint') }}
        </p>
        <p v-if="personalizationError" class="text-xs text-red-600 dark:text-red-400">
          {{ personalizationError }}
        </p>
      </label>

      <div v-if="form.kind === 'file' || form.kind === 'account_export'" class="mt-4 rounded-lg border border-dashed border-gray-300 p-4 dark:border-dark-600">
        <input type="file" :accept="form.kind === 'account_export' ? '.json,.zip,application/json,application/zip' : undefined" @change="selectFile" />
        <p class="mt-2 text-xs text-gray-500">{{ t('distributions.fileHint') }}</p>
        <p v-if="selectedFileName" class="mt-1 text-sm text-gray-700 dark:text-gray-200">{{ selectedFileName }}</p>
      </div>

      <div class="mt-5 border-t border-gray-100 pt-5 dark:border-dark-700">
        <div class="grid gap-4 lg:grid-cols-2">
          <label class="w-full max-w-xl space-y-1">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('distributions.audienceMode') }}</span>
            <select v-model="audienceMode" class="input distribution-select">
              <option value="explicit">{{ t('distributions.audienceExplicit') }}</option>
              <option value="advanced">{{ t('distributions.audienceAdvanced') }}</option>
              <option value="lines">{{ t('distributions.audienceLines') }}</option>
              <option value="all">{{ t('distributions.audienceAll') }}</option>
            </select>
          </label>
        </div>

        <div v-if="audienceMode === 'explicit'" class="mt-4 space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('distributions.recipients') }}</span>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('distributions.recipientsPickerHint') }}</p>
            </div>
            <button
              type="button"
              class="btn btn-secondary"
              @click="openUserPicker"
            >
              {{ t('distributions.openUserPicker') }}
            </button>
          </div>

          <div v-if="selectedUsers.length" class="distribution-selected-users">
            <div
              v-for="user in selectedUsers"
              :key="user.id"
              class="flex min-w-0 items-center justify-between gap-3 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-600 dark:bg-dark-900"
            >
              <div class="min-w-0">
                <div class="truncate text-sm font-medium text-gray-800 dark:text-gray-100">
                  {{ user.username || user.email }}
                </div>
                <div class="truncate text-xs text-gray-500 dark:text-gray-400">#{{ user.id }} · {{ user.email }}</div>
              </div>
              <button
                type="button"
                class="shrink-0 rounded-md px-2 py-1 text-xs text-gray-500 hover:bg-white hover:text-red-600 dark:hover:bg-dark-800"
                :aria-label="t('distributions.removeSelectedUser')"
                @click="toggleSelectedUser(user)"
              >
                ×
              </button>
            </div>
          </div>
          <div v-else class="rounded-lg border border-dashed border-gray-300 px-4 py-4 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400">
            {{ t('distributions.noUsersSelected') }}
          </div>

          <label v-if="canPersonalize" class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
            <input v-model="personalizedMode" type="checkbox" class="h-4 w-4 rounded accent-[hsl(var(--primary))]" />
            <span>{{ t('distributions.enablePersonalizedContent') }}</span>
          </label>
        </div>

        <div v-if="audienceMode === 'advanced'" class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-5">
          <input v-model="filters.search" class="input" :placeholder="t('distributions.filterSearch')" />
          <input v-model="filters.group_name" class="input" :placeholder="t('distributions.filterGroup')" />
          <select v-model="filters.status" class="input distribution-select">
            <option value="">{{ t('distributions.filterAnyStatus') }}</option>
            <option value="active">active</option>
            <option value="disabled">disabled</option>
          </select>
          <select v-model="filters.role" class="input distribution-select">
            <option value="">{{ t('distributions.filterAnyRole') }}</option>
            <option value="user">user</option>
            <option value="admin">admin</option>
          </select>
          <select v-model="filters.activity" class="input distribution-select">
            <option value="">{{ t('distributions.filterAnyActivity') }}</option>
            <option value="any">{{ t('distributions.filterHasActivity') }}</option>
            <option value="usage">{{ t('distributions.filterHasUsage') }}</option>
            <option value="balance_change">{{ t('distributions.filterHasBalanceChange') }}</option>
            <option value="none">{{ t('distributions.filterNoActivity') }}</option>
          </select>
        </div>

        <label v-if="audienceMode === 'lines'" class="mt-4 block space-y-1">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('distributions.multilineImport') }}</span>
          <textarea v-model="importLines" class="input min-h-36 font-mono text-sm" :placeholder="t('distributions.multilinePlaceholder')" />
        </label>

        <div class="mt-4 flex flex-wrap items-center gap-3">
          <button class="btn btn-secondary" :disabled="previewing || Boolean(personalizationError)" @click="previewAudience">
            {{ previewing ? t('common.loading') : t('distributions.previewAudience') }}
          </button>
          <span v-if="audienceCount !== null" class="text-sm text-gray-600 dark:text-gray-300">
            {{ t('distributions.audienceCount', { count: audienceCount }) }}
          </span>
        </div>
      </div>

      <div class="mt-5 flex justify-end">
        <button class="btn btn-primary btn-lg" :disabled="submitting || Boolean(personalizationError)" @click="submit">
          {{ submitting ? t('common.loading') : t('distributions.send') }}
        </button>
      </div>
    </section>

    <details class="card p-5">
      <summary class="cursor-pointer font-medium text-gray-800 dark:text-gray-100">{{ t('distributions.batchJson') }}</summary>
      <p class="mt-2 text-sm text-gray-500">{{ t('distributions.batchJsonHint') }}</p>
      <textarea v-model="batchJson" class="input mt-3 min-h-40 font-mono text-sm" />
      <button class="btn btn-secondary mt-3" @click="submitBatch">
        {{ t('distributions.sendBatch') }}
      </button>
    </details>

    <section class="card overflow-hidden">
      <div class="border-b border-gray-100 px-5 py-4 font-medium text-gray-800 dark:border-dark-700 dark:text-gray-100">{{ t('distributions.history') }}</div>
      <div v-if="loading" class="p-8 text-center text-gray-500">{{ t('common.loading') }}</div>
      <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
        <div v-for="item in items" :key="item.id" class="flex flex-col gap-3 p-5 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <div class="flex items-center gap-2">
              <span class="font-medium text-gray-900 dark:text-white">{{ item.title }}</span>
              <span class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">{{ t(`distributions.kinds.${item.kind}`) }}</span>
              <span v-if="item.kind === 'paid'" class="rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">{{ formatPrice(item.price) }}</span>
            </div>
            <div class="mt-1 text-sm text-gray-500">
              {{ t('distributions.deliveryStats', { recipients: item.recipient_count, read: item.read_count, downloads: item.download_count }) }}
            </div>
          </div>
          <div class="flex gap-2">
            <button v-if="item.has_attachment" class="rounded-lg border px-3 py-2 text-sm dark:border-dark-600" @click="downloadAdmin(item)">{{ t('distributions.download') }}</button>
            <button class="rounded-lg border border-red-200 px-3 py-2 text-sm text-red-600 hover:bg-red-50 dark:border-red-900" @click="remove(item)">{{ t('common.delete') }}</button>
          </div>
        </div>
        <div v-if="items.length === 0" class="p-8 text-center text-gray-500">{{ t('distributions.empty') }}</div>
      </div>
    </section>
        </div>

        <BaseDialog
        :show="showUserPicker"
        :title="t('distributions.userPickerTitle')"
        width="extra-wide"
        :close-on-click-outside="true"
        @close="showUserPicker = false"
      >
    <div class="space-y-4">
      <div class="grid gap-3 md:grid-cols-[minmax(0,1fr)_9rem_9rem_12rem_auto]">
        <input
          v-model="pickerSearch"
          class="input"
          :placeholder="t('distributions.userPickerSearch')"
          @keyup.enter="loadPickerUsers(true)"
        />
        <select v-model="pickerFilters.status" class="input distribution-select">
          <option value="">{{ t('distributions.filterAnyStatus') }}</option>
          <option value="active">active</option>
          <option value="disabled">disabled</option>
        </select>
        <select v-model="pickerFilters.role" class="input distribution-select">
          <option value="">{{ t('distributions.filterAnyRole') }}</option>
          <option value="user">user</option>
          <option value="admin">admin</option>
        </select>
        <select v-model="pickerFilters.activity" class="input distribution-select">
          <option value="">{{ t('distributions.filterAnyActivity') }}</option>
          <option value="any">{{ t('distributions.filterHasActivity') }}</option>
          <option value="usage">{{ t('distributions.filterHasUsage') }}</option>
          <option value="balance_change">{{ t('distributions.filterHasBalanceChange') }}</option>
          <option value="none">{{ t('distributions.filterNoActivity') }}</option>
        </select>
        <input v-model="pickerFilters.group_name" class="input" :placeholder="t('distributions.filterGroup')" @keyup.enter="loadPickerUsers(true)" />
        <button type="button" class="btn btn-secondary" :disabled="pickerLoading" @click="loadPickerUsers(true)">
          {{ pickerLoading ? t('common.loading') : t('distributions.userPickerFilter') }}
        </button>
      </div>

      <div class="flex flex-wrap items-center justify-between gap-3 text-sm">
        <span class="text-gray-500 dark:text-gray-400">{{ t('distributions.selectedUsers', { count: selectedUsers.length }) }}</span>
        <button
          type="button"
          class="text-sm font-medium text-[hsl(var(--primary))] hover:text-[hsl(var(--accent-foreground))]"
          :disabled="pickerUsers.length === 0"
          @click="togglePickerPageSelection"
        >
          {{ allPickerUsersSelected ? t('distributions.clearPageSelection') : t('distributions.selectPageUsers') }}
        </button>
      </div>

      <div class="overflow-hidden rounded-xl border border-gray-200 dark:border-dark-700">
        <div v-if="pickerLoading" class="p-8 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>
        <div v-else-if="pickerUsers.length === 0" class="p-8 text-center text-sm text-gray-500">{{ t('distributions.userPickerEmpty') }}</div>
        <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
          <label
            v-for="user in pickerUsers"
            :key="user.id"
            class="flex cursor-pointer items-center gap-3 px-4 py-3 transition-colors hover:bg-gray-50 dark:hover:bg-dark-800"
          >
            <input
              type="checkbox"
              class="h-4 w-4 rounded accent-[hsl(var(--primary))]"
              :checked="selectedUserIds.has(user.id)"
              @change="toggleSelectedUser(user)"
            />
            <span class="min-w-0 flex-1">
              <span class="block truncate text-sm font-medium text-gray-800 dark:text-gray-100">{{ user.username || user.email }}</span>
              <span class="block truncate text-xs text-gray-500 dark:text-gray-400">#{{ user.id }} · {{ user.email }}</span>
            </span>
            <span class="shrink-0 text-xs text-gray-400">{{ user.status }}</span>
          </label>
        </div>
      </div>

      <div v-if="pickerPages > 1" class="flex items-center justify-center gap-3">
        <button type="button" class="btn btn-secondary" :disabled="pickerPage <= 1 || pickerLoading" @click="changePickerPage(pickerPage - 1)">
          {{ t('pagination.previous') }}
        </button>
        <span class="text-sm text-gray-500">{{ pickerPage }} / {{ pickerPages }}</span>
        <button type="button" class="btn btn-secondary" :disabled="pickerPage >= pickerPages || pickerLoading" @click="changePickerPage(pickerPage + 1)">
          {{ t('pagination.next') }}
        </button>
      </div>
    </div>

    <template #footer>
      <div class="flex items-center justify-between gap-3">
        <button type="button" class="text-sm text-gray-500 hover:text-red-600" @click="clearSelectedUsers">
          {{ t('distributions.clearSelectedUsers') }}
        </button>
        <button type="button" class="btn btn-primary" @click="showUserPicker = false">
          {{ t('common.confirm') }}
        </button>
      </div>
    </template>
        </BaseDialog>
      </template>
    </ReactPageHost>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ReactPageHost from '@/console/ReactPageHost.vue'
import type { DistributionPageProps } from '@/console/react/DistributionPage'
import { adminAPI } from '@/api/admin'
import type { AdminUser } from '@/types'
import {
  batchCreateDistributions,
  createDistribution,
  deleteDistribution,
  downloadAdminDistribution,
  listAdminDistributions,
  previewDistributionAudience,
  type CreateDistributionRequest,
  type DistributionAudienceInput,
  type DistributionItem,
  type DistributionUserFilters
} from '@/api/distributions'

const { t } = useI18n()
const form = reactive<CreateDistributionRequest>({ title: '', kind: 'message', price: 0, content: '', audience: {} })
const filters = reactive<DistributionUserFilters>({ status: '', role: '', search: '', group_name: '', activity: '' })
const audienceMode = ref<'explicit' | 'advanced' | 'lines' | 'all'>('explicit')
const importLines = ref('')
const selectedUsers = ref<AdminUser[]>([])
const personalizedMode = ref(false)
const personalizedContent = ref('')
const selectedFileName = ref('')
const audienceCount = ref<number | null>(null)
const previewing = ref(false)
const submitting = ref(false)
const loading = ref(false)
const items = ref<DistributionItem[]>([])
const showUserPicker = ref(false)
const pickerUsers = ref<AdminUser[]>([])
const pickerLoading = ref(false)
const pickerSearch = ref('')
const pickerPage = ref(1)
const pickerPages = ref(1)
const pickerFilters = reactive({
  status: '' as '' | 'active' | 'disabled',
  role: '' as '' | 'admin' | 'user',
  activity: '' as '' | 'any' | 'usage' | 'balance_change' | 'none',
  group_name: ''
})
const batchJson = ref('[\n  {\n    "title": "Example",\n    "kind": "message",\n    "content": "Hello",\n    "audience": { "emails": ["user@example.com"] }\n  }\n]')

const canPersonalize = computed(() =>
  audienceMode.value === 'explicit' && (form.kind === 'text' || form.kind === 'message' || form.kind === 'paid')
)
const selectedUserIds = computed(() => new Set(selectedUsers.value.map((user) => user.id)))
const contentLines = computed(() => personalizedContent.value.split(/\r?\n/).map((line) => line.trim()).filter(Boolean))
const personalizationError = computed(() => {
  if (!personalizedMode.value) return ''
  if (selectedUsers.value.length === 0) return t('distributions.personalizedUsersRequired')
  if (contentLines.value.length !== selectedUsers.value.length) {
    return t('distributions.personalizedCountMismatch', {
      users: selectedUsers.value.length,
      contents: contentLines.value.length
    })
  }
  return ''
})
const allPickerUsersSelected = computed(() =>
  pickerUsers.value.length > 0 && pickerUsers.value.every((user) => selectedUserIds.value.has(user.id))
)
const pickerPageSize = 25
const loadDistributionPage = () => import('@/console/react/DistributionPage')

const pageProps = computed<DistributionPageProps>(() => ({
  form: { ...form },
  filters: { ...filters },
  audienceMode: audienceMode.value,
  importLines: importLines.value,
  selectedUsers: selectedUsers.value,
  selectedUserIds: Array.from(selectedUserIds.value),
  personalizedMode: personalizedMode.value,
  personalizedContent: personalizedContent.value,
  selectedFileName: selectedFileName.value,
  audienceCount: audienceCount.value,
  previewing: previewing.value,
  submitting: submitting.value,
  loading: loading.value,
  items: items.value,
  showUserPicker: showUserPicker.value,
  pickerUsers: pickerUsers.value,
  pickerLoading: pickerLoading.value,
  pickerSearch: pickerSearch.value,
  pickerPage: pickerPage.value,
  pickerPages: pickerPages.value,
  pickerFilters: { ...pickerFilters },
  batchJson: batchJson.value,
  canPersonalize: canPersonalize.value,
  personalizationError: personalizationError.value,
  allPickerUsersSelected: allPickerUsersSelected.value,
  copy: {
    title: t('distributions.title'),
    kind: t('distributions.kind'),
    kinds: {
      text: t('distributions.kinds.text'),
      message: t('distributions.kinds.message'),
      file: t('distributions.kinds.file'),
      account_export: t('distributions.kinds.account_export'),
      paid: t('distributions.kinds.paid'),
    },
    price: t('distributions.price'),
    priceHint: t('distributions.priceHint'),
    content: t('distributions.content'),
    contentPlaceholder: t('distributions.contentPlaceholder'),
    personalizedContent: t('distributions.personalizedContent'),
    personalizedContentPlaceholder: t('distributions.personalizedContentPlaceholder'),
    personalizedContentHint: t('distributions.personalizedContentHint'),
    fileHint: t('distributions.fileHint'),
    audienceMode: t('distributions.audienceMode'),
    audienceExplicit: t('distributions.audienceExplicit'),
    audienceAdvanced: t('distributions.audienceAdvanced'),
    audienceLines: t('distributions.audienceLines'),
    audienceAll: t('distributions.audienceAll'),
    recipients: t('distributions.recipients'),
    recipientsPickerHint: t('distributions.recipientsPickerHint'),
    openUserPicker: t('distributions.openUserPicker'),
    removeSelectedUser: t('distributions.removeSelectedUser'),
    noUsersSelected: t('distributions.noUsersSelected'),
    enablePersonalizedContent: t('distributions.enablePersonalizedContent'),
    filterSearch: t('distributions.filterSearch'),
    filterGroup: t('distributions.filterGroup'),
    filterAnyStatus: t('distributions.filterAnyStatus'),
    filterAnyRole: t('distributions.filterAnyRole'),
    filterAnyActivity: t('distributions.filterAnyActivity'),
    filterHasActivity: t('distributions.filterHasActivity'),
    filterHasUsage: t('distributions.filterHasUsage'),
    filterHasBalanceChange: t('distributions.filterHasBalanceChange'),
    filterNoActivity: t('distributions.filterNoActivity'),
    multilineImport: t('distributions.multilineImport'),
    multilinePlaceholder: t('distributions.multilinePlaceholder'),
    loading: t('common.loading'),
    previewAudience: t('distributions.previewAudience'),
    audienceCount: (count: number) => t('distributions.audienceCount', { count }),
    send: t('distributions.send'),
    batchJson: t('distributions.batchJson'),
    batchJsonHint: t('distributions.batchJsonHint'),
    sendBatch: t('distributions.sendBatch'),
    history: t('distributions.history'),
    deliveryStats: (item: DistributionItem) => t('distributions.deliveryStats', {
      recipients: item.recipient_count,
      read: item.read_count,
      downloads: item.download_count,
    }),
    download: t('distributions.download'),
    delete: t('common.delete'),
    empty: t('distributions.empty'),
    userPickerTitle: t('distributions.userPickerTitle'),
    userPickerSearch: t('distributions.userPickerSearch'),
    userPickerFilter: t('distributions.userPickerFilter'),
    selectedUsers: (count: number) => t('distributions.selectedUsers', { count }),
    selectPageUsers: t('distributions.selectPageUsers'),
    clearPageSelection: t('distributions.clearPageSelection'),
    userPickerEmpty: t('distributions.userPickerEmpty'),
    previous: t('pagination.previous'),
    next: t('pagination.next'),
    clearSelectedUsers: t('distributions.clearSelectedUsers'),
    confirm: t('common.confirm'),
    close: t('common.close'),
  },
  actions: {
    onTitleChange: (value) => { form.title = value },
    onKindChange: (value) => { form.kind = value },
    onPriceChange: (value) => { form.price = value },
    onContentChange: (value) => { form.content = value },
    onPersonalizedContentChange: (value) => { personalizedContent.value = value },
    onFileChange: (file) => { if (file) void selectFile(file) },
    onAudienceModeChange: (value) => { audienceMode.value = value },
    onFilterChange: (key, value) => { filters[key] = value },
    onImportLinesChange: (value) => { importLines.value = value },
    onPersonalizedModeChange: (value) => { personalizedMode.value = value },
    onOpenUserPicker: openUserPicker,
    onToggleSelectedUser: toggleSelectedUser,
    onPreviewAudience: () => { void previewAudience() },
    onSubmit: () => { void submit() },
    onBatchJsonChange: (value) => { batchJson.value = value },
    onSubmitBatch: () => { void submitBatch() },
    onRemove: (item) => { void remove(item) },
    onDownload: (item) => { void downloadAdmin(item) },
    onPickerSearchChange: (value) => { pickerSearch.value = value },
    onPickerFilterChange: (key, value) => {
      if (key === 'status') pickerFilters.status = value as typeof pickerFilters.status
      if (key === 'role') pickerFilters.role = value as typeof pickerFilters.role
      if (key === 'activity') pickerFilters.activity = value as typeof pickerFilters.activity
      if (key === 'group_name') pickerFilters.group_name = value
    },
    onLoadPickerUsers: (resetPage = false) => { void loadPickerUsers(resetPage) },
    onTogglePickerPageSelection: togglePickerPageSelection,
    onChangePickerPage: changePickerPage,
    onClearSelectedUsers: clearSelectedUsers,
    onCloseUserPicker: () => { showUserPicker.value = false },
  },
}))

function buildAudience(): DistributionAudienceInput {
  if (audienceMode.value === 'all') return { all: true }
  if (audienceMode.value === 'advanced') return { filters: { ...filters } }
  if (audienceMode.value === 'lines') return { lines: importLines.value }

  if (personalizedMode.value && canPersonalize.value) {
    const title = form.title.trim().replace(/\|/g, ' ')
    return {
      lines: selectedUsers.value
        .map((user, index) => `${user.id} | ${title} | ${contentLines.value[index]}`)
        .join('\n')
    }
  }

  return { user_ids: selectedUsers.value.map((user) => user.id) }
}

function toggleSelectedUser(user: AdminUser) {
  const index = selectedUsers.value.findIndex((candidate) => candidate.id === user.id)
  if (index >= 0) {
    selectedUsers.value.splice(index, 1)
  } else {
    selectedUsers.value.push(user)
  }
}

function clearSelectedUsers() {
  selectedUsers.value = []
}

function togglePickerPageSelection() {
  if (allPickerUsersSelected.value) {
    const pageIds = new Set(pickerUsers.value.map((user) => user.id))
    selectedUsers.value = selectedUsers.value.filter((user) => !pageIds.has(user.id))
    return
  }

  const existingIds = selectedUserIds.value
  for (const user of pickerUsers.value) {
    if (!existingIds.has(user.id)) selectedUsers.value.push(user)
  }
}

function openUserPicker() {
  showUserPicker.value = true
  if (pickerUsers.value.length === 0) void loadPickerUsers(true)
}

async function loadPickerUsers(resetPage = false) {
  if (resetPage) pickerPage.value = 1
  pickerLoading.value = true
  try {
    const result = await adminAPI.users.list(pickerPage.value, pickerPageSize, {
      search: pickerSearch.value.trim() || undefined,
      status: pickerFilters.status || undefined,
      role: pickerFilters.role || undefined,
      activity: pickerFilters.activity || undefined,
      group_name: pickerFilters.group_name.trim() || undefined,
      include_subscriptions: false,
      sort_by: 'created_at',
      sort_order: 'desc'
    })
    pickerUsers.value = result.items
    pickerPages.value = result.pages
  } finally {
    pickerLoading.value = false
  }
}

function changePickerPage(page: number) {
  pickerPage.value = page
  void loadPickerUsers()
}

async function selectFile(fileOrEvent: File | Event) {
  const file = fileOrEvent instanceof File
    ? fileOrEvent
    : (fileOrEvent.target as HTMLInputElement | null)?.files?.[0]
  if (!file) return
  if (file.size > 10 * 1024 * 1024) throw new Error(t('distributions.fileTooLarge'))
  selectedFileName.value = file.name
  form.file_name = file.name
  form.content_type = file.type || 'application/octet-stream'
  form.file_base64 = await fileToBase64(file)
}

function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(reader.error)
    reader.onload = () => resolve(String(reader.result).split(',', 2)[1] || '')
    reader.readAsDataURL(file)
  })
}

function formatPrice(price = 0) {
  return `$${price.toFixed(2)}`
}

async function previewAudience() {
  if (personalizationError.value) return
  if (audienceMode.value === 'explicit') {
    audienceCount.value = selectedUsers.value.length
    return
  }
  previewing.value = true
  try {
    const result = await previewDistributionAudience(buildAudience())
    audienceCount.value = result.count
  } finally {
    previewing.value = false
  }
}

async function submit() {
  if (personalizationError.value) return
  submitting.value = true
  try {
    await createDistribution({ ...form, audience: buildAudience() })
    form.title = ''
    form.price = 0
    form.content = ''
    form.file_name = undefined
    form.content_type = undefined
    form.file_base64 = undefined
    selectedFileName.value = ''
    selectedUsers.value = []
    personalizedMode.value = false
    personalizedContent.value = ''
    audienceCount.value = null
    await reload()
  } finally {
    submitting.value = false
  }
}

async function submitBatch() {
  const parsed = JSON.parse(batchJson.value) as CreateDistributionRequest[]
  await batchCreateDistributions(parsed)
  await reload()
}

async function reload() {
  loading.value = true
  try {
    const data = await listAdminDistributions(1, 50)
    items.value = data.items
  } finally {
    loading.value = false
  }
}

async function remove(item: DistributionItem) {
  if (!window.confirm(t('distributions.deleteConfirm'))) return
  await deleteDistribution(item.id)
  items.value = items.value.filter((candidate) => candidate.id !== item.id)
}

async function downloadAdmin(item: DistributionItem) {
  const blob = await downloadAdminDistribution(item.id)
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = item.file_name || `distribution-${item.id}`
  anchor.click()
  URL.revokeObjectURL(url)
}

onMounted(reload)

watch(
  [() => form.kind, audienceMode],
  () => {
    if (!canPersonalize.value) personalizedMode.value = false
    if (form.kind !== 'paid') form.price = 0
  }
)
</script>

<style scoped>
.input {
  width: 100%;
  min-height: var(--ui-control-height);
  border: 1px solid hsl(var(--input));
  border-radius: var(--ui-radius);
  background: hsl(var(--background));
  color: hsl(var(--foreground));
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  outline: none;
  transition: border-color 150ms ease, box-shadow 150ms ease;
}

.input:focus {
  border-color: hsl(var(--ring));
  box-shadow: 0 0 0 3px hsl(var(--ring) / 0.18);
}

.distribution-select {
  box-sizing: border-box;
  height: var(--ui-control-height);
  min-height: var(--ui-control-height);
  line-height: 1.5rem;
  padding-block: 0.5rem;
}

.distribution-selected-users {
  display: grid;
  max-height: 10rem;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 15rem), 1fr));
  gap: 0.5rem;
  overflow-y: auto;
}
</style>
