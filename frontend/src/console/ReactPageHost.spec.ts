import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { act, createElement } from 'react'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import ReactPageHost from './ReactPageHost.vue'

;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true

const src = resolve(dirname(fileURLToPath(import.meta.url)))
const hostSource = readFileSync(resolve(src, 'ReactPageHost.vue'), 'utf8')
const layoutSource = readFileSync(resolve(src, '../styles/ui-layout.css'), 'utf8')
const cardSource = readFileSync(resolve(src, 'react/ui/card.tsx'), 'utf8')

function VisibleDashboardStub() {
  return createElement(
    'div',
    { 'data-slot': 'card', className: 'card bg-[hsl(var(--card))] text-[hsl(var(--card-foreground))]' },
    'API Keys',
  )
}

describe('React console host visibility', () => {
  it('mounts React pages in a real layout box instead of display:contents', () => {
    expect(hostSource).not.toMatch(/class="contents"/)
    expect(hostSource).toContain('data-console-runtime="react"')
    expect(hostSource).toContain('console-runtime-host')
    expect(hostSource).toContain('flex h-full min-h-0 min-w-0 w-full flex-1 flex-col')
    expect(layoutSource).toContain("[data-console-runtime='react']")
    expect(layoutSource).toContain('display: flex;')
    expect(layoutSource).not.toMatch(/\[data-console-runtime='react'\][^{]*contents/)
    expect(cardSource).toContain('bg-[hsl(var(--card))]')
    expect(cardSource).toContain('text-[hsl(var(--card-foreground))]')
  })

  it('renders dashboard metric cards into the host so they stay in the box tree', async () => {
    const wrapper = mount(ReactPageHost, {
      props: {
        load: async () => ({ default: VisibleDashboardStub }),
        props: {},
        errorMessage: 'Failed to load console page',
      },
      attachTo: document.body,
    })

    await act(async () => {
      await flushPromises()
    })

    const host = wrapper.get('[data-console-runtime="react"]')
    expect(host.classes()).toContain('console-runtime-host')
    expect(host.classes()).not.toContain('contents')
    expect(wrapper.text()).toContain('API Keys')
    expect(wrapper.find('[data-slot="card"]').exists()).toBe(true)

    wrapper.unmount()
  })
})
