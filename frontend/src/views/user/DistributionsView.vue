<template>
  <AppLayout>
    <div class="space-y-6">
    <div class="flex justify-end">
      <label class="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
        <input v-model="unreadOnly" type="checkbox" class="rounded accent-[hsl(var(--primary))]" @change="reload" />
        {{ t('distributions.unreadOnly') }}
      </label>
    </div>

    <div v-if="loading" class="rounded-xl border border-gray-200 bg-white p-8 text-center text-gray-500 dark:border-dark-700 dark:bg-dark-800">
      {{ t('common.loading') }}
    </div>

    <div v-else-if="items.length === 0" class="rounded-xl border border-dashed border-gray-300 bg-white p-10 text-center dark:border-dark-600 dark:bg-dark-800">
      <div class="text-lg font-medium text-gray-700 dark:text-gray-200">{{ t('distributions.empty') }}</div>
      <div class="mt-1 text-sm text-gray-500">{{ t('distributions.emptyHint') }}</div>
    </div>

    <div v-else class="space-y-3">
      <article
        v-for="item in items"
        :key="item.id"
        class="card p-5 transition"
        :class="item.read_at ? '' : 'border-[hsl(var(--primary)/.35)] ring-1 ring-[hsl(var(--primary)/.14)]'"
      >
        <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <span class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                {{ kindLabel(item.kind) }}
              </span>
              <span v-if="item.kind === 'paid'" class="rounded-full bg-amber-50 px-2.5 py-1 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
                {{ formatPrice(item.price) }}
              </span>
              <span v-if="isPaidPending(item)" class="rounded-full bg-amber-50 px-2.5 py-1 text-xs font-medium text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">
                {{ t('distributions.pendingAcceptance') }}
              </span>
              <span v-else-if="isPaidRejected(item)" class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-500 dark:bg-dark-700 dark:text-gray-400">
                {{ t('distributions.rejected') }}
              </span>
              <span v-else-if="!item.read_at" class="rounded-full bg-[hsl(var(--primary)/.1)] px-2.5 py-1 text-xs font-medium text-[hsl(var(--primary))]">
                {{ t('distributions.unread') }}
              </span>
            </div>
            <h2 class="mt-3 text-lg font-semibold text-gray-900 dark:text-white">{{ item.title }}</h2>
            <p v-if="isPaidPending(item)" class="mt-2 text-sm leading-6 text-amber-700 dark:text-amber-300">
              {{ t('distributions.paidAcceptanceHint', { price: formatPrice(item.price) }) }}
            </p>
            <p v-else-if="isPaidRejected(item)" class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
              {{ t('distributions.paidRejectedHint') }}
            </p>
            <p v-else-if="item.content" class="mt-2 whitespace-pre-wrap text-sm leading-6 text-gray-600 dark:text-gray-300">{{ item.content }}</p>
            <div class="mt-3 text-xs text-gray-400">{{ formatDate(item.created_at) }}</div>
          </div>

          <div class="flex shrink-0 gap-2">
            <template v-if="isPaidPending(item)">
              <button
                class="btn btn-primary btn-sm"
                :disabled="pendingAction === item.id"
                @click="accept(item)"
              >
                {{ pendingAction === item.id ? t('common.loading') : t('distributions.acceptPaid') }}
              </button>
              <button
                class="btn btn-secondary btn-sm"
                :disabled="pendingAction === item.id"
                @click="reject(item)"
              >
                {{ t('distributions.rejectPaid') }}
              </button>
            </template>
            <button
              v-if="!isPaidPending(item) && !isPaidRejected(item) && !item.read_at"
              class="btn btn-secondary btn-sm"
              @click="markRead(item)"
            >
              {{ t('distributions.markRead') }}
            </button>
            <button
              v-if="!isPaidPending(item) && !isPaidRejected(item) && item.has_attachment"
              class="btn btn-primary btn-sm"
              @click="download(item)"
            >
              {{ item.file_name || t('distributions.download') }}
            </button>
          </div>
        </div>
      </article>
    </div>

    <div v-if="pages > 1" class="flex items-center justify-center gap-3">
      <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="changePage(page - 1)">
        {{ t('pagination.previous') }}
      </button>
      <span class="text-sm text-gray-500">{{ page }} / {{ pages }}</span>
      <button class="btn btn-secondary btn-sm" :disabled="page >= pages" @click="changePage(page + 1)">
        {{ t('pagination.next') }}
      </button>
    </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAuthStore } from '@/stores/auth'
import {
  acceptMyDistribution,
  downloadMyDistribution,
  listMyDistributions,
  markDistributionRead,
  rejectMyDistribution,
  type DistributionItem,
  type DistributionKind
} from '@/api/distributions'

const { t, locale } = useI18n()
const authStore = useAuthStore()
const items = ref<DistributionItem[]>([])
const loading = ref(false)
const unreadOnly = ref(false)
const page = ref(1)
const pages = ref(1)
const pendingAction = ref<number | null>(null)

async function reload() {
  loading.value = true
  try {
    const data = await listMyDistributions(page.value, 20, unreadOnly.value)
    items.value = data.items
    pages.value = data.pages
  } finally {
    loading.value = false
  }
}

async function markRead(item: DistributionItem) {
  await markDistributionRead(item.id)
  item.read_at = new Date().toISOString()
  if (unreadOnly.value) {
    items.value = items.value.filter((candidate) => candidate.id !== item.id)
  }
}

async function download(item: DistributionItem) {
  const blob = await downloadMyDistribution(item.id)
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = item.file_name || `distribution-${item.id}`
  anchor.click()
  URL.revokeObjectURL(url)
  item.downloaded_at = new Date().toISOString()
  item.read_at ||= item.downloaded_at
}

function changePage(next: number) {
  page.value = next
  void reload()
}

function kindLabel(kind: DistributionKind) {
  return t(`distributions.kinds.${kind}`)
}

function isPaidPending(item: DistributionItem) {
  return item.kind === 'paid' && !item.accepted_at && !item.rejected_at
}

function isPaidRejected(item: DistributionItem) {
  return item.kind === 'paid' && Boolean(item.rejected_at)
}

function formatPrice(price = 0) {
  return `$${price.toFixed(2)}`
}

async function accept(item: DistributionItem) {
  if (!window.confirm(t('distributions.acceptPaidConfirm', { price: formatPrice(item.price) }))) return
  pendingAction.value = item.id
  try {
    await acceptMyDistribution(item.id)
    await reload()
    void authStore.refreshUser().catch(() => undefined)
  } finally {
    pendingAction.value = null
  }
}

async function reject(item: DistributionItem) {
  if (!window.confirm(t('distributions.rejectPaidConfirm'))) return
  pendingAction.value = item.id
  try {
    await rejectMyDistribution(item.id)
    await reload()
  } finally {
    pendingAction.value = null
  }
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}

onMounted(reload)
</script>
