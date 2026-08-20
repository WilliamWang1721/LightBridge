import { act, createElement } from 'react'
import { createRoot } from 'react-dom/client'
import { afterEach, describe, expect, it } from 'vitest'

import DashboardPage, { type DashboardPageProps } from './DashboardPage'

;(globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true

const props: DashboardPageProps = {
  loading: false,
  cards: [
    { key: 'apiKeys', label: 'API Keys', value: '12', hint: '8 active', order: 0 },
    { key: 'todayRequests', label: 'Today requests', value: '128', hint: 'Total: 1,024', order: 1 },
  ],
  modelDistribution: {
    modelStats: [{
      model: 'gpt-4',
      requests: 10,
      input_tokens: 800,
      output_tokens: 200,
      cache_creation_tokens: 0,
      cache_read_tokens: 0,
      total_tokens: 1000,
      cost: 1.2,
      actual_cost: 1,
      account_cost: 1.1,
    }],
    rankingItems: [],
    loading: false,
    rankingLoading: false,
    rankingError: false,
    labels: {
      modelDistribution: 'Model distribution',
      spendingRankingTitle: 'Spending',
      viewModelDistribution: 'Models',
      viewSpendingRanking: 'Spend',
      spendingRankingUser: 'User',
      spendingRankingRequests: 'Requests',
      spendingRankingTokens: 'Tokens',
      spendingRankingSpend: 'Spend',
      spendingRankingOther: 'Other',
      model: 'Model',
      requests: 'Requests',
      tokens: 'Tokens',
      actual: 'Actual',
      accountCost: 'Account',
      standard: 'Standard',
      noData: 'No data',
      failedToLoad: 'Failed',
      userPrefix: (id: number) => `User ${id}`,
    },
  },
  tokenUsageTrend: {
    trendData: [],
    loading: false,
    title: 'Token usage trend',
    noData: 'No trend data',
  },
  userTrend: [{ date: '2026-08-20', user_id: 7, username: 'alice', email: 'alice@example.com', requests: 3, tokens: 300, cost: 0.4, actual_cost: 0.3 }],
  userTrendLoading: false,
  labels: {
    recentUsage: 'Recent usage',
    userUsageTrend: 'User usage trend',
    requests: 'Requests',
    tokens: 'Tokens',
    noData: 'No usage data',
    userPrefix: (id: number) => `User ${id}`,
  },
}

const roots: Array<{ container: HTMLDivElement; unmount: () => Promise<void> }> = []

afterEach(async () => {
  await act(async () => {
    await Promise.all(roots.splice(0).map(async ({ container, unmount }) => {
      await unmount()
      container.remove()
    }))
  })
})

describe('admin DashboardPage', () => {
  it('keeps metric, chart and ranking cards visible with concrete labels', async () => {
    const container = document.createElement('div')
    container.setAttribute('data-console-runtime', 'react')
    document.body.appendChild(container)
    const root = createRoot(container)
    roots.push({ container, unmount: async () => { root.unmount() } })

    await act(async () => {
      root.render(createElement(DashboardPage, props))
    })

    expect(container.textContent).toContain('API Keys')
    expect(container.textContent).toContain('Today requests')
    expect(container.textContent).toContain('Token usage trend')
    expect(container.textContent).toContain('Model distribution')
    expect(container.textContent).toContain('User usage trend')
    expect(container.textContent).toContain('alice')
    const cards = container.querySelectorAll('[data-slot="card"]')
    expect(cards.length).toBeGreaterThanOrEqual(4)
    for (const card of cards) {
      expect(card.className.split(/\s+/)).toContain('card')
      expect(card.className).toContain('text-[hsl(var(--card-foreground))]')
    }
  })
})
