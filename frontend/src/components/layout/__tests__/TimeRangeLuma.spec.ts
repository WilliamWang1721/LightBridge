import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const timeRange = readFileSync(resolve(directory, '../TimeRangeButton.vue'), 'utf8')
const dateRange = readFileSync(resolve(directory, '../../common/DateRangePicker.vue'), 'utf8')

describe('Luma time range controls', () => {
  it('uses accessible Popover primitives and native date validation', () => {
    for (const source of [timeRange, dateRange]) {
      expect(source).toContain('<Popover')
      expect(source).toContain('<PopoverContent')
      expect(source).toContain('<Button')
      expect(source).not.toContain('<BaseDialog')
    }

    expect(dateRange).toContain('<Input')
    expect(dateRange).toContain('<Label :for-id="startInputId"')
    expect(dateRange).toContain('<Label :for-id="endInputId"')
    expect(dateRange).toContain(':min="localStartDate"')
    expect(dateRange).toContain(':max="tomorrow"')
    expect(dateRange).not.toContain("document.addEventListener('keydown'")
    expect(dateRange).not.toContain("document.addEventListener('click'")
    expect(timeRange).toContain(':aria-pressed="draftGranularity === opt.value"')
  })
})
