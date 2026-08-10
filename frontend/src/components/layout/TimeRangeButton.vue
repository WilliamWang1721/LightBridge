<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        variant="ghost"
        size="icon"
        :title="t('admin.dashboard.timeRange')"
        :aria-label="t('admin.dashboard.timeRange')"
      >
        <Icon name="clock" size="md" />
      </Button>
    </PopoverTrigger>

    <PopoverContent align="end" class="w-[min(24rem,calc(100vw-1rem))]">
      <div class="space-y-5">
        <h2 class="text-base font-semibold text-[hsl(var(--foreground))]">
          {{ t('admin.dashboard.timeRange') }}
        </h2>
        <div>
          <Label>{{ t('admin.dashboard.timeRange') }}</Label>
          <div class="mt-2">
          <DateRangePicker
            :start-date="draftStart"
            :end-date="draftEnd"
            @update:start-date="draftStart = $event"
            @update:end-date="draftEnd = $event"
            @change="onPickerChange"
          />
          </div>
        </div>

        <div>
          <Label :id="granularityLabelId">{{ t('admin.dashboard.granularity') }}</Label>
          <div class="mt-2 grid grid-cols-2 gap-2" role="group" :aria-labelledby="granularityLabelId">
            <Button
              v-for="opt in granularityOptions"
              :key="opt.value"
              size="sm"
              :variant="draftGranularity === opt.value ? 'default' : 'outline'"
              :aria-pressed="draftGranularity === opt.value"
              @click="draftGranularity = opt.value"
            >
              {{ opt.label }}
            </Button>
          </div>
        </div>

        <div class="flex justify-between gap-3 border-t border-[hsl(var(--border))] pt-4">
          <Button variant="outline" size="sm" @click="handleReset">
            {{ t('common.reset') }}
          </Button>
          <div class="flex gap-3">
            <Button variant="ghost" size="sm" @click="open = false">
              {{ t('common.close') }}
            </Button>
            <Button size="sm" @click="handleApply">
              {{ t('common.apply') }}
            </Button>
          </div>
        </div>
      </div>
    </PopoverContent>
  </Popover>
</template>

<script setup lang="ts">
import { ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { useTimeRangeStore, type DashboardGranularity } from '@/stores/timeRange'

const { t } = useI18n()
const store = useTimeRangeStore()

const open = ref(false)
const granularityLabelId = `${useId()}-granularity`
const draftStart = ref(store.startDate)
const draftEnd = ref(store.endDate)
const draftGranularity = ref<DashboardGranularity>(store.granularity)

watch(open, (val) => {
  if (val) {
    draftStart.value = store.startDate
    draftEnd.value = store.endDate
    draftGranularity.value = store.granularity
  }
})

const granularityOptions = [
  { value: 'hour' as const, label: t('admin.dashboard.hour') },
  { value: 'day' as const, label: t('admin.dashboard.day') }
]

function onPickerChange(range: { startDate: string; endDate: string }) {
  if (range.startDate) draftStart.value = range.startDate
  if (range.endDate) draftEnd.value = range.endDate
}

function handleApply() {
  store.setRange(draftStart.value, draftEnd.value)
  store.setGranularity(draftGranularity.value)
  open.value = false
}

function handleReset() {
  store.reset()
  draftStart.value = store.startDate
  draftEnd.value = store.endDate
  draftGranularity.value = store.granularity
}
</script>
