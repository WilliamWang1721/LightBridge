import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type { UsageLog } from '@/types'
import UserDashboardRecentUsage from '../UserDashboardRecentUsage.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/utils/format', () => ({
  formatDateTime: (value: string) => value,
}))

vi.mock('@/components/common/LoadingSpinner.vue', () => ({ default: { template: '<span />' } }))
vi.mock('@/components/common/EmptyState.vue', () => ({ default: { template: '<span />' } }))
vi.mock('@/components/icons/Icon.vue', () => ({ default: { template: '<span />' } }))

describe('UserDashboardRecentUsage', () => {
  it('renders legacy nullable values without losing zeroes or overflowing narrow cards', () => {
    const data = [{
      id: 7,
      model: null,
      requested_model: 'legacy-model-with-a-very-long-name',
      createdAt: '2026-05-08T10:00:00Z',
      input_tokens: null,
      prompt_tokens: '12',
      output_tokens: '8',
      actual_cost: '0',
      user_cost: '2.5',
      total_cost: null,
      standard_cost: '1.25',
    }] as unknown as UsageLog[]

    const wrapper = mount(UserDashboardRecentUsage, {
      props: { data, loading: false },
      global: {
        stubs: {
          Icon: true,
          EmptyState: true,
          LoadingSpinner: true,
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    expect(wrapper.text()).toContain('legacy-model-with-a-very-long-name')
    expect(wrapper.text()).toContain('$0.0000')
    expect(wrapper.text()).toContain('$1.2500')
    expect(wrapper.text()).toContain('20 tokens')
    expect(wrapper.html()).toContain('sm:flex-row')
    expect(wrapper.html()).toContain('truncate')
  })
})
