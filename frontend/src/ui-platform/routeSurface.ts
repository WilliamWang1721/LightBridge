import type { RouteMeta } from 'vue-router'

export type UIRouteSurface = 'admin' | 'user' | 'auth' | 'public' | 'setup' | 'payment'

const authPaths = [
  '/login',
  '/register',
  '/forgot-password',
  '/reset-password',
  '/email-verify',
  '/auth/',
]

const paymentPaths = ['/payment/', '/purchase']

export function resolveUIRouteSurface(path: string, meta: RouteMeta = {}): UIRouteSurface {
  if (path === '/setup' || path.startsWith('/setup/')) return 'setup'
  if (path.startsWith('/admin')) return 'admin'
  if (paymentPaths.some((prefix) => path === prefix || path.startsWith(prefix))) return 'payment'
  if (authPaths.some((prefix) => path === prefix || path.startsWith(prefix))) return 'auth'
  if (meta.requiresAdmin) return 'admin'
  if (meta.requiresAuth) return 'user'
  return 'public'
}

export function isKnownUIRouteSurface(value: unknown): value is UIRouteSurface {
  return value === 'admin' ||
    value === 'user' ||
    value === 'auth' ||
    value === 'public' ||
    value === 'setup' ||
    value === 'payment'
}
