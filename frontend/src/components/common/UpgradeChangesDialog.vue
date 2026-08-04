<template>
  <BaseDialog :show="show" :title="t('version.upgradeChangesTitle')" width="wide" @close="$emit('close')">
    <div class="space-y-4">
      <div class="flex flex-wrap items-center gap-2 text-sm text-gray-500 dark:text-dark-400">
        <span class="font-medium text-gray-900 dark:text-white">{{ displayVersion(version) }}</span>
        <a
          v-if="htmlUrl"
          :href="htmlUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
        >
          {{ t('version.viewRelease') }}
        </a>
      </div>

      <!-- 富文本 / Markdown 渲染：撤销原有固定分类，直接呈现 release body -->
      <div
        class="release-notes markdown-body max-h-[56vh] overflow-y-auto rounded-xl border border-gray-200 bg-white p-5 text-sm leading-6 text-gray-700 dark:border-dark-700 dark:bg-dark-900/40 dark:text-dark-200"
        v-html="renderedBody"
      ></div>
    </div>

    <template #footer>
      <div class="grid w-full grid-cols-2 gap-3 sm:flex sm:w-auto sm:items-center">
        <button
          type="button"
          class="btn btn-secondary min-h-10 w-full sm:w-auto"
          data-testid="upgrade-dialog-close-button"
          @click="$emit('close')"
        >
          {{ t('common.close') }}
        </button>
        <button
          type="button"
          class="btn btn-primary min-h-10 w-full min-w-32 sm:w-auto"
          data-testid="restart-after-upgrade-button"
          :disabled="restarting"
          @click="$emit('restart')"
        >
          <svg
            v-if="restarting"
            class="h-4 w-4 animate-spin"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <svg
            v-else
            class="h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          {{ restarting ? t('version.restarting') : t('version.restartNow') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import BaseDialog from './BaseDialog.vue'

marked.setOptions({
  breaks: true,
  gfm: true
})

const props = defineProps<{
  show: boolean
  version?: string
  body?: string
  htmlUrl?: string
  /** 是否允许从该弹窗触发重启 */
  canUpgrade?: boolean
  /** 重启进行中（按钮转圈 + 禁用） */
  upgrading?: boolean
  restarting?: boolean
}>()

defineEmits<{
  (e: 'close'): void
  (e: 'upgrade'): void
  (e: 'restart'): void
}>()

const { t } = useI18n()

const renderedBody = computed(() => {
  const raw = (props.body || '').trim()
  if (!raw) return `<p class="text-gray-500 dark:text-dark-400">${t('version.noReleaseNotes')}</p>`
  const html = marked.parse(raw) as string
  return DOMPurify.sanitize(html)
})

function displayVersion(version?: string): string {
  const normalized = String(version || '').trim().replace(/^v/i, '')
  return normalized ? `v${normalized}` : '--'
}
</script>

<style scoped>
.release-notes :deep(h1),
.release-notes :deep(h2),
.release-notes :deep(h3),
.release-notes :deep(h4) {
  @apply font-semibold text-gray-900 dark:text-white;
}
.release-notes :deep(h1) { @apply mb-3 text-lg; }
.release-notes :deep(h2) { @apply mb-2 mt-4 text-base; }
.release-notes :deep(h3) { @apply mb-2 mt-3 text-sm; }
.release-notes :deep(h4) { @apply mb-1 mt-2 text-sm; }
.release-notes :deep(p) { @apply my-2; }
.release-notes :deep(ul) { @apply my-2 list-disc space-y-1 pl-5; }
.release-notes :deep(ol) { @apply my-2 list-decimal space-y-1 pl-5; }
.release-notes :deep(li) { @apply leading-6; }
.release-notes :deep(a) {
  @apply text-primary-600 hover:underline dark:text-primary-400;
}
.release-notes :deep(code) {
  @apply rounded bg-gray-100 px-1.5 py-0.5 font-mono text-[13px] text-gray-800 dark:bg-dark-700 dark:text-dark-100;
}
.release-notes :deep(pre) {
  @apply my-3 overflow-x-auto rounded-lg bg-gray-900 p-3 text-[13px] text-gray-100 dark:bg-black/50;
}
.release-notes :deep(pre code) {
  @apply bg-transparent p-0 text-inherit;
}
.release-notes :deep(blockquote) {
  @apply my-2 border-l-4 border-gray-200 pl-3 italic text-gray-600 dark:border-dark-600 dark:text-dark-300;
}
.release-notes :deep(hr) {
  @apply my-4 border-gray-200 dark:border-dark-700;
}
.release-notes :deep(table) {
  @apply my-3 w-full border-collapse text-sm;
}
.release-notes :deep(th),
.release-notes :deep(td) {
  @apply border border-gray-200 px-3 py-1.5 dark:border-dark-700;
}
.release-notes :deep(th) {
  @apply bg-gray-50 font-medium dark:bg-dark-800;
}
.release-notes :deep(img) {
  @apply max-w-full rounded;
}
</style>
