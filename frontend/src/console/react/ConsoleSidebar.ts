import { type ReactNode } from 'react'
import { sanitizeSvg } from '@/utils/sanitize'
import type {
  ConsoleSidebarItem,
  ConsoleSidebarProps,
  ConsoleSidebarSection,
} from '../sidebar/contract'
import { createShadcnElement as h } from './ui/createElement'

function Icon({ svg, className = 'h-5 w-5' }: { svg?: string; className?: string }) {
  if (!svg) return h('span', { className: `${className} shrink-0`, 'aria-hidden': 'true' })

  return h('span', {
    className: `${className} sidebar-svg-icon shrink-0`,
    'aria-hidden': 'true',
    dangerouslySetInnerHTML: { __html: sanitizeSvg(svg) },
  })
}

function Label({ children, collapsed, className = '' }: { children?: ReactNode; collapsed: boolean; className?: string }): ReactNode {
  return h('span', {
    className: `sidebar-label min-w-0 overflow-hidden text-ellipsis whitespace-nowrap transition-[max-width,opacity,transform] duration-200 ${collapsed ? 'pointer-events-none max-w-0 -translate-x-1 opacity-0' : 'max-w-48'} ${className}`.trim(),
    'aria-hidden': collapsed ? 'true' : undefined,
  }, children)
}

function isActive(path: string, currentPath: string) {
  return currentPath === path || currentPath.startsWith(`${path}/`)
}

function hasActiveChild(item: ConsoleSidebarItem, currentPath: string) {
  return item.children?.some((child) => isActive(child.path, currentPath)) ?? false
}

function NavItem({
  item,
  currentPath,
  expandedGroupPaths,
  collapsed,
  onNavigate,
  onToggleGroup,
}: Pick<ConsoleSidebarProps, 'currentPath' | 'expandedGroupPaths' | 'collapsed' | 'onNavigate' | 'onToggleGroup'> & { item: ConsoleSidebarItem }): ReactNode {
  const hasChildren = Boolean(item.children?.length)
  const activeChild = hasActiveChild(item, currentPath)
  const active = isActive(item.path, currentPath) || activeChild
  const expanded = expandedGroupPaths.includes(item.path) || activeChild
  const commonProps = {
    title: collapsed ? item.label : undefined,
    id: item.id,
    'data-tour': item.dataTour,
  }

  if (hasChildren) {
    return h('li', { key: item.path }, [
      h('button', {
        key: 'button',
        type: 'button',
        className: `sidebar-link sidebar-nav-item mb-1 w-full ${active && !expanded ? 'sidebar-link-active' : ''} ${collapsed ? 'sidebar-link-collapsed' : ''}`,
        ...commonProps,
        'aria-expanded': collapsed ? undefined : expanded,
        onClick: () => {
          if (collapsed && item.children?.[0]) onNavigate(item.children[0].path)
          else onToggleGroup(item.path)
        },
      }, [
        h(Icon, { key: 'icon', svg: item.iconSvg }),
        h(Label, { key: 'label', collapsed, className: 'sidebar-label-flex flex flex-1 items-center justify-between gap-2' }, [
          h('span', { key: 'text', className: 'min-w-0 truncate' }, item.label),
          h(Icon, { key: 'chevron', svg: '<svg fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5"/></svg>', className: `h-4 w-4 shrink-0 transition-transform duration-200 ${expanded ? 'rotate-180' : ''}` }),
        ]),
      ]),
      !collapsed && expanded
        ? h('ul', { key: 'children', className: 'sidebar-subnav mb-1 ml-4 border-l border-gray-200 pl-2 dark:border-dark-600' }, item.children?.map((child) => h(NavItem, {
          key: child.path,
          item: child,
          currentPath,
          expandedGroupPaths,
          collapsed: false,
          onNavigate,
          onToggleGroup,
        })))
        : null,
    ])
  }

  return h('li', { key: item.path }, h('button', {
    type: 'button',
    className: `sidebar-link sidebar-nav-item mb-1 ${active ? 'sidebar-link-active' : ''} ${collapsed ? 'sidebar-link-collapsed' : ''}`,
    ...commonProps,
    'aria-current': active ? 'page' : undefined,
    onClick: () => onNavigate(item.path),
  }, [
    h(Icon, { key: 'icon', svg: item.iconSvg }),
    h(Label, { key: 'label', collapsed }, item.label),
  ]))
}

function Section({ section, props }: { section: ConsoleSidebarSection; props: ConsoleSidebarProps }): ReactNode {
  return h('section', { key: section.key, className: 'sidebar-section' }, [
    section.title
      ? h('h2', {
        key: 'title',
        className: `sidebar-section-title mb-2 px-3 text-xs font-semibold ${props.collapsed ? 'my-3 border-t border-gray-200 px-0 dark:border-dark-600' : ''}`,
        'aria-hidden': props.collapsed ? 'true' : undefined,
      }, props.collapsed ? null : section.title)
      : null,
    h('ul', { key: 'items', className: 'flex w-full min-w-0 flex-col gap-1' }, section.items.map((item) => h(NavItem, {
      key: item.path,
      item,
      currentPath: props.currentPath,
      expandedGroupPaths: props.expandedGroupPaths,
      collapsed: props.collapsed,
      onNavigate: props.onNavigate,
      onToggleGroup: props.onToggleGroup,
    }))),
  ])
}

export default function ConsoleSidebar(props: ConsoleSidebarProps) {
  const themeLabel = props.isDark ? props.labels.lightMode : props.labels.darkMode
  const themeIcon = props.isDark
    ? '<svg fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 3v2.25m6.364.386-1.591 1.591M21 12h-2.25m-.386 6.364-1.591-1.591M12 18.75V21m-4.773-4.227-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0Z"/></svg>'
    : '<svg fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M21.752 15.002A9.718 9.718 0 0 1 18 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 0 0 3 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 0 0 9.002-5.998Z"/></svg>'
  const collapseIcon = props.collapsed
    ? '<svg fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="m5.25 4.5 7.5 7.5-7.5 7.5m6-15 7.5 7.5-7.5 7.5"/></svg>'
    : '<svg fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="m18.75 4.5-7.5 7.5 7.5 7.5m-6-15L5.25 12l7.5 7.5"/></svg>'

  return h('div', { className: 'contents' }, [
    h('nav', { key: 'nav', className: 'sidebar-nav scrollbar-hide', 'aria-label': props.labels.mainNavigation }, props.sections.map((section) => h(Section, { key: section.key, section, props }))),
    h('footer', { key: 'footer', className: 'sidebar-footer mt-auto shrink-0 border-t border-gray-100 p-3 dark:border-dark-800' }, [
        !props.collapsed ? h('div', { key: 'controls', className: 'mb-2 grid grid-cols-2 gap-2' }, [
          h('button', { key: 'theme', type: 'button', className: 'sidebar-link sidebar-footer-action w-full', onClick: props.onToggleTheme, 'aria-label': themeLabel }, [
            h(Icon, { key: 'icon', svg: themeIcon }),
            h(Label, { key: 'label', collapsed: false, className: 'sidebar-footer-label' }, themeLabel),
          ]),
          h('label', { key: 'locale-label', className: 'sr-only', htmlFor: 'console-sidebar-locale' }, props.labels.locale),
          h('select', { key: 'locale', id: 'console-sidebar-locale', className: 'sidebar-link h-10 min-h-10 w-full cursor-pointer border-0 bg-transparent', value: props.currentLocale, onChange: (event: { currentTarget: { value: string } }) => props.onLocaleChange(event.currentTarget.value), 'aria-label': props.labels.locale }, props.localeOptions.map((option) => h('option', { key: option.value, value: option.value }, option.label))),
        ]) : null,
        h('button', { key: 'collapse', type: 'button', className: `sidebar-link w-full ${props.collapsed ? 'sidebar-link-collapsed' : ''}`, onClick: props.onToggleCollapse, title: props.collapsed ? props.labels.expand : props.labels.collapse, 'aria-label': props.collapsed ? props.labels.expand : props.labels.collapse }, [
          h(Icon, { key: 'icon', svg: collapseIcon }),
          h(Label, { key: 'label', collapsed: props.collapsed, className: 'sidebar-footer-label' }, props.collapsed ? props.labels.expand : props.labels.collapse),
        ]),
      ]),
    props.mobileOpen ? h('button', { key: 'overlay', type: 'button', className: 'fixed inset-0 z-30 border-0 bg-black/50 p-0 lg:hidden', 'aria-label': props.labels.closeNavigation, onClick: props.onCloseMobile }) : null,
  ])
}
