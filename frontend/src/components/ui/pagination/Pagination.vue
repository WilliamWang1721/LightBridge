<template>
  <nav class="flex items-center justify-center gap-1" :aria-label="ariaLabel">
    <Button
      variant="ghost"
      size="icon"
      :disabled="disabled || normalizedPage <= 1"
      aria-label="First page"
      @click="setPage(1)"
    >
      <ChevronsLeft class="h-4 w-4" aria-hidden="true" />
    </Button>
    <Button
      variant="ghost"
      size="icon"
      :disabled="disabled || normalizedPage <= 1"
      aria-label="Previous page"
      @click="setPage(normalizedPage - 1)"
    >
      <ChevronLeft class="h-4 w-4" aria-hidden="true" />
    </Button>

    <template v-for="item in visiblePages" :key="item">
      <span
        v-if="typeof item !== 'number'"
        class="grid h-[var(--ui-control-height)] min-w-8 place-items-center px-1 text-sm text-[hsl(var(--muted-foreground))]"
        aria-hidden="true"
      >
        …
      </span>
      <Button
        v-else
        :variant="item === normalizedPage ? 'default' : 'ghost'"
        size="icon"
        :disabled="disabled"
        :aria-label="`Page ${item}`"
        :aria-current="item === normalizedPage ? 'page' : undefined"
        @click="setPage(item)"
      >
        {{ item }}
      </Button>
    </template>

    <Button
      variant="ghost"
      size="icon"
      :disabled="disabled || normalizedPage >= normalizedPageCount"
      aria-label="Next page"
      @click="setPage(normalizedPage + 1)"
    >
      <ChevronRight class="h-4 w-4" aria-hidden="true" />
    </Button>
    <Button
      variant="ghost"
      size="icon"
      :disabled="disabled || normalizedPage >= normalizedPageCount"
      aria-label="Last page"
      @click="setPage(normalizedPageCount)"
    >
      <ChevronsRight class="h-4 w-4" aria-hidden="true" />
    </Button>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from '@lucide/vue'
import { Button } from '@/components/ui/button'

const props = withDefaults(defineProps<{
  page: number
  pageCount: number
  siblingCount?: number
  disabled?: boolean
  ariaLabel?: string
}>(), {
  siblingCount: 1,
  disabled: false,
  ariaLabel: 'Pagination',
})

const emit = defineEmits<{
  (event: 'update:page', value: number): void
}>()

const normalizedPageCount = computed(() => Math.max(1, Math.floor(props.pageCount || 1)))
const normalizedPage = computed(() =>
  Math.min(normalizedPageCount.value, Math.max(1, Math.floor(props.page || 1))),
)

const visiblePages = computed<Array<number | 'start-ellipsis' | 'end-ellipsis'>>(() => {
  const count = normalizedPageCount.value
  const current = normalizedPage.value
  const siblings = Math.max(0, props.siblingCount)
  const visible = new Set<number>([1, count])

  for (let value = current - siblings; value <= current + siblings; value += 1) {
    if (value >= 1 && value <= count) visible.add(value)
  }

  if (current <= siblings + 3) {
    for (let value = 1; value <= Math.min(count, siblings * 2 + 4); value += 1) visible.add(value)
  }
  if (current >= count - siblings - 2) {
    for (let value = Math.max(1, count - siblings * 2 - 3); value <= count; value += 1) visible.add(value)
  }

  const sorted = [...visible].sort((a, b) => a - b)
  const result: Array<number | 'start-ellipsis' | 'end-ellipsis'> = []
  sorted.forEach((value, index) => {
    const previous = sorted[index - 1]
    if (previous && value - previous > 1) {
      result.push(previous === 1 ? 'start-ellipsis' : 'end-ellipsis')
    }
    result.push(value)
  })
  return result
})

function setPage(value: number) {
  if (props.disabled) return
  const next = Math.min(normalizedPageCount.value, Math.max(1, value))
  if (next !== normalizedPage.value) emit('update:page', next)
}
</script>
