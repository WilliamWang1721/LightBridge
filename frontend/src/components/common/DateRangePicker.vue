<template>
  <Popover v-model:open="isOpen">
    <PopoverTrigger as-child>
      <Button
        variant="outline"
        class="date-picker-trigger"
        :aria-label="t('dates.selectDateRange')"
      >
        <Icon name="calendar" size="sm" />
        <span class="font-medium">
        {{ displayValue }}
        </span>
        <Icon
          name="chevronDown"
          size="sm"
          :class="['transition-transform duration-200', isOpen && 'rotate-180']"
        />
      </Button>
    </PopoverTrigger>

    <PopoverContent align="start" class="date-picker-dropdown w-[min(20rem,calc(100vw-1rem))] p-0">
      <div class="grid grid-cols-2 gap-1 p-2">
        <Button
          v-for="preset in presets"
          :key="preset.value"
          size="sm"
          :variant="isPresetActive(preset) ? 'secondary' : 'ghost'"
          :aria-pressed="isPresetActive(preset)"
          class="justify-start"
          @click="selectPreset(preset)"
        >
            {{ t(preset.labelKey) }}
        </Button>
      </div>

      <div class="border-t border-[hsl(var(--border))] p-3">
        <div class="flex items-end gap-2">
          <div class="min-w-0 flex-1 space-y-1.5">
            <Label :for-id="startInputId">{{ t('dates.startDate') }}</Label>
            <Input
              :id="startInputId"
              v-model="localStartDate"
              type="date"
              :max="localEndDate || tomorrow"
              :aria-invalid="!isRangeValid"
              class="date-picker-input h-9 px-2"
              @change="onDateChange"
            />
          </div>
          <Icon name="arrowRight" size="sm" class="mb-2.5 shrink-0 text-[hsl(var(--muted-foreground))]" />
          <div class="min-w-0 flex-1 space-y-1.5">
            <Label :for-id="endInputId">{{ t('dates.endDate') }}</Label>
            <Input
              :id="endInputId"
              v-model="localEndDate"
              type="date"
              :min="localStartDate"
              :max="tomorrow"
              :aria-invalid="!isRangeValid"
              class="date-picker-input h-9 px-2"
              @change="onDateChange"
            />
          </div>
        </div>

        <div class="mt-3 flex justify-end">
          <Button size="sm" class="date-picker-apply" :disabled="!isRangeValid" @click="apply">
            {{ t('dates.apply') }}
          </Button>
        </div>
      </div>
    </PopoverContent>
  </Popover>
</template>

<script setup lang="ts">
import { ref, computed, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

interface DatePreset {
  labelKey: string
  value: string
  getRange: () => { start: string; end: string }
}

interface Props {
  startDate: string
  endDate: string
}

interface Emits {
  (e: 'update:startDate', value: string): void
  (e: 'update:endDate', value: string): void
  (e: 'change', range: { startDate: string; endDate: string; preset: string | null }): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t, locale } = useI18n()

const isOpen = ref(false)
const instanceId = useId()
const startInputId = `${instanceId}-start`
const endInputId = `${instanceId}-end`
const localStartDate = ref(props.startDate)
const localEndDate = ref(props.endDate)
const activePreset = ref<string | null>('last24Hours')
const isRangeValid = computed(() => Boolean(localStartDate.value && localEndDate.value && localStartDate.value <= localEndDate.value))

const today = computed(() => {
  // Use local timezone to avoid UTC timezone issues
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
})

// Tomorrow's date - used for max date to handle timezone differences
// When user is in a timezone behind the server, "today" on server might be "tomorrow" locally
const tomorrow = computed(() => {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  return formatDateToString(d)
})

// Helper function to format date to YYYY-MM-DD using local timezone
const formatDateToString = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const presets: DatePreset[] = [
  {
    labelKey: 'dates.today',
    value: 'today',
    getRange: () => {
      const t = today.value
      return { start: t, end: t }
    }
  },
  {
    labelKey: 'dates.yesterday',
    value: 'yesterday',
    getRange: () => {
      const d = new Date()
      d.setDate(d.getDate() - 1)
      const yesterday = formatDateToString(d)
      return { start: yesterday, end: yesterday }
    }
  },
  {
    labelKey: 'dates.yesterdayToToday',
    value: 'last24Hours',
    getRange: () => {
      const end = new Date()
      const start = new Date(end)
      start.setDate(start.getDate() - 1)
      return {
        start: formatDateToString(start),
        end: formatDateToString(end)
      }
    }
  },
  {
    labelKey: 'dates.last7Days',
    value: '7days',
    getRange: () => {
      const end = today.value
      const d = new Date()
      d.setDate(d.getDate() - 6)
      const start = formatDateToString(d)
      return { start, end }
    }
  },
  {
    labelKey: 'dates.last14Days',
    value: '14days',
    getRange: () => {
      const end = today.value
      const d = new Date()
      d.setDate(d.getDate() - 13)
      const start = formatDateToString(d)
      return { start, end }
    }
  },
  {
    labelKey: 'dates.last30Days',
    value: '30days',
    getRange: () => {
      const end = today.value
      const d = new Date()
      d.setDate(d.getDate() - 29)
      const start = formatDateToString(d)
      return { start, end }
    }
  },
  {
    labelKey: 'dates.thisMonth',
    value: 'thisMonth',
    getRange: () => {
      const now = new Date()
      const start = formatDateToString(new Date(now.getFullYear(), now.getMonth(), 1))
      return { start, end: today.value }
    }
  },
  {
    labelKey: 'dates.lastMonth',
    value: 'lastMonth',
    getRange: () => {
      const now = new Date()
      const start = formatDateToString(new Date(now.getFullYear(), now.getMonth() - 1, 1))
      const end = formatDateToString(new Date(now.getFullYear(), now.getMonth(), 0))
      return { start, end }
    }
  }
]

const displayValue = computed(() => {
  if (activePreset.value) {
    const preset = presets.find((p) => p.value === activePreset.value)
    if (preset) return t(preset.labelKey)
  }

  if (localStartDate.value && localEndDate.value) {
    if (localStartDate.value === localEndDate.value) {
      return formatDate(localStartDate.value)
    }
    return `${formatDate(localStartDate.value)} - ${formatDate(localEndDate.value)}`
  }

  return t('dates.selectDateRange')
})

const formatDate = (dateStr: string): string => {
  const date = new Date(dateStr + 'T00:00:00')
  const dateLocale = locale.value === 'zh' ? 'zh-CN' : 'en-US'
  return date.toLocaleDateString(dateLocale, { month: 'short', day: 'numeric' })
}

const isPresetActive = (preset: DatePreset): boolean => {
  return activePreset.value === preset.value
}

const selectPreset = (preset: DatePreset) => {
  const range = preset.getRange()
  localStartDate.value = range.start
  localEndDate.value = range.end
  activePreset.value = preset.value
}

const onDateChange = () => {
  // Check if current dates match any preset
  activePreset.value = null
  for (const preset of presets) {
    const range = preset.getRange()
    if (range.start === localStartDate.value && range.end === localEndDate.value) {
      activePreset.value = preset.value
      break
    }
  }
}

const apply = () => {
  if (!isRangeValid.value) return
  emit('update:startDate', localStartDate.value)
  emit('update:endDate', localEndDate.value)
  emit('change', {
    startDate: localStartDate.value,
    endDate: localEndDate.value,
    preset: activePreset.value
  })
  isOpen.value = false
}

// Sync local state with props
watch(
  () => props.startDate,
  (val) => {
    localStartDate.value = val
    onDateChange()
  }
)

watch(
  () => props.endDate,
  (val) => {
    localEndDate.value = val
    onDateChange()
  }
)

onDateChange()
</script>

<style scoped>
.date-picker-input::-webkit-calendar-picker-indicator {
  cursor: pointer;
  opacity: 0.65;
  filter: invert(0.5);
}

.dark .date-picker-input::-webkit-calendar-picker-indicator {
  filter: invert(0.7);
}

.date-picker-input::-webkit-calendar-picker-indicator:hover {
  opacity: 1;
}
</style>
