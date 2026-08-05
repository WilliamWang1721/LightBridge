<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('distributions.adminTitle') }}</h1>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('distributions.adminDescription') }}</p>
    </div>

    <section class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="grid gap-4 lg:grid-cols-2">
        <label class="space-y-1">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('distributions.title') }}</span>
          <input v-model="form.title" class="input" maxlength="200" />
        </label>
        <label class="space-y-1">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('distributions.kind') }}</span>
          <select v-model="form.kind" class="input">
            <option value="text">{{ t('distributions.kinds.text') }}</option>
            <option value="message">{{ t('distributions.kinds.message') }}</option>
            <option value="file">{{ t('distributions.kinds.file') }}</option>
            <option value="account_export">{{ t('distributions.kinds.account_export') }}</option>
          </select>
        </label>
      </div>

      <label class="mt-4 block space-y-1">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('distributions.content') }}</span>
        <textarea v-model="form.content" class="input min-h-28" :placeholder="t('distributions.contentPlaceholder')" />
      </label>

      <div v-if="form.kind === 'file' || form.kind === 'account_export'" class="mt-4 rounded-lg border border-dashed border-gray-300 p-4 dark:border-dark-600">
        <input type="file" :accept="form.kind === 'account_export' ? '.json,.zip,application/json,application/zip' : undefined" @change="selectFile" />
        <p class="mt-2 text-xs text-gray-500">{{ t('distributions.fileHint') }}</p>
        <p v-if="selectedFileName" class="mt-1 text-sm text-gray-700 dark:text-gray-200">{{ selectedFileName }}</p>
      </div>

      <div class="mt-5 border-t border-gray-100 pt-5 dark:border-dark-700">
        <div class="grid gap-4 lg:grid-cols-2">
          <label class="space-y-1">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('distributions.audienceMode') }}</span>
            <select v-model="audienceMode" class="input">
              <option value="explicit">{{ t('distributions.audienceExplicit') }}</option>
              <option value="advanced">{{ t('distributions.audienceAdvanced') }}</option>
              <option value="lines">{{ t('distributions.audienceLines') }}</option>
              <option value="all">{{ t('distributions.audienceAll') }}</option>
            </select>
          </label>
        </div>

        <label v-if="audienceMode === 'explicit'" class="mt-4 block space-y-1">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('distributions.recipients') }}</span>
          <textarea v-model="explicitRecipients" class="input min-h-28 font-mono text-sm" :placeholder="t('distributions.recipientsPlaceholder')" />
        </label>

        <div v-if="audienceMode === 'advanced'" class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-5">
          <input v-model="filters.search" class="input" :placeholder="t('distributions.filterSearch')" />
          <input v-model="filters.group_name" class="input" :placeholder="t('distributions.filterGroup')" />
          <select v-model="filters.status" class="input">
            <option value="">{{ t('distributions.filterAnyStatus') }}</option>
            <option value="active">active</option>
            <option value="disabled">disabled</option>
          </select>
          <select v-model="filters.role" class="input">
            <option value="">{{ t('distributions.filterAnyRole') }}</option>
            <option value="user">user</option>
            <option value="admin">admin</option>
          </select>
          <select v-model="filters.activity" class="input">
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
          <button class="rounded-lg border border-gray-200 px-4 py-2 text-sm hover:bg-gray-50 dark:border-dark-600 dark:hover:bg-dark-700" :disabled="previewing" @click="previewAudience">
            {{ previewing ? t('common.loading') : t('distributions.previewAudience') }}
          </button>
          <span v-if="audienceCount !== null" class="text-sm text-gray-600 dark:text-gray-300">
            {{ t('distributions.audienceCount', { count: audienceCount }) }}
          </span>
        </div>
      </div>

      <div class="mt-5 flex justify-end">
        <button class="rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700 disabled:opacity-50" :disabled="submitting" @click="submit">
          {{ submitting ? t('common.loading') : t('distributions.send') }}
        </button>
      </div>
    </section>

    <details class="rounded-xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
      <summary class="cursor-pointer font-medium text-gray-800 dark:text-gray-100">{{ t('distributions.batchJson') }}</summary>
      <p class="mt-2 text-sm text-gray-500">{{ t('distributions.batchJsonHint') }}</p>
      <textarea v-model="batchJson" class="input mt-3 min-h-40 font-mono text-sm" />
      <button class="mt-3 rounded-lg bg-gray-900 px-4 py-2 text-sm text-white dark:bg-gray-100 dark:text-gray-900" @click="submitBatch">
        {{ t('distributions.sendBatch') }}
      </button>
    </details>

    <section class="rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="border-b border-gray-100 px-5 py-4 font-medium text-gray-800 dark:border-dark-700 dark:text-gray-100">{{ t('distributions.history') }}</div>
      <div v-if="loading" class="p-8 text-center text-gray-500">{{ t('common.loading') }}</div>
      <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
        <div v-for="item in items" :key="item.id" class="flex flex-col gap-3 p-5 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <div class="flex items-center gap-2">
              <span class="font-medium text-gray-900 dark:text-white">{{ item.title }}</span>
              <span class="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">{{ t(`distributions.kinds.${item.kind}`) }}</span>
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
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
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
const form = reactive<CreateDistributionRequest>({ title: '', kind: 'message', content: '', audience: {} })
const filters = reactive<DistributionUserFilters>({ status: '', role: '', search: '', group_name: '', activity: '' })
const audienceMode = ref<'explicit' | 'advanced' | 'lines' | 'all'>('explicit')
const explicitRecipients = ref('')
const importLines = ref('')
const selectedFileName = ref('')
const audienceCount = ref<number | null>(null)
const previewing = ref(false)
const submitting = ref(false)
const loading = ref(false)
const items = ref<DistributionItem[]>([])
const batchJson = ref('[\n  {\n    "title": "Example",\n    "kind": "message",\n    "content": "Hello",\n    "audience": { "emails": ["user@example.com"] }\n  }\n]')

function buildAudience(): DistributionAudienceInput {
  if (audienceMode.value === 'all') return { all: true }
  if (audienceMode.value === 'advanced') return { filters: { ...filters } }
  if (audienceMode.value === 'lines') return { lines: importLines.value }

  const user_ids: number[] = []
  const emails: string[] = []
  for (const line of explicitRecipients.value.split(/\r?\n|,/)) {
    const value = line.trim()
    if (!value) continue
    if (/^\d+$/.test(value)) user_ids.push(Number(value))
    else emails.push(value)
  }
  return { user_ids, emails }
}

async function selectFile(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
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

async function previewAudience() {
  previewing.value = true
  try {
    const result = await previewDistributionAudience(buildAudience())
    audienceCount.value = result.count
  } finally {
    previewing.value = false
  }
}

async function submit() {
  submitting.value = true
  try {
    await createDistribution({ ...form, audience: buildAudience() })
    form.title = ''
    form.content = ''
    form.file_name = undefined
    form.content_type = undefined
    form.file_base64 = undefined
    selectedFileName.value = ''
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
</script>

<style scoped>
.input {
  @apply w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none transition focus:border-primary-500 focus:ring-2 focus:ring-primary-100 dark:border-dark-600 dark:bg-dark-900 dark:text-gray-100 dark:focus:ring-primary-900/30;
}
</style>
