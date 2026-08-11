import type { RouteRecordRaw, Router } from 'vue-router'
import i18n from '@/i18n'

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

function anchorPath(anchor: HTMLAnchorElement): string {
  try {
    return new URL(anchor.href, window.location.href).pathname
  } catch {
    return anchor.getAttribute('href') || ''
  }
}

function findAnchor(paths: string[]): HTMLAnchorElement | null {
  const anchors = Array.from(document.querySelectorAll<HTMLAnchorElement>('a[href]'))
  return anchors.find((anchor) => paths.includes(anchorPath(anchor))) || null
}

function distributionIcon(): string {
  return `
    <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
      <path d="M4 6.5h16v11H4z" />
      <path d="m4 7 8 6 8-6" />
    </svg>`
}

function distributionLabel(): string {
  return String(i18n.global.t('nav.distributions'))
}

function createNavigationLink(
  router: Router,
  reference: HTMLAnchorElement,
  path: string,
  kind: 'user' | 'admin'
): HTMLAnchorElement {
  const link = document.createElement('a')
  link.href = router.resolve(path).href
  link.className = reference.className
  link.classList.remove('sidebar-link-active', 'router-link-active', 'router-link-exact-active')
  link.dataset.distributionNav = kind
  link.setAttribute('data-router-link', 'true')
  link.innerHTML = `${distributionIcon()}<span>${escapeHTML(distributionLabel())}</span>`
  link.addEventListener('click', (event) => {
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
    event.preventDefault()
    void router.push(path)
  })
  return link
}

function escapeHTML(value: string): string {
  const element = document.createElement('span')
  element.textContent = value
  return element.innerHTML
}

function insertAfter(reference: HTMLElement, element: HTMLElement): void {
  reference.parentNode?.insertBefore(element, reference.nextSibling)
}

function syncDistributionNavigation(router: Router): void {
  const userReference = findAnchor(['/dashboard', '/profile'])
  if (userReference) {
    const parent = userReference.parentElement
    if (parent && !parent.querySelector('[data-distribution-nav="user"]')) {
      insertAfter(userReference, createNavigationLink(router, userReference, USER_PATH, 'user'))
    }
  }

  // Keep the administrator entry top-level and independent of optional
  // marketing feature flags, so it remains visible even when announcements,
  // affiliate, redeem, and promotion modules are disabled.
  const adminReference = findAnchor(['/admin/dashboard'])
  if (adminReference) {
    const parent = adminReference.parentElement
    if (parent && !parent.querySelector('[data-distribution-nav="admin"]')) {
      insertAfter(adminReference, createNavigationLink(router, adminReference, ADMIN_PATH, 'admin'))
    }
  }

  document.querySelectorAll<HTMLAnchorElement>('[data-distribution-nav]').forEach((link) => {
    const isActive = anchorPath(link) === router.currentRoute.value.path
    link.setAttribute('aria-current', isActive ? 'page' : 'false')
    link.classList.toggle('sidebar-link-active', isActive)
    link.classList.toggle('router-link-active', isActive)
    const label = link.querySelector('span')
    const text = distributionLabel()
    if (label && label.textContent !== text) label.textContent = text
  })
}

export function installDistributionNavigation(router: Router): () => void {
  let scheduled = false
  const scheduleSync = () => {
    if (scheduled) return
    scheduled = true
    queueMicrotask(() => {
      scheduled = false
      syncDistributionNavigation(router)
    })
  }

  const observer = new MutationObserver(scheduleSync)
  observer.observe(document.body, { childList: true, subtree: true })
  const removeAfterEach = router.afterEach(scheduleSync)
  scheduleSync()

  return () => {
    observer.disconnect()
    removeAfterEach()
  }
}
