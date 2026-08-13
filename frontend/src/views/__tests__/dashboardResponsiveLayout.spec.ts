import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const src = resolve(dirname(fileURLToPath(import.meta.url)), '../..')
const read = (path: string) => readFileSync(resolve(src, path), 'utf8')

describe('Luma dashboard responsive layouts', () => {
  it('keeps administrator metrics single-column on narrow screens', () => {
    const source = read('console/react/DashboardPage.tsx')

    expect(source).toContain('grid grid-cols-1 gap-3 px-3')
    expect(source).toContain('sm:grid-cols-2')
    expect(source).toContain('lg:grid-cols-4')
    expect(source).toContain('grid min-w-0 grid-cols-1 gap-4 px-3 lg:grid-cols-2')
  })

  it('uses progressive metric and detail grids on the user dashboard', () => {
    const dashboard = read('views/user/DashboardView.vue')
    const stats = read('components/user/dashboard/UserDashboardStats.vue')
    const charts = read('components/user/dashboard/UserDashboardCharts.vue')

    expect(stats).toContain('grid min-w-0 grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4')
    expect(dashboard).toContain('grid min-w-0 grid-cols-1 gap-5 xl:grid-cols-3')
    expect(charts).toContain('flex min-w-0 flex-col gap-5 sm:flex-row sm:items-center')
  })

  it('delays dense operations grids until wide displays', () => {
    const dashboard = read('views/admin/ops/OpsDashboard.vue')
    const header = read('views/admin/ops/components/OpsDashboardHeader.vue')

    expect(dashboard).toContain('md:grid-cols-2 2xl:grid-cols-4')
    expect(dashboard).toContain('md:grid-cols-2 2xl:grid-cols-3')
    expect(header).toContain('grid min-w-0 grid-cols-1 gap-5 xl:grid-cols-12')
    expect(header).not.toContain("return '#10b981'")
  })
})
