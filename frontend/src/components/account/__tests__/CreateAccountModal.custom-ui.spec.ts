import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const template = readFileSync(
  resolve(process.cwd(), 'src/components/account/templates/CreateAccountModal.template.html'),
  'utf-8'
)

describe('CreateAccountModal custom provider UI', () => {
  it('renders one required Custom API key and excludes the generic API key branch', () => {
    expect(template).toContain('data-testid="custom-api-key"')
    expect(template).toContain(`v-else-if="form.platform !== 'custom'"`)
    expect(template.match(/v-model="form\.customApiKey"/g)).toHaveLength(1)
    expect(template.match(/v-model="apiKeyValue"/g)).toHaveLength(1)
  })

  it('separates connection and routing parameters into clear sections', () => {
    expect(template).toContain('data-testid="custom-connection-settings"')
    expect(template).toContain('data-testid="custom-routing-settings"')
  })
})
