import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../../..')
const vueSource = readFileSync(resolve(root, 'src/components/layout/AppSidebar.vue'), 'utf8')
const reactSource = readFileSync(resolve(root, 'src/console/react/ConsoleSidebar.tsx'), 'utf8')

describe('React shadcn sidebar integration', () => {
  it('uses the official sidebar-16 composition', () => {
    for (const primitive of [
      'Card',
      'SidebarProvider',
      'SidebarHeader',
      'SidebarContent',
      'SidebarGroup',
      'SidebarGroupLabel',
      'SidebarGroupContent',
      'SidebarMenu',
      'SidebarMenuItem',
      'SidebarMenuButton',
      'SidebarSeparator',
      'SidebarFooter',
      'DropdownMenu',
    ]) {
      expect(reactSource).toContain(primitive)
    }
  })

  it('keeps the Vue shell responsible for business navigation and removes legacy collapse controls', () => {
    expect(vueSource).toContain('consoleSidebarSections')
    expect(vueSource).toContain('featureFlag')
    expect(vueSource).not.toContain('sidebarCollapsed')
    expect(vueSource).not.toContain('toggleSidebar')
    expect(reactSource).not.toContain('onToggleCollapse')
    expect(reactSource).not.toContain('onToggleTheme')
    expect(reactSource).not.toContain('onLocaleChange')
  })
})
