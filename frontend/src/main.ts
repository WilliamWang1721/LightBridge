import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router, { syncProgressiveRoutes } from './router'
import i18n, { initI18n } from './i18n'
import { useAppStore } from '@/stores/app'
import { hydrateProgressiveFeatureManifest } from '@/utils/progressiveFeatures'
import { initializeUIPlatform } from '@/ui-platform/runtime'
import './style.css'
import './styles/ui-platform.css'

function initThemeClass() {
  const savedTheme = localStorage.getItem('theme')
  const shouldUseDark =
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  document.documentElement.classList.toggle('dark', shouldUseDark)
}

async function bootstrap() {
  // Apply theme class globally before app mount to keep all routes consistent.
  initThemeClass()

  // Resolve the independent UI axes before Vue mounts so legacy, modern, and
  // package modes can share one startup path without a visual configuration flash.
  initializeUIPlatform()

  const app = createApp(App)
  const pinia = createPinia()
  app.use(pinia)

  // Initialize settings from injected config BEFORE mounting (prevents flash)
  // This must happen after pinia is installed but before router and i18n
  const appStore = useAppStore()
  appStore.initFromInjectedConfig()

  // Resolve the backend feature catalog before registering optional routes.
  // Bootstrap remains resilient against older or temporarily unavailable backends.
  await hydrateProgressiveFeatureManifest().catch(() => undefined)
  await syncProgressiveRoutes()

  // Set document title immediately after config is loaded
  if (appStore.siteName && appStore.siteName !== 'LightBridge') {
    document.title = `${appStore.siteName} - AI API Gateway`
  }

  await initI18n()

  app.use(router)
  app.use(i18n)

  // 等待路由器完成初始导航后再挂载，避免竞态条件导致的空白渲染
  await router.isReady()
  app.mount('#app')
}

bootstrap().catch((error: unknown) => {
  console.error('Application bootstrap failed:', error)
  const root = document.getElementById('app')
  if (!root) return

  root.innerHTML = `
    <main style="min-height:100vh;display:grid;place-items:center;padding:24px;font-family:system-ui,sans-serif">
      <section role="alert" style="max-width:560px;text-align:center">
        <h1 style="font-size:1.5rem;margin:0 0 12px">LightBridge could not start</h1>
        <p style="color:#6b7280;margin:0 0 20px">Please check your connection and try again.</p>
        <button id="startup-retry" type="button" style="cursor:pointer;padding:10px 16px;border:0;border-radius:8px;background:#2563eb;color:white">Retry</button>
      </section>
    </main>`
  root.querySelector<HTMLButtonElement>('#startup-retry')?.addEventListener('click', () => {
    window.location.reload()
  })
})
