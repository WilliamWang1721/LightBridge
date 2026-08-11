import { afterEach, describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import {
  installDistributionNavigation,
  installDistributionRoutes,
} from './runtime'

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'Home', component: { template: '<div />' } },
      { path: '/dashboard', name: 'Dashboard', component: { template: '<div />' } },
      {
        path: '/admin/dashboard',
        name: 'AdminDashboard',
        component: { template: '<div />' },
      },
    ],
  })
}

async function flushNavigationSync(): Promise<void> {
  await Promise.resolve()
  await Promise.resolve()
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('content distribution runtime', () => {
  it('installs the user and administrator routes idempotently', () => {
    const router = createTestRouter()

    installDistributionRoutes(router)
    installDistributionRoutes(router)

    expect(router.hasRoute('Distributions')).toBe(true)
    expect(router.hasRoute('AdminDistributions')).toBe(true)
    expect(router.getRoutes().filter((route) => route.name === 'Distributions')).toHaveLength(1)
    expect(router.getRoutes().filter((route) => route.name === 'AdminDistributions')).toHaveLength(1)
  })

  it('adds independent navigation entries without inheriting active classes', async () => {
    document.body.innerHTML = `
      <nav id="user-navigation">
        <a href="/dashboard" class="sidebar-link sidebar-link-active router-link-active">Dashboard</a>
      </nav>
      <nav id="admin-navigation">
        <a href="/admin/dashboard" class="sidebar-link sidebar-link-active router-link-active">Admin</a>
      </nav>
    `

    const router = createTestRouter()
    await router.push('/')
    await router.isReady()

    const stopFirst = installDistributionNavigation(router)
    const stopSecond = installDistributionNavigation(router)
    await flushNavigationSync()

    const userLinks = document.querySelectorAll('[data-distribution-nav="user"]')
    const adminLinks = document.querySelectorAll('[data-distribution-nav="admin"]')

    expect(userLinks).toHaveLength(1)
    expect(adminLinks).toHaveLength(1)
    expect(userLinks[0].classList.contains('sidebar-link-active')).toBe(false)
    expect(userLinks[0].classList.contains('router-link-active')).toBe(false)
    expect(adminLinks[0].classList.contains('sidebar-link-active')).toBe(false)
    expect(adminLinks[0].classList.contains('router-link-active')).toBe(false)
    expect(userLinks[0].getAttribute('href')).toBe('/distributions')
    expect(adminLinks[0].getAttribute('href')).toBe('/admin/distributions')

    const label = userLinks[0].querySelector('span')!
    let labelMutations = 0
    const observer = new MutationObserver((mutations) => { labelMutations += mutations.length })
    observer.observe(label, { childList: true })
    document.body.appendChild(document.createElement('div'))
    await flushNavigationSync()
    expect(labelMutations).toBe(0)
    observer.disconnect()

    stopFirst()
    stopSecond()
  })
})
