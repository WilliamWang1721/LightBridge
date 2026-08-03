import type { RouteRecordRaw } from 'vue-router'
import { ProgressiveFeatures, type ProgressiveFeatureDefinition } from '@/utils/progressiveFeatures'

export interface ProgressiveRouteGroup {
  readonly feature: ProgressiveFeatureDefinition
  readonly routes: readonly RouteRecordRaw[]
}

function defineProgressiveRouteGroup(group: ProgressiveRouteGroup): ProgressiveRouteGroup {
  return group
}

export const progressiveRouteGroups = [
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.affiliate,
    routes: [
      {
        path: '/affiliate',
        name: 'Affiliate',
        component: () => import('@/views/user/AffiliateView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: false,
          title: 'Affiliate',
          titleKey: 'affiliate.title',
          descriptionKey: 'affiliate.description',
        },
      },
      {
        path: '/admin/affiliates',
        name: 'AdminAffiliatesRoot',
        redirect: '/admin/affiliates/invites',
      },
      {
        path: '/admin/affiliates/invites',
        name: 'AdminAffiliateInvites',
        component: () => import('@/views/admin/affiliates/AdminAffiliateInvitesView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Affiliate Invite Records',
          titleKey: 'nav.affiliateInviteRecords',
          descriptionKey: 'admin.affiliates.invitesDescription',
        },
      },
      {
        path: '/admin/affiliates/rebates',
        name: 'AdminAffiliateRebates',
        component: () => import('@/views/admin/affiliates/AdminAffiliateRebatesView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Affiliate Rebate Records',
          titleKey: 'nav.affiliateRebateRecords',
          descriptionKey: 'admin.affiliates.rebatesDescription',
        },
      },
      {
        path: '/admin/affiliates/transfers',
        name: 'AdminAffiliateTransfers',
        component: () => import('@/views/admin/affiliates/AdminAffiliateTransfersView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Affiliate Transfer Records',
          titleKey: 'nav.affiliateTransferRecords',
          descriptionKey: 'admin.affiliates.transfersDescription',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.availableChannels,
    routes: [
      {
        path: '/available-channels',
        name: 'UserAvailableChannels',
        component: () => import('@/views/user/AvailableChannelsView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: false,
          title: 'Available Channels',
          titleKey: 'availableChannels.title',
          descriptionKey: 'availableChannels.description',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.payment,
    routes: [
      {
        path: '/purchase',
        name: 'PurchaseSubscription',
        component: () => import('@/views/user/PaymentView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: false,
          title: 'Purchase Subscription',
          titleKey: 'nav.buySubscription',
          descriptionKey: 'purchase.description',
        },
      },
      {
        path: '/orders',
        name: 'OrderList',
        component: () => import('@/views/user/UserOrdersView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: false,
          title: 'My Orders',
          titleKey: 'nav.myOrders',
        },
      },
      {
        path: '/payment/qrcode',
        name: 'PaymentQRCode',
        component: () => import('@/views/user/PaymentQRCodeView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: false,
          title: 'Payment',
          titleKey: 'payment.qr.scanToPay',
        },
      },
      {
        path: '/payment/result',
        name: 'PaymentResult',
        component: () => import('@/views/user/PaymentResultView.vue'),
        meta: {
          requiresAuth: false,
          requiresAdmin: false,
          title: 'Payment Result',
          titleKey: 'payment.result.title',
        },
      },
      {
        path: '/payment/stripe',
        name: 'StripePayment',
        component: () => import('@/views/user/StripePaymentView.vue'),
        meta: {
          requiresAuth: false,
          requiresAdmin: false,
          title: 'Stripe Payment',
          titleKey: 'payment.stripePay',
        },
      },
      {
        path: '/payment/airwallex',
        name: 'AirwallexPayment',
        component: () => import('@/views/user/AirwallexPaymentView.vue'),
        meta: {
          requiresAuth: false,
          requiresAdmin: false,
          title: 'Airwallex Payment',
          titleKey: 'payment.airwallexPay',
        },
      },
      {
        path: '/payment/stripe-popup',
        name: 'StripePopup',
        component: () => import('@/views/user/StripePopupView.vue'),
        meta: {
          requiresAuth: false,
          requiresAdmin: false,
          title: 'Payment',
        },
      },
      {
        path: '/admin/orders/dashboard',
        name: 'AdminPaymentDashboard',
        component: () => import('@/views/admin/orders/AdminPaymentDashboardView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Payment Dashboard',
          titleKey: 'nav.paymentDashboard',
        },
      },
      {
        path: '/admin/orders',
        name: 'AdminOrders',
        component: () => import('@/views/admin/orders/AdminOrdersView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Order Management',
          titleKey: 'nav.orderManagement',
        },
      },
      {
        path: '/admin/orders/plans',
        name: 'AdminPaymentPlans',
        component: () => import('@/views/admin/orders/AdminPaymentPlansView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Subscription Plans',
          titleKey: 'nav.paymentPlans',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.channelPricing,
    routes: [
      {
        path: '/admin/channels',
        name: 'AdminChannelsRoot',
        redirect: '/admin/channels/pricing',
      },
      {
        path: '/admin/channels/pricing',
        name: 'AdminChannels',
        component: () => import('@/views/admin/ChannelsView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Channel Management',
          titleKey: 'admin.channels.title',
          descriptionKey: 'admin.channels.description',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.channelMonitor,
    routes: [
      {
        path: '/admin/channels/monitor',
        name: 'AdminChannelMonitor',
        component: () => import('@/views/admin/ChannelMonitorView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Channel Monitor',
          titleKey: 'admin.channelMonitor.title',
          descriptionKey: 'admin.channelMonitor.description',
        },
      },
      {
        path: '/monitor',
        name: 'ChannelStatus',
        component: () => import('@/views/user/ChannelStatusView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: false,
          title: 'Channel Status',
          titleKey: 'nav.channelStatus',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.proxies,
    routes: [
      {
        path: '/admin/proxies',
        name: 'AdminProxies',
        component: () => import('@/views/admin/ProxiesView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Proxy Management',
          titleKey: 'admin.proxies.title',
          descriptionKey: 'admin.proxies.description',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.privacyFilter,
    routes: [
      {
        path: '/admin/privacy-filter',
        name: 'AdminPrivacyFilter',
        component: () => import('@/views/admin/PrivacyFilterView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Privacy Filter',
          titleKey: 'admin.privacyFilter.title',
          descriptionKey: 'admin.privacyFilter.description',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.subscriptions,
    routes: [
      {
        path: '/subscriptions',
        name: 'Subscriptions',
        component: () => import('@/views/user/SubscriptionsView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: false,
          title: 'My Subscriptions',
          titleKey: 'userSubscriptions.title',
          descriptionKey: 'userSubscriptions.description',
        },
      },
      {
        path: '/admin/subscriptions',
        name: 'AdminSubscriptions',
        component: () => import('@/views/admin/SubscriptionsView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Subscription Management',
          titleKey: 'admin.subscriptions.title',
          descriptionKey: 'admin.subscriptions.description',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.announcements,
    routes: [
      {
        path: '/admin/announcements',
        name: 'AdminAnnouncements',
        component: () => import('@/views/admin/AnnouncementsView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Announcements',
          titleKey: 'admin.announcements.title',
          descriptionKey: 'admin.announcements.description',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.redeem,
    routes: [
      {
        path: '/redeem',
        name: 'Redeem',
        component: () => import('@/views/user/RedeemView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: false,
          title: 'Redeem Code',
          titleKey: 'redeem.title',
          descriptionKey: 'redeem.description',
        },
      },
      {
        path: '/admin/redeem',
        name: 'AdminRedeem',
        component: () => import('@/views/admin/RedeemView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Redeem Code Management',
          titleKey: 'admin.redeem.title',
          descriptionKey: 'admin.redeem.description',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.promo,
    routes: [
      {
        path: '/admin/promo-codes',
        name: 'AdminPromoCodes',
        component: () => import('@/views/admin/PromoCodesView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Promo Code Management',
          titleKey: 'admin.promo.title',
          descriptionKey: 'admin.promo.description',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.riskControl,
    routes: [
      {
        path: '/admin/risk-control',
        name: 'AdminRiskControl',
        component: () => import('@/views/admin/RiskControlView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Risk Control',
          titleKey: 'admin.riskControl.title',
          descriptionKey: 'admin.riskControl.description',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.opsMonitoring,
    routes: [
      {
        path: '/admin/ops',
        name: 'AdminOps',
        component: () => import('@/views/admin/ops/OpsDashboard.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Ops Monitoring',
          titleKey: 'admin.ops.title',
          descriptionKey: 'admin.ops.description',
        },
      },
      {
        path: '/admin/error-analysis',
        name: 'AdminErrorAnalysis',
        component: () => import('@/views/admin/ops/ErrorAnalysisView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Error Analysis',
          titleKey: 'admin.ops.errorAnalysis.title',
          descriptionKey: 'admin.ops.errorAnalysis.description',
        },
      },
    ],
  }),
  defineProgressiveRouteGroup({
    feature: ProgressiveFeatures.proxies,
    routes: [
      {
        path: '/admin/proxy',
        name: 'AdminProxyModule',
        component: () => import('@/views/admin/ProxyModuleView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Proxy Runtime',
        },
      },
    ],
  }),
] as const satisfies readonly ProgressiveRouteGroup[]
