import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Input from '@/components/common/Input.vue'
import TextArea from '@/components/common/TextArea.vue'
import Toggle from '@/components/common/Toggle.vue'
import ToggleSwitch from '@/components/payment/ToggleSwitch.vue'

describe('legacy component Luma adapters', () => {
  it('routes both legacy toggle APIs through the Luma switch', async () => {
    const toggle = mount(Toggle, { props: { modelValue: false } })
    const paymentToggle = mount(ToggleSwitch, { props: { checked: true, label: 'Enabled' } })

    await toggle.get('[role="switch"]').trigger('click')
    await paymentToggle.get('[role="switch"]').trigger('click')

    expect(toggle.emitted('update:modelValue')?.[0]).toEqual([true])
    expect(paymentToggle.emitted('toggle')).toHaveLength(1)
    expect(paymentToggle.get('[role="switch"]').attributes('aria-label')).toBe('Enabled')
  })

  it('preserves v-model behavior through Luma text controls', async () => {
    const input = mount(Input, { props: { modelValue: '' } })
    const textarea = mount(TextArea, { props: { modelValue: '' } })

    await input.get('input').setValue('query')
    await textarea.get('textarea').setValue('notes')

    expect(input.emitted('update:modelValue')?.at(-1)).toEqual(['query'])
    expect(textarea.emitted('update:modelValue')?.at(-1)).toEqual(['notes'])
  })

  it('gives mobile table rows a Luma-only card surface', () => {
    const base = resolve(dirname(fileURLToPath(import.meta.url)), '../../..')
    const table = readFileSync(resolve(base, 'components/common/DataTable.vue'), 'utf8')
    const compatibility = readFileSync(
      resolve(base, 'styles/ui-modern-compat-parts/part-07.css'),
      'utf8'
    )

    expect(table.match(/data-mobile-table-card/g)).toHaveLength(3)
    expect(compatibility).toContain("[data-mobile-table-card]")
    expect(compatibility).toContain("html[data-ui-mode='modern']")
    expect(compatibility).toContain("html[data-ui-mode='package']")
    expect(compatibility).toContain(':is(.select-trigger, .date-picker-trigger)')
    expect(compatibility).toContain(':is(.select-dropdown-portal, .date-picker-dropdown)')
  })
})
