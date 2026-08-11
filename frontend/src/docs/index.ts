import gettingStarted from './content/getting-started.md?raw'
import lightbridgeRouter from './content/lightbridge-router.md?raw'
import release029Preview1 from './content/release-0.2.9-preview.1.md?raw'
import release0210Preview from './content/release-0.2.10-preview.md?raw'
import release0213Preview from './content/release-0.2.13-preview.md?raw'
import release0230 from './content/release-0.2.30.md?raw'
import release0240Preview from './content/release-0.2.40-preview.md?raw'
import release0250 from './content/release-0.2.50.md?raw'
import release0260 from './content/release-0.2.60.md?raw'
import release030Preview from './content/release-0.3.0-preview.md?raw'
import release034 from './content/release-0.3.4.md?raw'
import release0364 from './content/release-0.3.64.md?raw'
import opsPublicPages from './content/ops-public-pages.md?raw'
import opsUserCenter from './content/ops-user-center.md?raw'
import opsUserPayment from './content/ops-user-payment.md?raw'
import opsUserDistribution from './content/ops-user-distribution.md?raw'
import opsAdminCore from './content/ops-admin-core.md?raw'
import opsAdminChannels from './content/ops-admin-channels.md?raw'
import opsAdminOps from './content/ops-admin-ops.md?raw'
import opsAdminSettings from './content/ops-admin-settings.md?raw'
import opsAdminPayment from './content/ops-admin-payment.md?raw'
import opsAdminAffiliate from './content/ops-admin-affiliate.md?raw'
import opsAdminDistribution from './content/ops-admin-distribution.md?raw'

export interface LightBridgeDoc {
  id: string
  title: string
  group: string
  description?: string
  content: string
}

export const lightBridgeDocs: LightBridgeDoc[] = [
  {
    id: 'getting-started',
    title: '文档中心',
    group: '入门',
    description: 'LightBridge 模块与功能文档入口',
    content: gettingStarted,
  },
  {
    id: 'lightbridge-router',
    title: 'LightBridge Router',
    group: '核心能力',
    description: '全协议消息路由、透传模式与混合分组调度说明',
    content: lightbridgeRouter,
  },
  {
    id: 'ops-public-pages',
    title: '公共页面（无需登录）',
    group: '操作文档',
    description: '首页、登录、注册、密码重置、初始化向导等公共页面功能说明',
    content: opsPublicPages,
  },
  {
    id: 'ops-user-center',
    title: '用户中心页面（需登录）',
    group: '操作文档',
    description: '仪表盘、API 密钥、使用记录、个人资料、渠道监控等用户功能说明',
    content: opsUserCenter,
  },
  {
    id: 'ops-user-payment',
    title: '用户支付页面',
    group: '操作文档',
    description: '充值购买、订单管理、QR 码支付、Stripe、Airwallex 等支付功能说明',
    content: opsUserPayment,
  },
  {
    id: 'ops-user-distribution',
    title: '分发模式用户页面',
    group: '操作文档',
    description: '兑换码、我的订阅等分发模式专属用户功能说明',
    content: opsUserDistribution,
  },
  {
    id: 'ops-admin-core',
    title: '管理员 - 核心管理',
    group: '操作文档',
    description: '管理仪表盘、用户管理、分组管理功能说明',
    content: opsAdminCore,
  },
  {
    id: 'ops-admin-channels',
    title: '管理员 - 渠道与账号',
    group: '操作文档',
    description: '渠道定价、渠道监控、账号管理、代理管理功能说明',
    content: opsAdminChannels,
  },
  {
    id: 'ops-admin-ops',
    title: '管理员 - 运维监控',
    group: '操作文档',
    description: '运维仪表盘、错误分析功能说明',
    content: opsAdminOps,
  },
  {
    id: 'ops-admin-settings',
    title: '管理员 - 系统设置',
    group: '操作文档',
    description: '系统设置、模块管理、隐私过滤、版本控制功能说明',
    content: opsAdminSettings,
  },
  {
    id: 'ops-admin-payment',
    title: '管理员 - 支付与订单',
    group: '操作文档',
    description: '支付仪表盘、订单管理、订阅计划管理功能说明',
    content: opsAdminPayment,
  },
  {
    id: 'ops-admin-affiliate',
    title: '管理员 - 联盟推广',
    group: '操作文档',
    description: '邀请记录、返利记录、转账记录管理功能说明',
    content: opsAdminAffiliate,
  },
  {
    id: 'ops-admin-distribution',
    title: '管理员 - 分发模式专属',
    group: '操作文档',
    description: '订阅管理、公告管理、兑换码管理、优惠码管理、风控管理功能说明',
    content: opsAdminDistribution,
  },
  {
    id: 'release-0.3.64',
    title: '0.3.64 版本更新',
    group: '版本更新',
    description: '修复登录后控制台白屏、卡死或浏览器崩溃问题',
    content: release0364,
  },
  {
    id: 'release-0.3.4',
    title: '0.3.4 版本更新',
    group: '版本更新',
    description: 'OpenAI OAuth Tool Calling namespace 断流、instructions 403 与多传输模式兼容修复正式版',
    content: release034,
  },
  {
    id: 'release-0.3.0-preview',
    title: '0.3.0-preview 版本更新',
    group: '版本更新',
    description: 'Grok Build Token context、多协议 Router、渠道限制恢复与 Release 加固预览版',
    content: release030Preview,
  },
  {
    id: 'release-0.2.60',
    title: '0.2.60 版本更新',
    group: '版本更新',
    description: 'OAuth 平台迁移修复、CRS 同步保护与 OpenAI Responses 调度修复正式版',
    content: release0260,
  },
  {
    id: 'release-0.2.50',
    title: '0.2.50 版本更新',
    group: '版本更新',
    description: '高峰倍率、OpenAI WS HTTP Bridge、Router 协议调度修复正式版',
    content: release0250,
  },
  {
    id: 'release-0.2.40-preview',
    title: '0.2.40-preview 版本更新',
    group: '版本更新',
    description: 'Grok 平台接入、账号导入导出增强与用量统计修复预览版',
    content: release0240Preview,
  },
  {
    id: 'release-0.2.30',
    title: '0.2.30 版本更新',
    group: '版本更新',
    description: '模型目录、模块中心与渐进式模块架构正式版',
    content: release0230,
  },
  {
    id: 'release-0.2.13-preview',
    title: '0.2.13 版本更新',
    group: '版本更新',
    description: '路由错误诊断增强、应用内文档中心与前端重构',
    content: release0213Preview,
  },
  {
    id: 'release-0.2.10-preview',
    title: '0.2.10-preview 版本更新',
    group: '版本更新',
    description: '分组调度修复与预览版升级检测修复',
    content: release0210Preview,
  },
  {
    id: 'release-0.2.9-preview.1',
    title: '0.2.9-preview.1 版本更新',
    group: '版本更新',
    description: '全协议 Router 分组调度修复预览版',
    content: release029Preview1,
  },
]

export function findDocById(id: string | null | undefined): LightBridgeDoc {
  return lightBridgeDocs.find((doc) => doc.id === id) ?? lightBridgeDocs[0]
}
