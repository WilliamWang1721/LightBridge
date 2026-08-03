import type { GroupUpstreamProtocol } from '@/types'

export type CcSwitchClientType = 'claude' | 'gemini' | 'codex'

export interface CcSwitchImportConfig {
  app: string
  endpoint: string
  model?: string
}

export interface CcSwitchImportDeeplinkInput {
  baseUrl: string
  endpoint?: string
  model?: string
  availableIngressProtocols?: GroupUpstreamProtocol[] | null
  clientType: CcSwitchClientType
  providerName: string
  apiKey: string
  usageScript: string
}

export function resolveCcSwitchImportConfig(
  availableIngressProtocols: GroupUpstreamProtocol[] | undefined | null,
  clientType: CcSwitchClientType,
  endpoint: string,
  model?: string
): CcSwitchImportConfig | null {
  const protocols = new Set(availableIngressProtocols ?? [])
  const normalizedEndpoint = endpoint.trim()
  const normalizedModel = model?.trim()
  switch (clientType) {
    case 'codex':
      if (!protocols.has('openai_responses')) return null
      return {
        app: 'codex',
        endpoint: normalizedEndpoint,
        ...(normalizedModel ? { model: normalizedModel } : {})
      }
    case 'gemini':
      if (!protocols.has('gemini')) return null
      return {
        app: 'gemini',
        endpoint: normalizedEndpoint
      }
    case 'claude':
      if (!protocols.has('anthropic_messages')) return null
      return {
        app: 'claude',
        endpoint: normalizedEndpoint
      }
  }
}

export function buildCcSwitchImportDeeplink(input: CcSwitchImportDeeplinkInput): string {
  const config = resolveCcSwitchImportConfig(
    input.availableIngressProtocols,
    input.clientType,
    input.endpoint || input.baseUrl,
    input.model
  )
  if (!config) {
    throw new Error(`Unsupported CC Switch client protocol: ${input.clientType}`)
  }
  const entries: [string, string][] = [
    ['resource', 'provider'],
    ['app', config.app],
    ['name', input.providerName],
    ['homepage', input.baseUrl],
    ['endpoint', config.endpoint],
    ['apiKey', input.apiKey],
    ['configFormat', 'json'],
    ['usageEnabled', 'true'],
    ['usageScript', btoa(input.usageScript)],
    ['usageAutoInterval', '30']
  ]

  if (config.model) {
    entries.splice(2, 0, ['model', config.model])
  }

  return `ccswitch://v1/import?${new URLSearchParams(entries).toString()}`
}
