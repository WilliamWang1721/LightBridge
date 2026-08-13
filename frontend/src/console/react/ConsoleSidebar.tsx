import * as React from 'react'
import {
  Bell,
  ChartNoAxesColumn,
  CircleDashed,
  CircleHelp,
  CircleDollarSign,
  ChevronRight,
  ChevronsUpDown,
  CreditCard,
  Database,
  FileText,
  Folder,
  Gift,
  Globe2,
  GitBranch,
  KeyRound,
  Layers3,
  LayoutDashboard,
  ListOrdered,
  LogOut,
  MessageSquare,
  Radio,
  Send,
  Server,
  Settings,
  ShieldCheck,
  Sparkles,
  Tag,
  Ticket,
  TriangleAlert,
  UserRound,
  UsersRound,
} from 'lucide-react'
import type { User } from '@/types'
import type {
  ConsoleSidebarItem,
  ConsoleSidebarProps,
  ConsoleSidebarSection,
} from '../sidebar/contract'
import { Card } from './ui/card'
import { Avatar, AvatarFallback, AvatarImage } from './ui/avatar'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from './ui/collapsible'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from './ui/dropdown-menu'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuAction,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  SidebarProvider,
  SidebarSeparator,
  useSidebar,
} from './ui/sidebar'

const iconMap = {
  dashboard: LayoutDashboard,
  key: KeyRound,
  chart: ChartNoAxesColumn,
  gift: Gift,
  user: UserRound,
  users: UsersRound,
  folder: Folder,
  channel: Layers3,
  database: Database,
  creditCard: CreditCard,
  recharge: CircleDollarSign,
  globe: Globe2,
  server: Server,
  bell: Bell,
  ticket: Ticket,
  settings: Settings,
  order: FileText,
  orderList: ListOrdered,
  signal: Radio,
  send: Send,
  shield: ShieldCheck,
  alert: TriangleAlert,
  tag: Tag,
  feedback: MessageSquare,
  sparkles: Sparkles,
} as const

function Icon({
  name,
}: {
  name?: keyof typeof iconMap
  svg?: string
}) {
  const IconComponent = name ? iconMap[name] : undefined
  if (IconComponent) return <IconComponent className="size-4" aria-hidden="true" />
  return <CircleDashed className="size-4" aria-hidden="true" />
}

function isActive(path: string, currentPath: string) {
  return currentPath === path || currentPath.startsWith(`${path}/`)
}

function hasActiveChild(item: ConsoleSidebarItem, currentPath: string) {
  return item.children?.some((child) => isActive(child.path, currentPath)) ?? false
}

function initials(user?: User | null) {
  const value = user?.username || user?.email?.split('@')[0] || ''
  return value.slice(0, 2).toUpperCase()
}

function navigate(event: React.MouseEvent<HTMLAnchorElement>, path: string, onNavigate: (path: string) => void) {
  event.preventDefault()
  onNavigate(path)
}

function NavItem({ item, props }: { item: ConsoleSidebarItem; props: ConsoleSidebarProps }) {
  const hasChildren = Boolean(item.children?.length)
  const activeChild = hasActiveChild(item, props.currentPath)
  const active = isActive(item.path, props.currentPath) || activeChild
  const expanded = props.expandedGroupPaths.includes(item.path) || activeChild

  if (hasChildren) {
    return (
      <Collapsible
        asChild
        open={expanded}
        onOpenChange={(open) => {
          if (open !== expanded) props.onToggleGroup(item.path)
        }}
        className="group/collapsible"
      >
        <SidebarMenuItem>
          <SidebarMenuButton
            asChild
            isActive={active && !expanded}
            tooltip={item.label}
            className="h-9 rounded-full"
            id={item.id}
            data-tour={item.dataTour}
          >
            <a
              href={item.expandOnly ? '#' : item.path}
              aria-current={!item.expandOnly && active ? 'page' : undefined}
              onClick={(event) => {
                if (item.expandOnly) {
                  event.preventDefault()
                  props.onToggleGroup(item.path)
                } else {
                  navigate(event, item.path, props.onNavigate)
                }
              }}
            >
              <Icon name={item.iconName as keyof typeof iconMap | undefined} svg={item.iconSvg} />
              <span>{item.label}</span>
            </a>
          </SidebarMenuButton>
          <CollapsibleTrigger asChild>
            <SidebarMenuAction className="data-[state=open]:rotate-90">
              <ChevronRight />
              <span className="sr-only">{item.label}</span>
            </SidebarMenuAction>
          </CollapsibleTrigger>
          <CollapsibleContent>
            <SidebarMenuSub className="gap-1.5 py-1">
              {item.children?.map((child) => {
                const childActive = isActive(child.path, props.currentPath)
                return (
                  <SidebarMenuSubItem key={child.path}>
                    <SidebarMenuSubButton asChild isActive={childActive} className="h-8 rounded-full">
                      <a
                        href={child.path}
                        id={child.id}
                        data-tour={child.dataTour}
                        aria-current={childActive ? 'page' : undefined}
                        onClick={(event) => navigate(event, child.path, props.onNavigate)}
                      >
                        <Icon name={child.iconName as keyof typeof iconMap | undefined} svg={child.iconSvg} />
                        <span>{child.label}</span>
                      </a>
                    </SidebarMenuSubButton>
                  </SidebarMenuSubItem>
                )
              })}
            </SidebarMenuSub>
          </CollapsibleContent>
        </SidebarMenuItem>
      </Collapsible>
    )
  }

  const activeItem = isActive(item.path, props.currentPath)
  return (
    <SidebarMenuItem>
      <SidebarMenuButton asChild isActive={activeItem} tooltip={item.label} className="h-9 rounded-full">
        <a
          href={item.path}
          id={item.id}
          data-tour={item.dataTour}
          aria-current={activeItem ? 'page' : undefined}
          onClick={(event) => navigate(event, item.path, props.onNavigate)}
        >
          <Icon name={item.iconName as keyof typeof iconMap | undefined} svg={item.iconSvg} />
          <span>{item.label}</span>
        </a>
      </SidebarMenuButton>
    </SidebarMenuItem>
  )
}

function Section({ section, index, props }: { section: ConsoleSidebarSection; index: number; props: ConsoleSidebarProps }) {
  return (
    <React.Fragment key={section.key}>
      <SidebarGroup>
        <SidebarGroupLabel>{section.title || props.labels.mainNavigation}</SidebarGroupLabel>
        <SidebarGroupContent>
          <SidebarMenu className="gap-1.5">
            {section.items.map((item) => <NavItem key={item.path} item={item} props={props} />)}
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
      {index < props.sections.length - 1 && <SidebarSeparator />}
    </React.Fragment>
  )
}

function UserNavigation({ props }: { props: ConsoleSidebarProps }) {
  const { isMobile } = useSidebar()
  const user = props.user
  if (!user) return null

  const name = user.username || user.email.split('@')[0]
  const selectPath = (path: string) => () => props.onNavigate(path)

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton size="lg" className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground">
              <Avatar className="h-8 w-8 rounded-lg">
                <AvatarImage src={user.avatar_url || undefined} alt={name} />
                <AvatarFallback className="rounded-lg">{initials(user)}</AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">{name}</span>
                <span className="truncate text-xs">{user.email}</span>
              </div>
              <ChevronsUpDown className="ml-auto size-4" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
            side={isMobile ? 'bottom' : 'right'}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuLabel className="p-0 font-normal">
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                <Avatar className="h-8 w-8 rounded-lg">
                  <AvatarImage src={user.avatar_url || undefined} alt={name} />
                  <AvatarFallback className="rounded-lg">{initials(user)}</AvatarFallback>
                </Avatar>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-semibold">{name}</span>
                  <span className="truncate text-xs">{user.email}</span>
                </div>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem onSelect={selectPath('/profile')}><UserRound />{props.labels.profile}</DropdownMenuItem>
              <DropdownMenuItem onSelect={selectPath('/keys')}><KeyRound />{props.labels.apiKeys}</DropdownMenuItem>
              <DropdownMenuItem onSelect={selectPath('/admin/settings')}><Settings />{props.labels.settings}</DropdownMenuItem>
            </DropdownMenuGroup>
            {props.isAdmin && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem onSelect={() => window.open('https://github.com/WilliamWang1721/LightBridge', '_blank', 'noopener,noreferrer')}>
                  <GitBranch />{props.labels.github}
                </DropdownMenuItem>
              </>
            )}
            {(props.contactInfo || props.showOnboardingButton) && <DropdownMenuSeparator />}
            {props.contactInfo && (
              <DropdownMenuLabel className="flex items-center gap-2 text-xs font-normal text-muted-foreground">
                <CircleHelp className="size-4" />
                <span>{props.labels.contactSupport}: {props.contactInfo}</span>
              </DropdownMenuLabel>
            )}
            {props.showOnboardingButton && (
              <DropdownMenuItem onSelect={props.onReplayGuide}>
                <CircleHelp />{props.labels.restartTour}
              </DropdownMenuItem>
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => void props.onLogout()}>
              <LogOut />{props.labels.logout}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}

export default function ConsoleSidebar(props: ConsoleSidebarProps) {
  return (
    <>
      <Card className="sidebar-nav-card min-h-0 w-full flex-1 overflow-hidden py-0">
        <SidebarProvider open={true} onOpenChange={() => undefined} className="h-full min-h-0 w-full">
          <Sidebar collapsible="none" className="w-full bg-transparent" aria-label={props.labels.mainNavigation}>
            <SidebarHeader>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton size="lg" asChild>
                    <a href="/admin/dashboard" onClick={(event) => navigate(event, '/admin/dashboard', props.onNavigate)}>
                      <div className="flex aspect-square size-9 items-center justify-center overflow-hidden rounded-xl bg-sidebar-primary text-sidebar-primary-foreground shadow-glow">
                        <img src={props.siteLogo || '/logo.png'} alt="" className="h-full w-full object-contain" />
                      </div>
                      <div className="grid min-w-0 flex-1 text-left text-sm leading-tight">
                        <span className="truncate font-semibold">{props.siteName}</span>
                        <span className="truncate text-xs">{props.siteVersion || 'LightBridge'}</span>
                      </div>
                    </a>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarHeader>
            <SidebarContent className="gap-0 overflow-x-hidden">
              {props.sections.map((section, index) => <Section key={section.key} section={section} index={index} props={props} />)}
            </SidebarContent>
            <SidebarFooter>
              <UserNavigation props={props} />
            </SidebarFooter>
          </Sidebar>
        </SidebarProvider>
      </Card>
      {props.mobileOpen && (
        <button
          type="button"
          className="fixed inset-0 z-30 border-0 bg-black/50 p-0 lg:hidden"
          aria-label={props.labels.closeNavigation}
          onClick={props.onCloseMobile}
        />
      )}
    </>
  )
}
