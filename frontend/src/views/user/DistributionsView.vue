<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('distributions.userTitle') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('distributions.userDescription') }}</p>
      </div>
      <label class="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
        <input v-model="unreadOnly" type="checkbox" class="rounded border-gray-300" @change="reload" />
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
        class="rounded-xl border bg-white p-5 shadow-sm transition dark:bg-dark-800"
        :class="item.read_at ? 'border-gray-200 dark:border-dark-700' : 'border-primary-300 ring-1 ring-primary-100 dark:border-primary-700 dark:ring-primary-900/30'"
      >
        <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <span class="rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                {{ kindLabel(item.kind) }}
              </span>
              <span v-if="!item.read_at" class="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
                {{ t('distributions.unread') }}
              </span>
            </div>
            <h2 class="mt-3 text-lg font-semibold text-gray-900 dark:text-white">{{ item.title }}</h2>
            <p v-if="item.content" class="mt-2 whitespace-pre-wrap text-sm leading-6 text-gray-600 dark:text-gray-300">{{ item.content }}</p>
            <div class="mt-3 text-xs text-gray-400">{{ formatDate(item.created_at) }}</div>
          </div>

          <div class="flex shrink-0 gap-2">
            <button
              v-if="!item.read_at"
              class="rounded-lg border border-gray-200 px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 dark:border-dark-600 dark:text-gray-200 dark:hover:bg-dark-700"
              @click="markRead(item)"
            >
              {{ t('distributions.markRead') }}
            </button>
            <button
              v-if="item.has_attachment"
              class="rounded-lg bg-primary-600 px-3 py-2 text-sm font-medium text-white hover:bg-primary-700"
              @click="download(item)"
            >
              {{ item.file_name || t('distributions.download') }}
            </button>
          </div>
        </div>
      </article>
    </div>

    <div v-if="pages > 1" class="flex items-center justify-center gap-3">
      <button class="rounded-lg border px-3 py-2 text-sm disabled:opacity-40 dark:border-dark-600" :disabled="page <= 1" @click="changePage(page - 1)">
        {{ t('pagination.previous') }}
      </button>
      <span class="text-sm text-gray-500">{{ page }} / {{ pages }}</span>
      <button class="rounded-lg border px-3 py-2 text-sm disabled:opacity-40 dark:border-dark-600" :disabled="page >= pages" @click="changePage(page + 1)">
        {{ t('pagination.next') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  downloadMyDistribution,
  listMyDistributions,
  markDistributionRead,
  type DistributionItem,
  type DistributionKind
} from '@/api/distributions'

const { t, locale } = useI18n()
const items = ref<DistributionItem[]>([])
const loading = ref(false)
const unreadOnly = ref(false)
const page = ref(1)
const pages = ref(1)

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

function formatDate(value: string) {
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}

onMounted(reload)
</script>
