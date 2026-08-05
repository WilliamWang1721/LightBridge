import { describe, expect, it } from 'vitest'
import router from '@/router'
import { progressiveRouteGroups } from '@/router/progressiveRoutes'
import { isKnownUIRouteSurface, resolveUIRouteSurface } from './routeSurface'

describe('UI route surface coverage', () => {
  const staticRoutes = router.getRoutes().map((route) => ({
    path: route.path,
    meta: route.meta,
    name: String(route.name ?? route.path),
  }))
  const progressiveRoutes = progressiveRouteGroups.flatMap((group) =>
    group.routes.map((route) => ({
      path: route.path,
      meta: route.meta ?? {},
      name: String(route.name ?? route.path),
    })),
  )
  const routes = [...staticRoutes, ...progressiveRoutes]

  it('classifies every static and progressive route', () => {
    expect(routes.length).toBeGreaterThan(0)
    for (const route of routes) {
      const surface = resolveUIRouteSurface(route.path, route.meta)
      expect(isKnownUIRouteSurface(surface), `${route.name} (${route.path})`).toBe(true)
    }
  })

  it('keeps administrator routes in the admin shell', () => {
    for (const route of routes.filter((item) => item.path.startsWith('/admin') || item.meta.requiresAdmin)) {
      expect(resolveUIRouteSurface(route.path, route.meta), route.name).toBe('admin')
    }
  })

  it('keeps setup, authentication and payment callbacks out of the app shell', () => {
    expect(resolveUIRouteSurface('/setup')).toBe('setup')
    expect(resolveUIRouteSurface('/login')).toBe('auth')
    expect(resolveUIRouteSurface('/auth/oidc/callback')).toBe('auth')
    expect(resolveUIRouteSurface('/payment/result')).toBe('payment')
    expect(resolveUIRouteSurface('/purchase', { requiresAuth: true })).toBe('payment')
  })

  it('classifies authenticated non-admin product routes as user surfaces', () => {
    const userRoutes = routes.filter((item) =>
      item.meta.requiresAuth &&
      !item.meta.requiresAdmin &&
      !item.path.startsWith('/payment') &&
      item.path !== '/purchase',
    )

    for (const route of userRoutes) {
      expect(resolveUIRouteSurface(route.path, route.meta), route.name).toBe('user')
    }
  })
})
