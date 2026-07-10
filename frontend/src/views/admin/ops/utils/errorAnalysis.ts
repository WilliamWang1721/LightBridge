import type { OpsErrorDetail, OpsErrorLog } from '@/api/admin/ops'
import type { Account } from '@/types'

export type ErrorAnalysisStepState = 'passed' | 'failed' | 'warning' | 'skipped' | 'pending'

export type ErrorAnalysisStepKey =
  | 'request_intake'
  | 'auth'
  | 'routing'
  | 'account_scheduler'
  | 'provider_adapter'
  | 'upstream'
  | 'response'

export type ErrorAnalysisRootCause =
  | 'no_available_account'
  | 'auth_forbidden'
  | 'client_request'
  | 'provider_upstream'
  | 'platform_internal'
  | 'network'
  | 'unknown'

export interface ErrorAnalysisEvidence {
  key: string
  value: string
  tone?: 'neutral' | 'good' | 'warning' | 'danger'
}

export interface ErrorAnalysisStep {
  key: ErrorAnalysisStepKey
  module: string
  state: ErrorAnalysisStepState
  evidence: ErrorAnalysisEvidence[]
}

export interface ErrorAnalysisResult {
  rootCause: ErrorAnalysisRootCause
  rootModule: string
  confidence: 'high' | 'medium' | 'low'
  failedStep: ErrorAnalysisStepKey
  steps: ErrorAnalysisStep[]
  evidence: ErrorAnalysisEvidence[]
  suggestionKeys: string[]
}

export type ErrorAnalysisAccountReasonKey =
  | 'group_mismatch'
  | 'status_inactive'
  | 'status_error'
  | 'unschedulable'
  | 'rate_limited'
  | 'temp_unschedulable'
  | 'overloaded'
  | 'expired'
  | 'concurrency_full'
  | 'rpm_limit'
  | 'quota_exhausted'
  | 'daily_quota_exhausted'
  | 'weekly_quota_exhausted'
  | 'session_window_rejected'
  | 'model_not_allowed'
  | 'model_rate_limited'
  | 'quota_auto_paused'
  | 'account_nil'
  | 'excluded'
  | 'custom_protocol_missing'
  | 'protocol_incompatible'
  | 'relay_mode_protocol_mismatch'
  | 'protocol_conversion_unavailable'
  | 'schedulable_disabled'
  | 'privacy_required'
  | 'channel_restricted'
  | 'window_cost_exceeded'
  | 'credentials_missing'

export interface ErrorAnalysisAccountReason {
  key: ErrorAnalysisAccountReasonKey
  detail?: string
  source?: 'request_time' | 'current_state'
}

export interface ErrorAnalysisAccountDiagnostic {
  account: Account
  available: boolean
  reasons: ErrorAnalysisAccountReason[]
  source?: 'request_time' | 'current_state'
}

interface RecordedSchedulerAccountDiagnostic {
  account_id: number
  available: boolean
  reason?: ErrorAnalysisAccountReasonKey
  detail?: string
}

interface RecordedSchedulerDiagnostics {
  version: number
  inbound_protocol?: string
  requested_model?: string
  group_id?: number
  platform?: string
  reason_counts?: Record<string, number>
  accounts?: RecordedSchedulerAccountDiagnostic[]
}

const NO_AVAILABLE_ACCOUNT_RE = /no\s+available\s+accounts?|无可用账号/i
const MODULE_PROVIDER_RE = /module provider|provider registry|provider id|adapter/i
const NETWORK_RE = /timeout|deadline|connection refused|connection reset|dial tcp|tls|dns|network/i
const SCHEDULER_DIAGNOSTICS_RE = /scheduler_diagnostics=([A-Za-z0-9_-]+)/

function textOf(detail: OpsErrorDetail | OpsErrorLog | null | undefined): string {
  if (!detail) return ''
  return [
    detail.message,
    'error_body' in detail ? detail.error_body : '',
    'upstream_error_message' in detail ? detail.upstream_error_message : '',
    'upstream_error_detail' in detail ? detail.upstream_error_detail : '',
    'upstream_errors' in detail ? detail.upstream_errors : ''
  ].filter(Boolean).join('\n')
}

function normalize(value: unknown): string {
  return String(value ?? '').trim().toLowerCase()
}

export function parseRecordedSchedulerDiagnostics(
  detail: OpsErrorDetail | OpsErrorLog | null | undefined
): RecordedSchedulerDiagnostics | null {
  const match = textOf(detail).match(SCHEDULER_DIAGNOSTICS_RE)
  if (!match?.[1]) return null
  try {
    const normalized = match[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = normalized + '='.repeat((4 - normalized.length % 4) % 4)
    const binary = atob(padded)
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
    return JSON.parse(new TextDecoder().decode(bytes)) as RecordedSchedulerDiagnostics
  } catch {
    return null
  }
}

function isFuture(value: string | number | null | undefined, now = Date.now()): boolean {
  if (value == null || value === '') return false
  const timestamp = typeof value === 'number'
    ? value < 1000000000000 ? value * 1000 : value
    : new Date(value).getTime()
  return Number.isFinite(timestamp) && timestamp > now
}

function getAccountModelWhitelist(account: Account): string[] {
  const raw = account.extra?.model_whitelist ?? account.extra?.models ?? account.extra?.supported_models
  if (Array.isArray(raw)) return raw.map((item) => String(item).trim()).filter(Boolean)
  if (typeof raw === 'string') {
    return raw.split(/[,\n]/).map((item) => item.trim()).filter(Boolean)
  }
  return []
}

function modelMatchesPattern(model: string, pattern: string): boolean {
  const normalizedModel = normalize(model)
  const normalizedPattern = normalize(pattern)
  if (!normalizedModel || !normalizedPattern) return false
  if (normalizedPattern === '*') return true
  if (!normalizedPattern.includes('*')) return normalizedModel === normalizedPattern

  const escaped = normalizedPattern
    .split('*')
    .map((part) => part.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
    .join('.*')
  return new RegExp(`^${escaped}$`).test(normalizedModel)
}

function accountAllowsModel(account: Account, model: string): boolean {
  const whitelist = getAccountModelWhitelist(account)
  if (whitelist.length === 0 || !model.trim()) return true
  return whitelist.some((pattern) => modelMatchesPattern(model, pattern))
}

function hasNoAvailableAccount(detail: OpsErrorDetail | null): boolean {
  return NO_AVAILABLE_ACCOUNT_RE.test(textOf(detail))
}

function hasNetworkSignal(detail: OpsErrorDetail | null, upstreamErrors: OpsErrorDetail[]): boolean {
  if (NETWORK_RE.test(textOf(detail))) return true
  return upstreamErrors.some((item) => NETWORK_RE.test(textOf(item)))
}

function isProviderBridgeFailure(detail: OpsErrorDetail | null): boolean {
  return MODULE_PROVIDER_RE.test(textOf(detail))
}

function hasUpstreamAttempt(detail: OpsErrorDetail | null, upstreamErrors: OpsErrorDetail[]): boolean {
  if (upstreamErrors.length > 0) return true
  if (detail?.upstream_status_code != null) return true
  if (String(detail?.upstream_error_message || '').trim()) return true
  if (String(detail?.upstream_error_detail || '').trim()) return true
  return false
}

function isCustomProviderContext(detail: OpsErrorDetail | null, upstreamErrors: OpsErrorDetail[]): boolean {
  if (!detail) return false
  if (normalize(detail.platform) === 'custom') return true
  if (normalize(detail.upstream_endpoint).includes('custom')) return true
  return upstreamErrors.some((item) => normalize(item.platform) === 'custom')
}

function determineRootCause(detail: OpsErrorDetail | null, upstreamErrors: OpsErrorDetail[]): ErrorAnalysisRootCause {
  if (!detail) return 'unknown'

  const phase = normalize(detail.phase)
  const owner = normalize(detail.error_owner)
  const source = normalize(detail.error_source)
  const status = detail.status_code || 0
  const upstreamAttempted = hasUpstreamAttempt(detail, upstreamErrors)

  if (hasNoAvailableAccount(detail)) return 'no_available_account'
  if (phase === 'routing' && status === 503 && !upstreamAttempted) return 'no_available_account'
  if (status === 503 && !detail.account_id && !upstreamAttempted) return 'no_available_account'

  if (phase === 'auth' || (status === 403 && owner !== 'provider' && !upstreamAttempted)) {
    return 'auth_forbidden'
  }

  if (owner === 'client' || source === 'client_request' || phase === 'request') {
    return 'client_request'
  }

  if (hasNetworkSignal(detail, upstreamErrors)) return 'network'

  if (phase === 'upstream' || owner === 'provider' || upstreamAttempted) {
    return 'provider_upstream'
  }

  if (phase === 'internal' || owner === 'platform' || status >= 500) {
    return 'platform_internal'
  }

  return 'unknown'
}

function failedStepForCause(cause: ErrorAnalysisRootCause, detail: OpsErrorDetail | null): ErrorAnalysisStepKey {
  if (cause === 'auth_forbidden') return 'auth'
  if (cause === 'client_request') return normalize(detail?.phase) === 'routing' ? 'routing' : 'request_intake'
  if (cause === 'no_available_account') return 'account_scheduler'
  if (cause === 'provider_upstream' || cause === 'network') {
    return isProviderBridgeFailure(detail) ? 'provider_adapter' : 'upstream'
  }
  if (cause === 'platform_internal') return 'response'
  return 'response'
}

function stateForStep(
  key: ErrorAnalysisStepKey,
  failedStep: ErrorAnalysisStepKey,
  detail: OpsErrorDetail | null,
  upstreamErrors: OpsErrorDetail[]
): ErrorAnalysisStepState {
  const order: ErrorAnalysisStepKey[] = [
    'request_intake',
    'auth',
    'routing',
    'account_scheduler',
    'provider_adapter',
    'upstream',
    'response'
  ]
  const failedIdx = order.indexOf(failedStep)
  const idx = order.indexOf(key)
  const upstreamAttempted = hasUpstreamAttempt(detail, upstreamErrors)

  if (idx < failedIdx) return 'passed'
  if (key === failedStep) return 'failed'

  if (failedStep === 'account_scheduler' && (key === 'provider_adapter' || key === 'upstream')) {
    return 'skipped'
  }
  if (key === 'upstream' && upstreamAttempted) return 'warning'
  if (key === 'response') return 'warning'
  return idx > failedIdx ? 'pending' : 'passed'
}

function pushEvidence(out: ErrorAnalysisEvidence[], key: string, value: unknown, tone?: ErrorAnalysisEvidence['tone']) {
  const str = String(value ?? '').trim()
  if (!str) return
  out.push({ key, value: str, tone })
}

function compactRequestId(detail: OpsErrorDetail | null): string {
  return String(detail?.request_id || detail?.client_request_id || '').trim()
}

function modelLabel(detail: OpsErrorDetail | null): string {
  if (!detail) return ''
  const requested = String(detail.requested_model || '').trim()
  const upstream = String(detail.upstream_model || '').trim()
  if (requested && upstream && requested !== upstream) return `${requested} -> ${upstream}`
  return upstream || requested || String(detail.model || '').trim()
}

function requestedModelLabel(detail: OpsErrorDetail | null): string {
  if (!detail) return ''
  return String(detail.requested_model || detail.model || detail.upstream_model || '').trim()
}

function buildStepEvidence(
  key: ErrorAnalysisStepKey,
  detail: OpsErrorDetail | null,
  upstreamErrors: OpsErrorDetail[]
): ErrorAnalysisEvidence[] {
  const evidence: ErrorAnalysisEvidence[] = []
  if (!detail) return evidence

  switch (key) {
    case 'request_intake':
      pushEvidence(evidence, 'request_id', compactRequestId(detail))
      pushEvidence(evidence, 'endpoint', detail.inbound_endpoint || detail.request_path)
      pushEvidence(evidence, 'request_type', detail.stream ? 'stream' : String(detail.request_type ?? 'sync'))
      break
    case 'auth':
      pushEvidence(evidence, 'user', detail.user_email || detail.user_id)
      pushEvidence(evidence, 'api_key_id', detail.api_key_id)
      pushEvidence(evidence, 'client_ip', detail.client_ip)
      break
    case 'routing':
      pushEvidence(evidence, 'platform', detail.platform)
      pushEvidence(evidence, 'group', detail.group_name || detail.group_id)
      pushEvidence(evidence, 'model', modelLabel(detail))
      break
    case 'account_scheduler': {
      pushEvidence(evidence, 'account', detail.account_name || detail.account_id || 'none', detail.account_id ? 'neutral' : 'warning')
      pushEvidence(evidence, 'status', detail.status_code, detail.status_code >= 500 ? 'danger' : 'warning')
      if (hasNoAvailableAccount(detail)) pushEvidence(evidence, 'scheduler_result', 'No available accounts', 'danger')
      if (isCustomProviderContext(detail, upstreamErrors)) pushEvidence(evidence, 'provider_kind', 'custom', 'warning')
      const recorded = parseRecordedSchedulerDiagnostics(detail)
      if (recorded?.inbound_protocol) pushEvidence(evidence, 'inbound_protocol', recorded.inbound_protocol)
      if (recorded?.accounts) pushEvidence(evidence, 'evaluated_accounts', recorded.accounts.length)
      if (recorded?.reason_counts) {
        const summary = Object.entries(recorded.reason_counts)
          .sort((a, b) => b[1] - a[1])
          .map(([reason, count]) => `${reason}:${count}`)
          .join(', ')
        pushEvidence(evidence, 'rejection_reasons', summary, 'danger')
      }
      break
    }
    case 'provider_adapter':
      pushEvidence(evidence, 'provider_module', isCustomProviderContext(detail, upstreamErrors) ? 'custom provider adapter' : 'built-in provider adapter')
      pushEvidence(evidence, 'upstream_endpoint', detail.upstream_endpoint)
      pushEvidence(evidence, 'selected_account', detail.account_name || detail.account_id)
      break
    case 'upstream':
      pushEvidence(evidence, 'attempts', upstreamErrors.length || (detail.upstream_status_code != null ? 1 : 0), upstreamErrors.length ? 'warning' : 'neutral')
      pushEvidence(evidence, 'upstream_status', detail.upstream_status_code)
      pushEvidence(evidence, 'upstream_message', detail.upstream_error_message)
      pushEvidence(evidence, 'upstream_detail', detail.upstream_error_detail)
      break
    case 'response':
      pushEvidence(evidence, 'client_status', detail.status_code, detail.status_code >= 500 ? 'danger' : 'warning')
      pushEvidence(evidence, 'message', detail.message)
      break
  }

  return evidence
}

function moduleForStep(key: ErrorAnalysisStepKey): string {
  switch (key) {
    case 'request_intake':
      return 'server.routes.gateway + ops_error_logger'
    case 'auth':
      return 'middleware.api_key_auth + subscription_guard'
    case 'routing':
      return 'gateway_handler + model_routing'
    case 'account_scheduler':
      return 'openai_account_scheduler + scheduling_platform'
    case 'provider_adapter':
      return 'provider_module_bridge / custom_provider_adapter'
    case 'upstream':
      return 'upstream_http_client + provider_runtime'
    case 'response':
      return 'response_writer + error_passthrough'
  }
}

function rootModule(cause: ErrorAnalysisRootCause, failedStep: ErrorAnalysisStepKey): string {
  if (cause === 'no_available_account') return 'openai_account_scheduler'
  if (cause === 'auth_forbidden') return 'middleware.api_key_auth'
  if (cause === 'client_request') return 'gateway_request_validator'
  if (cause === 'provider_upstream') return 'upstream_http_client'
  if (cause === 'network') return 'upstream_http_client'
  if (cause === 'platform_internal') return 'gateway_service'
  return moduleForStep(failedStep)
}

function confidenceFor(cause: ErrorAnalysisRootCause, detail: OpsErrorDetail | null, upstreamErrors: OpsErrorDetail[]): ErrorAnalysisResult['confidence'] {
  if (!detail) return 'low'
  if (cause === 'no_available_account' && hasNoAvailableAccount(detail)) return 'high'
  if (cause === 'provider_upstream' && hasUpstreamAttempt(detail, upstreamErrors)) return 'high'
  if (cause === 'auth_forbidden' && normalize(detail.phase) === 'auth') return 'high'
  if (cause === 'network' && hasNetworkSignal(detail, upstreamErrors)) return 'high'
  if (cause === 'unknown') return 'low'
  return 'medium'
}

function suggestionKeysFor(cause: ErrorAnalysisRootCause, detail: OpsErrorDetail | null, upstreamErrors: OpsErrorDetail[]): string[] {
  const custom = isCustomProviderContext(detail, upstreamErrors)

  if (cause === 'no_available_account') {
    return custom
      ? [
          'customCheckAccountGroup',
          'customCheckProtocol',
          'customCheckModelScope',
          'customCheckAccountAvailability',
          'customNoUpstreamAttempt'
        ]
      : ['checkAccountGroup', 'checkModelScope', 'checkAccountAvailability', 'checkRateLimitOrTempUnsched']
  }

  if (cause === 'auth_forbidden') return ['checkApiKeyPermission', 'checkUserGroupAccess', 'checkSubscriptionAndBalance']
  if (cause === 'client_request') return ['checkRequestEndpoint', 'checkRequestModel', 'checkRequestPayload']
  if (cause === 'provider_upstream') return ['checkProviderStatus', 'checkProviderCredential', 'checkProviderModelPermission']
  if (cause === 'network') return ['checkProxyNetwork', 'checkProviderBaseUrl', 'checkTlsFingerprint']
  if (cause === 'platform_internal') return ['checkSystemLogs', 'checkGatewayConfig', 'checkRecentDeployment']
  return ['inspectRawError', 'compareNearbyRequests']
}

export function buildErrorAnalysis(
  detail: OpsErrorDetail | null,
  upstreamErrors: OpsErrorDetail[] = []
): ErrorAnalysisResult {
  const cause = determineRootCause(detail, upstreamErrors)
  const failedStep = failedStepForCause(cause, detail)
  const stepKeys: ErrorAnalysisStepKey[] = [
    'request_intake',
    'auth',
    'routing',
    'account_scheduler',
    'provider_adapter',
    'upstream',
    'response'
  ]

  const steps = stepKeys.map((key) => ({
    key,
    module: moduleForStep(key),
    state: stateForStep(key, failedStep, detail, upstreamErrors),
    evidence: buildStepEvidence(key, detail, upstreamErrors)
  }))

  const evidence: ErrorAnalysisEvidence[] = []
  if (detail) {
    pushEvidence(evidence, 'status', detail.status_code, detail.status_code >= 500 ? 'danger' : 'warning')
    pushEvidence(evidence, 'phase', detail.phase)
    pushEvidence(evidence, 'owner', detail.error_owner)
    pushEvidence(evidence, 'source', detail.error_source)
    pushEvidence(evidence, 'request_id', compactRequestId(detail))
    pushEvidence(evidence, 'platform', detail.platform)
    pushEvidence(evidence, 'group', detail.group_name || detail.group_id)
    pushEvidence(evidence, 'model', modelLabel(detail))
    pushEvidence(evidence, 'upstream_attempts', upstreamErrors.length, upstreamErrors.length ? 'warning' : 'neutral')
  }

  return {
    rootCause: cause,
    rootModule: rootModule(cause, failedStep),
    confidence: confidenceFor(cause, detail, upstreamErrors),
    failedStep,
    steps,
    evidence,
    suggestionKeys: suggestionKeysFor(cause, detail, upstreamErrors)
  }
}

export function shortErrorMessage(detail: OpsErrorLog | OpsErrorDetail | null | undefined): string {
  const message = String(detail?.message || '').trim()
  if (!message) return ''
  if (message.length <= 160) return message
  return `${message.slice(0, 157)}...`
}

export function accountDisplayLabel(account: Account): string {
  return account.name || `#${account.id}`
}

export function diagnoseSchedulerAccount(
  account: Account,
  detail: OpsErrorDetail | null,
  now = Date.now()
): ErrorAnalysisAccountDiagnostic {
  const recorded = parseRecordedSchedulerDiagnostics(detail)
  const recordedAccount = recorded?.accounts?.find((item) => item.account_id === account.id)
  if (recordedAccount) {
    const reasons: ErrorAnalysisAccountReason[] = []
    if (!recordedAccount.available && recordedAccount.reason) {
      reasons.push({ key: recordedAccount.reason, detail: recordedAccount.detail, source: 'request_time' })
    }
    return { account, available: recordedAccount.available, reasons, source: 'request_time' }
  }

  const reasons: ErrorAnalysisAccountReason[] = []
  const groupID = detail?.group_id ?? null
  const model = requestedModelLabel(detail)
  const accountGroupIDs = account.group_ids ?? account.groups?.map((group) => group.id) ?? []

  if (groupID != null && !accountGroupIDs.includes(groupID)) {
    reasons.push({ key: 'group_mismatch', detail: String(groupID) })
  }

  if (account.status === 'inactive') reasons.push({ key: 'status_inactive' })
  if (account.status === 'error') reasons.push({ key: 'status_error', detail: account.error_message || undefined })
  if (account.schedulable === false) reasons.push({ key: 'unschedulable' })
  if (isFuture(account.rate_limit_reset_at, now)) reasons.push({ key: 'rate_limited', detail: account.rate_limit_reset_at || undefined })
  if (isFuture(account.temp_unschedulable_until, now)) reasons.push({ key: 'temp_unschedulable', detail: account.temp_unschedulable_reason || account.temp_unschedulable_until || undefined })
  if (isFuture(account.overload_until, now)) reasons.push({ key: 'overloaded', detail: account.overload_until || undefined })
  if (isFuture(account.expires_at, now) === false && account.expires_at != null && account.expires_at > 0) reasons.push({ key: 'expired' })

  const concurrency = Number(account.concurrency ?? 0)
  const currentConcurrency = Number(account.current_concurrency ?? 0)
  if (concurrency > 0 && currentConcurrency >= concurrency) {
    reasons.push({ key: 'concurrency_full', detail: `${currentConcurrency}/${concurrency}` })
  }

  const baseRPM = Number(account.base_rpm ?? 0)
  const currentRPM = Number(account.current_rpm ?? 0)
  if (baseRPM > 0 && currentRPM >= baseRPM) {
    reasons.push({ key: 'rpm_limit', detail: `${currentRPM}/${baseRPM}` })
  }

  if (typeof account.quota_limit === 'number' && account.quota_limit > 0 && Number(account.quota_used ?? 0) >= account.quota_limit) {
    reasons.push({ key: 'quota_exhausted', detail: `${account.quota_used ?? 0}/${account.quota_limit}` })
  }
  if (typeof account.quota_daily_limit === 'number' && account.quota_daily_limit > 0 && Number(account.quota_daily_used ?? 0) >= account.quota_daily_limit) {
    reasons.push({ key: 'daily_quota_exhausted', detail: `${account.quota_daily_used ?? 0}/${account.quota_daily_limit}` })
  }
  if (typeof account.quota_weekly_limit === 'number' && account.quota_weekly_limit > 0 && Number(account.quota_weekly_used ?? 0) >= account.quota_weekly_limit) {
    reasons.push({ key: 'weekly_quota_exhausted', detail: `${account.quota_weekly_used ?? 0}/${account.quota_weekly_limit}` })
  }

  if (account.session_window_status === 'rejected') {
    reasons.push({ key: 'session_window_rejected' })
  }

  if (model && !accountAllowsModel(account, model)) {
    reasons.push({ key: 'model_not_allowed', detail: model })
  }

  // Model-level rate limit (extra.model_rate_limits)
  if (model) {
    const modelRateLimits = (account.extra as Record<string, unknown> | undefined)?.model_rate_limits as Record<string, { rate_limited_at?: string; rate_limit_reset_at?: string }> | undefined
    if (modelRateLimits && typeof modelRateLimits === 'object') {
      const limitEntry = modelRateLimits[model]
      if (limitEntry && isFuture(limitEntry.rate_limit_reset_at, now)) {
        reasons.push({ key: 'model_rate_limited', detail: `${model}: resets at ${limitEntry.rate_limit_reset_at || '?'}` })
      }
    }
  }

  // OpenAI quota auto-pause (5h / 7d utilization threshold)
  if (account.platform === 'openai' || account.platform === 'antigravity') {
    const extra = (account.extra as Record<string, unknown> | undefined) ?? {}
    for (const window of ['5h', '7d'] as const) {
      const disabledKey = `auto_pause_${window}_disabled`
      if (extra[disabledKey]) continue
      const threshold = Number(extra[`auto_pause_threshold_${window}`] ?? 0)
      if (threshold <= 0) continue
      const statsKey = `usage_${window}_requests`
      const limitKey = `quota_${window}_requests`
      const stats = extra[statsKey] as Record<string, unknown> | undefined
      const limit = extra[limitKey] as Record<string, unknown> | undefined
      if (stats && limit) {
        const used = Number(stats.used ?? stats.total ?? 0)
        const max = Number(limit.limit ?? limit.total ?? 0)
        if (max > 0) {
          const utilization = used / max
          if (utilization >= threshold) {
            reasons.push({ key: 'quota_auto_paused', detail: `${window}: ${(utilization * 100).toFixed(0)}% utilized (threshold ${(threshold * 100).toFixed(0)}%)` })
          }
        }
      }
    }
  }

  return {
  account,
  available: reasons.length === 0,
  reasons: reasons.map((reason) => ({ ...reason, source: 'current_state' })),
  source: 'current_state'
}
}

export function diagnoseSchedulerAccounts(
  accounts: Account[],
  detail: OpsErrorDetail | null,
  now = Date.now()
): ErrorAnalysisAccountDiagnostic[] {
  return accounts.map((account) => diagnoseSchedulerAccount(account, detail, now))
}
