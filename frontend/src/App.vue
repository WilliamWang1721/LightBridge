<script setup lang="ts">
import { RouterView, useRouter, useRoute } from 'vue-router'
import { computed, onMounted, onBeforeUnmount, watch } from 'vue'
import Toast from '@/components/common/Toast.vue'
import NavigationProgress from '@/components/common/NavigationProgress.vue'
import UIProfileSync from '@/components/settings/UIProfileSync.vue'
import { resolveDocumentTitle } from '@/router/title'
import { resolveUIRouteSurface } from '@/ui-platform/routeSurface'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import { useAppStore, useAuthStore, useSubscriptionStore, useAnnouncementStore } from '@/stores'
import { getSetupStatus } from '@/api/setup'
import { syncProgressiveRoutes } from '@/router'
import {
  hydrateProgressiveFeatureManifest,
  isProgressiveFeatureEnabled,
  isProgressivePathDisabled,
  ProgressiveFeatures,
} from '@/utils/progressiveFeatures'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const subscriptionStore = useSubscriptionStore()
const announcementStore = useAnnouncementStore()
const subscriptionsFeatureEnabled = computed(() =>
  isProgressiveFeatureEnabled(ProgressiveFeatures.subscriptions),
)
const announcementsFeatureEnabled = computed(() =>
  isProgressiveFeatureEnabled(ProgressiveFeatures.announcements),
)
const routeSurface = computed(() => resolveUIRouteSurface(route.path, route.meta))
let visibilityListenerRegistered = false

function redirectDisabledProgressiveRoute() {
  if (!isProgressivePathDisabled(route.path)) return
  router.replace(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
}

// Runtime module toggles: after public settings changes, add/remove progressive
// routes from the matcher and leave disabled pages immediately.
watch(
  () => appStore.cachedPublicSettings,
  async () => {
    await hydrateProgressiveFeatureManifest(true).catch(() => undefined)
    await syncProgressiveRoutes()
    redirectDisabledProgressiveRoute()
  },
)

/**
 * Update favicon dynamically
 * @param logoUrl - URL of the logo to use as favicon
 */
function updateFavicon(logoUrl: string) {
  // Find existing favicon link or create new one
  let link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.type = logoUrl.endsWith('.svg') ? 'image/svg+xml' : 'image/x-icon'
  link.href = logoUrl
}

// Watch for site settings changes and update favicon/title
watch(
  () => appStore.siteLogo,
  (newLogo) => {
    if (newLogo) {
      updateFavicon(newLogo)
    }
  },
  { immediate: true }
)

// Watch for authentication state and manage subscription data + announcements
function onVisibilityChange() {
  if (
    document.visibilityState === 'visible' &&
    authStore.isAuthenticated &&
    announcementsFeatureEnabled.value
  ) {
    announcementStore.fetchAnnouncements()
  }
}

watch(
  [
    () => authStore.isAuthenticated,
    () => subscriptionsFeatureEnabled.value,
    () => announcementsFeatureEnabled.value,
  ],
  ([isAuthenticated, subscriptionsEnabled, announcementsEnabled], oldValues) => {
    const oldAuthenticated = oldValues?.[0]
    if (!isAuthenticated || !subscriptionsEnabled) {
      subscriptionStore.clear()
    } else {
      subscriptionStore.fetchActiveSubscriptions().catch((error) => {
        console.error('Failed to preload subscriptions:', error)
      })
      subscriptionStore.startPolling()
    }

    if (!isAuthenticated || !announcementsEnabled) {
      announcementStore.reset()
      if (visibilityListenerRegistered) {
        document.removeEventListener('visibilitychange', onVisibilityChange)
        visibilityListenerRegistered = false
      }
      return
    }

    if (oldAuthenticated === false) {
      setTimeout(() => {
        if (authStore.isAuthenticated && announcementsFeatureEnabled.value) {
          announcementStore.fetchAnnouncements(true)
        }
      }, 3000)
    } else {
      announcementStore.fetchAnnouncements()
    }

    if (!visibilityListenerRegistered) {
      document.addEventListener('visibilitychange', onVisibilityChange)
      visibilityListenerRegistered = true
    }
  },
  { immediate: true }
)

// Route change trigger (throttled by store)
router.afterEach(() => {
  if (authStore.isAuthenticated && announcementsFeatureEnabled.value) {
    announcementStore.fetchAnnouncements()
  }
})

onBeforeUnmount(() => {
  if (visibilityListenerRegistered) {
    document.removeEventListener('visibilitychange', onVisibilityChange)
    visibilityListenerRegistered = false
  }
})

onMounted(async () => {
  // Check if setup is needed
  try {
    const status = await getSetupStatus()
    if (status.needs_setup && route.path !== '/setup') {
      router.replace('/setup')
      return
    }
  } catch {
    // If setup endpoint fails, assume normal mode and continue
  }

  // Load public settings into appStore (will be cached for other components)
  await appStore.fetchPublicSettings()
  await hydrateProgressiveFeatureManifest(true).catch(() => undefined)

  // SSR 注入缺失时，public settings 通过异步接口才到位；此处依据最新值再同步一次
  // 渐进式路由的注册状态（与 main.ts 的同步互补，确保两条通道都被覆盖）。
  await syncProgressiveRoutes()
  redirectDisabledProgressiveRoute()

  // Re-resolve document title now that siteName is available
  document.title = resolveDocumentTitle(route.meta.title, appStore.siteName, route.meta.titleKey as string)
})
</script>

<template>
  <UIProfileSync />
  <NavigationProgress />
  <div class="ui-route-surface" :data-ui-surface="routeSurface">
    <RouterView />
  </div>
  <Toast />
  <AnnouncementPopup v-if="announcementsFeatureEnabled" />
</template>
