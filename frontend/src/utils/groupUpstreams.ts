import type { GroupIcon, GroupPlatform, GroupUpstreamProtocol } from '@/types'

const upstreamPlatformValues = new Set<GroupPlatform>([
  'anthropic',
  'openai',
  'gemini',
  'grok',
  'antigravity',
  'custom',
])

export function normalizeGroupUpstreamPlatforms(
  platforms: readonly string[] | null | undefined,
): GroupPlatform[] {
  if (!platforms?.length) return []

  return [...new Set(platforms)]
    .map((platform) => platform.trim().toLowerCase())
    .filter((platform): platform is GroupPlatform =>
      upstreamPlatformValues.has(platform as GroupPlatform),
    )
}

export function normalizeGroupUpstreamProtocols(
  protocols: readonly string[] | null | undefined,
): GroupUpstreamProtocol[] {
  if (!protocols?.length) return []
  const supported = new Set<GroupUpstreamProtocol>([
    'openai_responses',
    'openai_chat_completions',
    'openai_embeddings',
    'anthropic_messages',
    'gemini',
  ])
  return [...new Set(protocols)]
    .map((protocol) => protocol.trim().toLowerCase())
    .filter((protocol): protocol is GroupUpstreamProtocol =>
      supported.has(protocol as GroupUpstreamProtocol),
    )
}

export const groupIconValues: GroupIcon[] = [
  'folder',
  'server',
  'cloud',
  'bolt',
  'shield',
  'cube',
  'terminal',
  'sparkles',
  'users',
]

const groupIconValueSet = new Set<GroupIcon>(groupIconValues)

export function normalizeGroupIcon(icon: string | null | undefined): GroupIcon {
  return groupIconValueSet.has(icon as GroupIcon) ? (icon as GroupIcon) : 'folder'
}

export function normalizeGroupColor(color: string | null | undefined): string {
  const normalized = color?.trim().toUpperCase() || ''
  return /^#[0-9A-F]{6}$/.test(normalized) ? normalized : ''
}

export function groupUpstreamProtocolBadgeClass(
  protocol: GroupUpstreamProtocol | string,
): string[] {
  return [
    'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
    protocol === 'openai_responses'
      ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
      : protocol === 'openai_chat_completions'
        ? 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-400'
        : protocol === 'openai_embeddings'
          ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400'
        : protocol === 'anthropic_messages'
          ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
          : protocol === 'gemini'
            ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
            : 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300',
  ]
}

export function groupUpstreamPlatformTextClass(platform: GroupPlatform): string {
  switch (platform) {
    case 'anthropic':
      return 'text-orange-700 dark:text-orange-400'
    case 'openai':
      return 'text-emerald-700 dark:text-emerald-400'
    case 'grok':
      return 'text-zinc-700 dark:text-zinc-200'
    case 'gemini':
    case 'antigravity':
      return 'text-blue-700 dark:text-blue-400'
    default:
      return 'text-gray-600 dark:text-gray-300'
  }
}
