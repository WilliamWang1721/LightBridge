import type { User } from '@/types'

export interface ConsoleSidebarItem {
  path: string
  label: string
  iconName?: string
  iconSvg?: string
  id?: string
  dataTour?: string
  children?: ConsoleSidebarItem[]
  expandOnly?: boolean
}

export interface ConsoleSidebarSection {
  key: string
  title?: string
  items: ConsoleSidebarItem[]
}

export interface ConsoleSidebarLabels {
  mainNavigation: string
  closeNavigation: string
  profile: string
  apiKeys: string
  settings: string
  github: string
  contactSupport: string
  restartTour: string
  logout: string
}

export interface ConsoleSidebarCallbacks {
  onNavigate: (path: string) => void
  onToggleGroup: (path: string) => void
  onCloseMobile: () => void
  onLogout: () => void | Promise<void>
  onReplayGuide?: () => void
}

export interface ConsoleSidebarProps extends ConsoleSidebarCallbacks {
  sections: ConsoleSidebarSection[]
  currentPath: string
  expandedGroupPaths: readonly string[]
  labels: ConsoleSidebarLabels
  mobileOpen: boolean
  user?: User | null
  isAdmin: boolean
  contactInfo?: string | null
  showOnboardingButton: boolean
  siteName: string
  siteLogo?: string | null
  siteVersion?: string
}
