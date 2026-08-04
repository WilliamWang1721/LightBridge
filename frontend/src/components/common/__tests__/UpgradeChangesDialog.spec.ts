import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UpgradeChangesDialog from '../UpgradeChangesDialog.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

describe('UpgradeChangesDialog', () => {
  it('keeps both footer actions visible, aligned, and emits restart', async () => {
    const wrapper = mount(UpgradeChangesDialog, {
      props: {
        show: true,
        version: '0.3.0',
        body: '# Changes\n\n' + 'Long release note. '.repeat(500)
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<section class="modal-content"><div class="modal-body"><slot /></div><footer class="modal-footer"><slot name="footer" /></footer></section>'
          }
        }
      }
    })

    const closeButton = wrapper.get('[data-testid="upgrade-dialog-close-button"]')
    const restartButton = wrapper.get('[data-testid="restart-after-upgrade-button"]')

    expect(closeButton.classes()).toEqual(expect.arrayContaining(['btn', 'btn-secondary', 'min-h-10']))
    expect(restartButton.classes()).toEqual(expect.arrayContaining(['btn', 'btn-primary', 'min-h-10']))
    expect(restartButton.element.closest('.modal-footer')).not.toBeNull()

    await restartButton.trigger('click')
    expect(wrapper.emitted('restart')).toHaveLength(1)
  })
})
