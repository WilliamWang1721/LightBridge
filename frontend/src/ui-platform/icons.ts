import type { Component } from 'vue'
import {
  Activity,
  Bell,
  ChartNoAxesCombined,
  CircleHelp,
  CreditCard,
  Database,
  House,
  KeyRound,
  Menu,
  Package,
  RefreshCw,
  Search,
  Settings,
  ShieldCheck,
  User,
  X,
} from '@lucide/vue'

export type AppIconName =
  | 'activity'
  | 'bell'
  | 'chart'
  | 'close'
  | 'credit-card'
  | 'database'
  | 'help'
  | 'home'
  | 'key'
  | 'menu'
  | 'package'
  | 'refresh'
  | 'search'
  | 'settings'
  | 'shield'
  | 'user'

export type IconProvider = Record<AppIconName, Component>

const lucideProvider: IconProvider = {
  activity: Activity,
  bell: Bell,
  chart: ChartNoAxesCombined,
  close: X,
  'credit-card': CreditCard,
  database: Database,
  help: CircleHelp,
  home: House,
  key: KeyRound,
  menu: Menu,
  package: Package,
  refresh: RefreshCw,
  search: Search,
  settings: Settings,
  shield: ShieldCheck,
  user: User,
}

const providers: Record<string, IconProvider> = {
  lucide: lucideProvider,
}

export function resolveAppIcon(name: AppIconName, provider = 'lucide'): Component {
  return providers[provider]?.[name] || lucideProvider[name]
}

export function registerIconProvider(id: string, provider: IconProvider) {
  if (!id.trim()) throw new Error('Icon provider id is required')
  providers[id] = provider
}
