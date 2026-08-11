import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')
const modernCompatPath = resolve(
  dirname(fileURLToPath(import.meta.url)),
  '../../../styles/ui-modern-compat-parts/part-05.css'
)
const modernCompatSource = readFileSync(modernCompatPath, 'utf8')

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar header styles', () => {
  it('keeps the LightBridge brand mark independent from site and theme settings', () => {
    const brandMarkStyle = componentSource.match(/\.lightbridge-brand-mark\s*\{[\s\S]*?\n\}/)?.[0]

    expect(componentSource).toContain('<div class="lightbridge-brand-mark" aria-hidden="true"></div>')
    expect(componentSource).not.toContain(':src="siteLogo')
    expect(brandMarkStyle).toContain('width: 28px !important;')
    expect(brandMarkStyle).toContain('height: 28px !important;')
    expect(brandMarkStyle).toContain('border-radius: 0 !important;')
    expect(brandMarkStyle).toContain('background: #e42313 !important;')
    expect(brandMarkStyle).toContain('box-shadow: none !important;')
  })

  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBlocks = [...styleSource.matchAll(/\.sidebar\s*\{[\s\S]*?\n {2}\}/g)].map(
      (match) => match[0]
    )
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBlocks.length).toBeGreaterThan(0)
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarBlocks.at(-1)).toContain('overflow: visible;')
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})

describe('AppSidebar active item styles', () => {
  it('keeps the selected neutral pill aligned with the navigation column', () => {
    const expandedActiveStyle = modernCompatSource.match(
      /\.sidebar-link-active:not\(\.sidebar-link-collapsed\)\s*\{[\s\S]*?\n\}/
    )?.[0]

    expect(expandedActiveStyle).toContain('width: 100%;')
    expect(expandedActiveStyle).toContain('margin-inline: 0;')
    expect(expandedActiveStyle).toContain('padding-inline: 0.75rem !important;')
  })
})

describe('AppSidebar theme control', () => {
  it('switches theme directly and keeps the accessible target label in sync', () => {
    expect(componentSource).toContain('@click="toggleTheme"')
    expect(componentSource).toContain(':aria-label="isDark ? t(\'nav.lightMode\') : t(\'nav.darkMode\')"')
    expect(componentSource).toContain("setTheme(isDark.value ? 'light' : 'dark')")
    expect(componentSource).toContain("document.documentElement.classList.toggle('dark', isDark.value)")
    expect(componentSource).toContain("localStorage.setItem('theme', theme)")
    expect(componentSource).not.toContain('showThemeDropdown')
  })
})
