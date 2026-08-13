import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { installDistributionRoutes } from './runtime'

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'Home', component: { template: '<div />' } },
    ],
  })
}

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
})
