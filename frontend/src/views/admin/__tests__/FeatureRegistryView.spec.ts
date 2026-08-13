import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const { listFeatureControls, setFeatureControl, clearFeatureControlOverride } = vi.hoisted(() => ({
  listFeatureControls: vi.fn(),
  setFeatureControl: vi.fn(),
  clearFeatureControlOverride: vi.fn(),
}))

vi.mock('@/api/admin/features', () => ({
  listFeatureControls,
  setFeatureControl,
  clearFeatureControlOverride,
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (error: { message?: string }, fallback: string) => error?.message || fallback,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'featureRegistry.profile') return `Resource profile: ${params?.profile}`
        if (key === 'featureRegistry.reason') return `Reason: ${params?.reason}`
        return key.replace(/\{(\w+)\}/g, (_, token) => String(params?.[token] ?? `{${token}}`))
      },
    }),
  }
})

import FeatureRegistryView from '../FeatureRegistryView.vue'

const AppLayoutStub = { template: '<div><slot /></div>' }
const ReactPageHostStub = defineComponent({
  setup(_, { slots }) {
    return () => slots.fallback?.() ?? null
  },
})
const ToggleStub = defineComponent({
  props: {
    modelValue: { type: Boolean, required: true },
    disabled: { type: Boolean, default: false },
    ariaLabel: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('button', {
      type: 'button',
      'data-test': `toggle-${props.ariaLabel}`,
      disabled: props.disabled,
      onClick: () => emit('update:modelValue', !props.modelValue),
    })
  },
})

const feature = (overrides: Record<string, unknown> = {}) => ({
  id: 'module_runtime',
  label: 'Module Runtime',
  tier: 'extension',
  activation: 'boot',
  enabled: true,
  configuredEnabled: true,
  available: true,
  controllable: true,
  override: false,
  requiresRestart: true,
  reason: 'restart_required_enable',
  minimumProfile: 'full',
  dependencies: ['core_gateway'],
  surfaces: ['menu', 'frontend_route'],
  runtimeComponents: [{ name: 'module_runtime', status: 'paused', lastError: 'waiting for restart' }],
  ...overrides,
})

describe('FeatureRegistryView', () => {
  beforeEach(() => {
    listFeatureControls.mockReset()
    setFeatureControl.mockReset()
    clearFeatureControlOverride.mockReset()
    listFeatureControls.mockResolvedValue({ profile: 'standard', features: [feature()] })
  })

  it('shows feature status and applies toggle and restore responses as complete overviews', async () => {
    setFeatureControl.mockResolvedValue({ profile: 'full', features: [feature({ enabled: false, configuredEnabled: false, override: false })] })
    clearFeatureControlOverride.mockResolvedValue({ profile: 'full', features: [feature({ enabled: true, configuredEnabled: true, override: null, requiresRestart: false })] })

    const wrapper = mount(FeatureRegistryView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          ReactPageHost: ReactPageHostStub,
          Icon: true,
          Toggle: ToggleStub,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('featureRegistry.restartRequired')
    expect(wrapper.text()).toContain('Reason: restart_required_enable')
    expect(wrapper.text()).toContain('Resource profile: standard')
    expect(wrapper.get('[data-test="runtime-error-module_runtime"]').text()).toContain('waiting for restart')

    await wrapper.get('[data-test="toggle-Module Runtime"]').trigger('click')
    await flushPromises()
    expect(setFeatureControl).toHaveBeenCalledWith('module_runtime', false)
    expect(wrapper.text()).toContain('Resource profile: full')

    await wrapper.get('button.btn-secondary.btn-sm').trigger('click')
    await flushPromises()
    expect(clearFeatureControlOverride).toHaveBeenCalledWith('module_runtime')
    expect(wrapper.find('button.btn-secondary.btn-sm').exists()).toBe(false)
  })

  it('surfaces control-plane errors without discarding the current overview', async () => {
    setFeatureControl.mockRejectedValue(new Error('feature update rejected'))

    const wrapper = mount(FeatureRegistryView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          ReactPageHost: ReactPageHostStub,
          Icon: true,
          Toggle: ToggleStub,
        },
      },
    })
    await flushPromises()

    await wrapper.get('[data-test="toggle-Module Runtime"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('feature update rejected')
    expect(wrapper.get('[data-test="feature-module_runtime"]').exists()).toBe(true)
  })
})
