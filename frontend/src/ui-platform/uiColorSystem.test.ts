import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const platform = readFileSync(resolve(process.cwd(), 'src/styles/ui-platform.css'), 'utf8')
const semanticDefaults = readFileSync(
  resolve(process.cwd(), 'src/styles/ui-semantic-defaults.css'),
  'utf8',
)
const legacyBridge = readFileSync(
  resolve(process.cwd(), 'src/styles/ui-modern-tailwind-bridge-parts/part-03.css'),
  'utf8',
)

function controlledHues(source: string): number[] {
  return [...source.matchAll(
    /--(?:primary|accent|accent-foreground|ring|chart-[1-5]):\s*([\d.]+)/g,
  )].map((match) => Number(match[1]))
}

describe('Luma color system', () => {
  it('keeps semantic actions and charts out of the green hue range', () => {
    const hues = [...controlledHues(platform), ...controlledHues(semanticDefaults)]

    expect(hues.length).toBeGreaterThan(20)
    expect(hues.every((hue) => hue < 90 || hue >= 170)).toBe(true)
    expect(platform).toContain('--chart-1: 221 83% 53%;')
    expect(platform).toContain('--chart-2: 271 81% 56%;')
    expect(platform).toContain('--chart-3: 38 92% 50%;')
    expect(platform).toContain('--chart-4: 0 84% 60%;')
    expect(platform).toContain('--chart-5: 190 90% 44%;')
  })

  it('maps legacy green utility families onto semantic Luma tokens', () => {
    for (const family of ['text', 'bg', 'border', 'fill', 'stroke']) {
      expect(legacyBridge).toContain(`[class^='${family}-green-']`)
      expect(legacyBridge).toContain(`[class^='${family}-emerald-']`)
      expect(legacyBridge).toContain(`[class^='${family}-lime-']`)
    }
    expect(legacyBridge).toContain('color: hsl(var(--primary)) !important;')
    expect(legacyBridge).toContain('background-color: hsl(var(--primary) / 0.12) !important;')
  })
})
