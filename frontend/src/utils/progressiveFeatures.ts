import { shallowRef } from 'vue'
import { isDistributionModeNow } from '@/composables/useDeploymentMode'
import { getFeatureManifest, type BackendFeatureState } from '@/api/features'
import {
  FeatureFlags,
  isFeatureFlagEnabled,
  type FeatureFlagDefinition,
  type RegisteredFeatureFlag,
} from '@/utils/featureFlags'

export type ProgressiveFeatureId =
  | 'channelMonitor'
  | 'availableChannels'
  | 'payment'
  | 'riskControl'
  | 'privacyFilter'
  | 'affiliate'
  | 'announcements'
  | 'redeem'
  | 'promo'
  | 'proxies'
  | 'channelPricing'
  | 'subscriptions'
  | 'opsMonitoring'
  | 'scheduledTests'
  | 'backup'
  | 'moduleRuntime'
  | 'lightbridgeConnect'

export interface ProgressiveFeatureDefinition {
  readonly id: ProgressiveFeatureId
  readonly backendId: string
  readonly label: string
  readonly flag?: RegisteredFeatureFlag
  readonly distributionOnly?: boolean
  readonly routeNames: readonly string[]
  readonly exactPaths?: readonly string[]
  readonly pathPrefixes?: readonly string[]
}

function defineProgressiveFeature(def: ProgressiveFeatureDefinition): ProgressiveFeatureDefinition {
  return def
}

/**
 * Frontend feature registry. `backendId` is the stable contract with the
 * backend Feature Catalog; legacy injected flags are retained only as a safe
 * bootstrap fallback when an older backend does not expose the manifest.
 */
export const ProgressiveFeatures = {
  channelMonitor: defineProgressiveFeature({
    id: 'channelMonitor', backendId: 'channel_monitor', label: 'Channel Monitor', flag: 'channelMonitor',
    routeNames: ['AdminChannelMonitor', 'ChannelStatus'], pathPrefixes: ['/admin/channels/monitor', '/monitor'],
  }),
  availableChannels: defineProgressiveFeature({
    id: 'availableChannels', backendId: 'available_channels', label: 'Available Channels', flag: 'availableChannels',
    routeNames: ['UserAvailableChannels'], pathPrefixes: ['/available-channels'],
  }),
  payment: defineProgressiveFeature({
    id: 'payment', backendId: 'payment', label: 'Payment', flag: 'payment',
    routeNames: ['PurchaseSubscription', 'OrderList', 'PaymentQRCode', 'PaymentResult', 'StripePayment', 'AirwallexPayment', 'StripePopup', 'AdminPaymentDashboard', 'AdminOrders', 'AdminPaymentPlans'],
    pathPrefixes: ['/purchase', '/orders', '/payment', '/admin/orders'],
  }),
  riskControl: defineProgressiveFeature({
    id: 'riskControl', backendId: 'risk_control', label: 'Risk Control', flag: 'riskControl', distributionOnly: true,
    routeNames: ['AdminRiskControl'], pathPrefixes: ['/admin/risk-control'],
  }),
  privacyFilter: defineProgressiveFeature({
    id: 'privacyFilter', backendId: 'privacy_filter', label: 'Privacy Filter', flag: 'privacyFilter',
    routeNames: ['AdminPrivacyFilter'], pathPrefixes: ['/admin/privacy-filter'],
  }),
  affiliate: defineProgressiveFeature({
    id: 'affiliate', backendId: 'affiliate', label: 'Affiliate', flag: 'affiliate',
    routeNames: ['Affiliate', 'AdminAffiliatesRoot', 'AdminAffiliateInvites', 'AdminAffiliateRebates', 'AdminAffiliateTransfers'],
    pathPrefixes: ['/affiliate', '/admin/affiliates'],
  }),
  announcements: defineProgressiveFeature({
    id: 'announcements', backendId: 'announcements', label: 'Announcements', flag: 'announcements', distributionOnly: true,
    routeNames: ['AdminAnnouncements'], pathPrefixes: ['/admin/announcements'],
  }),
  redeem: defineProgressiveFeature({
    id: 'redeem', backendId: 'redeem', label: 'Redeem Codes', flag: 'redeem', distributionOnly: true,
    routeNames: ['AdminRedeem', 'Redeem'], pathPrefixes: ['/admin/redeem', '/redeem'],
  }),
  promo: defineProgressiveFeature({
    id: 'promo', backendId: 'promo', label: 'Promo Codes', flag: 'promo', distributionOnly: true,
    routeNames: ['AdminPromoCodes'], pathPrefixes: ['/admin/promo-codes'],
  }),
  proxies: defineProgressiveFeature({
    id: 'proxies', backendId: 'proxies', label: 'IP Management', flag: 'proxies',
    routeNames: ['AdminProxies', 'AdminProxyModule'], pathPrefixes: ['/admin/proxies', '/admin/proxy'],
  }),
  channelPricing: defineProgressiveFeature({
    id: 'channelPricing', backendId: 'channel_pricing', label: 'Channel Pricing', flag: 'channelPricing',
    routeNames: ['AdminChannelsRoot', 'AdminChannels'], exactPaths: ['/admin/channels'], pathPrefixes: ['/admin/channels/pricing'],
  }),
  subscriptions: defineProgressiveFeature({
    id: 'subscriptions', backendId: 'subscriptions', label: 'Subscriptions', distributionOnly: true,
    routeNames: ['AdminSubscriptions', 'Subscriptions'], pathPrefixes: ['/admin/subscriptions', '/subscriptions'],
  }),
  opsMonitoring: defineProgressiveFeature({
    id: 'opsMonitoring', backendId: 'ops_monitoring', label: 'Ops Monitoring',
    routeNames: ['AdminOps', 'AdminErrorAnalysis'], pathPrefixes: ['/admin/ops', '/admin/error-analysis'],
  }),
  scheduledTests: defineProgressiveFeature({
    id: 'scheduledTests', backendId: 'scheduled_tests', label: 'Scheduled Tests',
    routeNames: [],
  }),
  backup: defineProgressiveFeature({
    id: 'backup', backendId: 'backup', label: 'Backup',
    routeNames: ['AdminBackupSettings'], exactPaths: ['/admin/settings/backup'],
  }),
  moduleRuntime: defineProgressiveFeature({
    id: 'moduleRuntime', backendId: 'module_runtime', label: 'Module Runtime',
    routeNames: [],
  }),
  lightbridgeConnect: defineProgressiveFeature({
    id: 'lightbridgeConnect', backendId: 'lightbridge_connect', label: 'LightBridge Connect', routeNames: [],
  }),
} as const

export const progressiveFeatureList = Object.values(ProgressiveFeatures) as ProgressiveFeatureDefinition[]

const featureManifest = shallowRef<ReadonlyMap<string, BackendFeatureState>>(new Map())
const featureManifestHydrated = shallowRef(false)
let manifestRequest: Promise<void> | null = null

export function backendFeatureState(feature: ProgressiveFeatureDefinition): BackendFeatureState | undefined {
  return featureManifest.value.get(feature.backendId)
}

export function resolveProgressiveFeatureFlag(feature: ProgressiveFeatureDefinition): FeatureFlagDefinition | undefined {
  return feature.flag ? FeatureFlags[feature.flag] : undefined
}

export function isProgressiveFeatureEnabled(feature: ProgressiveFeatureDefinition): boolean {
  const state = backendFeatureState(feature)
  if (state) return state.enabled
  if (feature.distributionOnly && !isDistributionModeNow()) return false
  const flag = resolveProgressiveFeatureFlag(feature)
  return flag ? isFeatureFlagEnabled(flag) : true
}

export function makeProgressiveSidebarFlag(feature: ProgressiveFeatureDefinition): () => boolean {
  return () => isProgressiveFeatureEnabled(feature)
}

export async function hydrateProgressiveFeatureManifest(force = false): Promise<void> {
  if (featureManifestHydrated.value && !force) return
  if (manifestRequest) return manifestRequest

  manifestRequest = (async () => {
    const controller = new AbortController()
    const timer = window.setTimeout(() => controller.abort(), 5000)
    try {
      const states = await getFeatureManifest(controller.signal)
      featureManifest.value = new Map(states.map((state) => [state.id, Object.freeze({ ...state })]))
      featureManifestHydrated.value = true
    } finally {
      window.clearTimeout(timer)
    }
  })()

  try {
    await manifestRequest
  } finally {
    manifestRequest = null
  }
}

export function invalidateProgressiveFeatureManifest(): void {
  featureManifestHydrated.value = false
}

function pathMatchesFeature(path: string, feature: ProgressiveFeatureDefinition): boolean {
  if (feature.exactPaths?.some((exact) => path === exact)) return true
  return feature.pathPrefixes?.some((prefix) => path === prefix || path.startsWith(prefix + '/')) ?? false
}

export function findProgressiveFeatureByPath(path: string): ProgressiveFeatureDefinition | undefined {
  return progressiveFeatureList.find((feature) => pathMatchesFeature(path, feature))
}

export function findDisabledProgressiveFeatureByPath(path: string): ProgressiveFeatureDefinition | undefined {
  return progressiveFeatureList.find((feature) => pathMatchesFeature(path, feature) && !isProgressiveFeatureEnabled(feature))
}

export function isProgressivePathDisabled(path: string): boolean {
  return Boolean(findDisabledProgressiveFeatureByPath(path))
}
