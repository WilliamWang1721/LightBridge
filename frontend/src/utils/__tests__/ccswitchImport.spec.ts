import { describe, expect, it } from 'vitest'
import {
  buildCcSwitchImportDeeplink,
  resolveCcSwitchImportConfig
} from '@/utils/ccswitchImport'

function paramsFromDeeplink(deeplink: string): URLSearchParams {
  const query = deeplink.split('?')[1] || ''
  return new URLSearchParams(query)
}

describe('ccswitchImport utils', () => {
  const baseInput = {
    baseUrl: 'https://api.example.com',
    providerName: 'LightBridge',
    apiKey: 'sk-test',
    usageScript: 'return true'
  }

  it('uses the operator-selected endpoint and model for Codex imports', () => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        endpoint: 'https://codex.example.com/v1',
        model: 'gpt-custom-latest',
        availableIngressProtocols: ['openai_responses'],
        clientType: 'codex'
      })
    )

    expect(params.get('resource')).toBe('provider')
    expect(params.get('app')).toBe('codex')
    expect(params.get('endpoint')).toBe('https://codex.example.com/v1')
    expect(params.get('model')).toBe('gpt-custom-latest')
    expect(atob(params.get('usageScript') || '')).toBe(baseInput.usageScript)
  })

  it('does not inject a hard-coded Codex model when the operator leaves it blank', () => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        availableIngressProtocols: ['openai_responses'],
        clientType: 'codex',
        model: '  '
      })
    )

    expect(params.get('app')).toBe('codex')
    expect(params.has('model')).toBe(false)
  })

  it.each([
    { availableIngressProtocols: ['anthropic_messages'] as const, clientType: 'claude' as const, app: 'claude' },
    { availableIngressProtocols: ['gemini'] as const, clientType: 'gemini' as const, app: 'gemini' }
  ])('does not add a model parameter for $clientType imports', ({ availableIngressProtocols, clientType, app }) => {
    const params = paramsFromDeeplink(
      buildCcSwitchImportDeeplink({
        ...baseInput,
        availableIngressProtocols: [...availableIngressProtocols],
        clientType
      })
    )

    expect(params.get('app')).toBe(app)
    expect(params.get('endpoint')).toBe(baseInput.baseUrl)
    expect(params.has('model')).toBe(false)
  })

  it('does not fall back to Claude when no ingress protocol is known', () => {
    expect(resolveCcSwitchImportConfig([], 'claude', baseInput.baseUrl)).toBeNull()
    expect(() =>
      buildCcSwitchImportDeeplink({
        ...baseInput,
        availableIngressProtocols: [],
        clientType: 'claude'
      })
    ).toThrow('Unsupported CC Switch client protocol')
  })
})
