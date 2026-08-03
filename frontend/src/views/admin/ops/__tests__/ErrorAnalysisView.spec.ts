import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import type { OpsErrorDetail, OpsErrorLog } from '@/api/admin/ops'

const {
  listRequestErrors,
  getRequestErrorDetail,
  listRequestErrorUpstreamErrors,
  markRequestErrorsRead,
  deleteRequestErrorsBatch,
  listAccounts,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  listRequestErrors: vi.fn(),
  getRequestErrorDetail: vi.fn(),
  listRequestErrorUpstreamErrors: vi.fn(),
  markRequestErrorsRead: vi.fn(),
  deleteRequestErrorsBatch: vi.fn(),
  listAccounts: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listRequestErrors,
    getRequestErrorDetail,
    listRequestErrorUpstreamErrors,
  },
  markRequestErrorsRead,
  deleteRequestErrorsBatch,
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
    },
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    currentVersion: 'test-version',
    showError,
    showSuccess,
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

import ErrorAnalysisView from '../ErrorAnalysisView.vue'

const AppLayoutStub = defineComponent({
  template: '<div><slot /></div>',
})

const SelectStub = defineComponent({
  props: {
    modelValue: {
      type: [String, Number],
      default: '',
    },
  },
  emits: ['update:modelValue'],
  template: '<div class="select-stub" />',
})

const originalDocumentHiddenDescriptor = Object.getOwnPropertyDescriptor(document, 'hidden')
let documentIsHidden = false
let activeWrapper: ReturnType<typeof mount> | undefined

function makeError(id: number, overrides: Partial<OpsErrorLog> = {}): OpsErrorLog {
  return {
    id,
    created_at: '2026-08-02T00:00:00Z',
    phase: 'routing',
    type: 'api_error',
    error_owner: 'platform',
    error_source: 'gateway',
    severity: 'P1',
    status_code: 503,
    platform: 'openai',
    model: 'gpt-4o-mini',
    resolved: false,
    client_request_id: `client-${id}`,
    request_id: `request-${id}`,
    message: `Error ${id}`,
    provider_error_code: '',
    provider_error_type: '',
    user_email: 'user@example.com',
    account_name: '',
    group_name: '',
    is_read: true,
    ...overrides,
  }
}

function makeDetail(item: OpsErrorLog): OpsErrorDetail {
  return {
    ...item,
    error_body: '',
    user_agent: 'vitest',
    is_business_limited: false,
  }
}

function listResponse(items: OpsErrorLog[]) {
  return {
    items,
    total: items.length,
    page: 1,
    page_size: 10,
  }
}

function mountView() {
  activeWrapper = mount(ErrorAnalysisView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        Icon: true,
        Select: SelectStub,
        RouterLink: true,
      },
    },
  })
  return activeWrapper
}

function setDocumentHidden(hidden: boolean) {
  documentIsHidden = hidden
  document.dispatchEvent(new Event('visibilitychange'))
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.clearAllMocks()
  documentIsHidden = false
  Object.defineProperty(document, 'hidden', {
    configurable: true,
    get: () => documentIsHidden,
  })

  const initialError = makeError(10)
  listRequestErrors.mockResolvedValue(listResponse([initialError]))
  getRequestErrorDetail.mockImplementation(async (id: number) => makeDetail(makeError(id)))
  listRequestErrorUpstreamErrors.mockResolvedValue({ items: [], total: 0 })
  markRequestErrorsRead.mockResolvedValue({ affected: 1 })
  deleteRequestErrorsBatch.mockResolvedValue({ deleted: 0 })
  listAccounts.mockResolvedValue({ items: [], total: 0 })
})

afterEach(() => {
  activeWrapper?.unmount()
  activeWrapper = undefined
  vi.useRealTimers()

  if (originalDocumentHiddenDescriptor) {
    Object.defineProperty(document, 'hidden', originalDocumentHiddenDescriptor)
  } else {
    delete (document as Document & { hidden?: boolean }).hidden
  }
})

describe('ErrorAnalysisView automatic refresh', () => {
  it('silently reloads the current results every five seconds without reloading the selected detail', async () => {
    const selectedError = makeError(10)
    const newerError = makeError(11)
    listRequestErrors
      .mockResolvedValueOnce(listResponse([selectedError]))
      .mockResolvedValueOnce(listResponse([newerError, selectedError]))

    mountView()
    await flushPromises()

    expect(getRequestErrorDetail).toHaveBeenCalledTimes(1)
    expect(getRequestErrorDetail).toHaveBeenLastCalledWith(selectedError.id)

    await vi.advanceTimersByTimeAsync(5_000)
    await flushPromises()

    expect(listRequestErrors).toHaveBeenCalledTimes(2)
    expect(getRequestErrorDetail).toHaveBeenCalledTimes(1)
    expect(markRequestErrorsRead).not.toHaveBeenCalled()
  })

  it('pauses while hidden and immediately catches up once visible again', async () => {
    mountView()
    await flushPromises()

    setDocumentHidden(true)
    await vi.advanceTimersByTimeAsync(15_000)
    await flushPromises()
    expect(listRequestErrors).toHaveBeenCalledTimes(1)

    setDocumentHidden(false)
    await flushPromises()
    expect(listRequestErrors).toHaveBeenCalledTimes(2)

    await vi.advanceTimersByTimeAsync(5_000)
    await flushPromises()
    expect(listRequestErrors).toHaveBeenCalledTimes(3)
  })

  it('does not start another automatic request while a silent refresh is still pending', async () => {
    let resolvePendingRefresh: ((response: ReturnType<typeof listResponse>) => void) | undefined
    const pendingRefresh = new Promise<ReturnType<typeof listResponse>>((resolve) => {
      resolvePendingRefresh = resolve
    })
    const initialError = makeError(10)

    listRequestErrors
      .mockResolvedValueOnce(listResponse([initialError]))
      .mockImplementationOnce(() => pendingRefresh)

    mountView()
    await flushPromises()

    vi.advanceTimersByTime(5_000)
    await Promise.resolve()
    expect(listRequestErrors).toHaveBeenCalledTimes(2)

    vi.advanceTimersByTime(10_000)
    await Promise.resolve()
    expect(listRequestErrors).toHaveBeenCalledTimes(2)

    resolvePendingRefresh?.(listResponse([initialError]))
    await flushPromises()

    await vi.advanceTimersByTimeAsync(5_000)
    await flushPromises()
    expect(listRequestErrors).toHaveBeenCalledTimes(3)
  })

  it('keeps the visible list and suppresses errors when a silent refresh fails', async () => {
    const initialError = makeError(10)
    listRequestErrors
      .mockResolvedValueOnce(listResponse([initialError]))
      .mockRejectedValueOnce(new Error('background refresh failed'))

    const wrapper = mountView()
    await flushPromises()

    await vi.advanceTimersByTimeAsync(5_000)
    await flushPromises()

    expect(showError).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Error 10')
  })

  it('cleans up the timer and visibility listener when unmounted', async () => {
    const wrapper = mountView()
    await flushPromises()
    wrapper.unmount()
    activeWrapper = undefined

    await vi.advanceTimersByTimeAsync(5_000)
    document.dispatchEvent(new Event('visibilitychange'))
    await flushPromises()

    expect(listRequestErrors).toHaveBeenCalledTimes(1)
  })
})
