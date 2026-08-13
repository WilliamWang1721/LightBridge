import type { RouteRecordRaw, Router } from 'vue-router'

const USER_ROUTE_NAME = 'Distributions'
const ADMIN_ROUTE_NAME = 'AdminDistributions'
const USER_PATH = '/distributions'
const ADMIN_PATH = '/admin/distributions'

const distributionRoutes: RouteRecordRaw[] = [
  {
    path: USER_PATH,
    name: USER_ROUTE_NAME,
    component: () => import('@/views/user/DistributionsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Distribution Center',
      titleKey: 'distributions.userTitle',
      descriptionKey: 'distributions.userDescription'
    }
  },
  {
    path: ADMIN_PATH,
    name: ADMIN_ROUTE_NAME,
    component: () => import('@/views/admin/DistributionsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Content Distribution',
      titleKey: 'distributions.adminTitle',
      descriptionKey: 'distributions.adminDescription'
    }
  }
]

export function installDistributionRoutes(router: Router): void {
  for (const route of distributionRoutes) {
    if (route.name && !router.hasRoute(route.name)) {
      router.addRoute(route)
    }
  }
}
