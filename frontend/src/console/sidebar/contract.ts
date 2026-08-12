export interface ConsoleSidebarItem {
  path: string
  label: string
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

export interface ConsoleSidebarLocaleOption {
  value: string
  label: string
}

export interface ConsoleSidebarLabels {
  mainNavigation: string
  lightMode: string
  darkMode: string
  collapse: string
  expand: string
  closeNavigation: string
  locale: string
}

export interface ConsoleSidebarCallbacks {
  onNavigate: (path: string) => void
  onToggleGroup: (path: string) => void
  onToggleTheme: () => void
  onLocaleChange: (locale: string) => void
  onToggleCollapse: () => void
  onCloseMobile: () => void
}

export interface ConsoleSidebarProps extends ConsoleSidebarCallbacks {
  sections: ConsoleSidebarSection[]
  currentPath: string
  expandedGroupPaths: readonly string[]
  labels: ConsoleSidebarLabels
  localeOptions: ConsoleSidebarLocaleOption[]
  currentLocale: string
  isDark: boolean
  collapsed: boolean
  mobileOpen: boolean
}
