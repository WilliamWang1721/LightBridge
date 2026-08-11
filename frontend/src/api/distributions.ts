import { apiClient } from './client'
import type { BasePaginationResponse } from '@/types'

export type DistributionKind = 'text' | 'message' | 'file' | 'account_export' | 'paid'

export interface DistributionUserFilters {
  status?: string
  role?: string
  search?: string
  group_name?: string
  activity?: string
  attributes?: Record<number, string>
}

export interface DistributionAudienceInput {
  all?: boolean
  user_ids?: number[]
  emails?: string[]
  filters?: DistributionUserFilters
  lines?: string
}

export interface DistributionItem {
  id: number
  title: string
  kind: DistributionKind
  price?: number
  content: string
  file_name?: string
  content_type?: string
  file_size: number
  has_attachment: boolean
  metadata?: Record<string, unknown>
  audience?: Record<string, unknown>
  created_by?: number
  created_at: string
  recipient_count: number
  read_count: number
  download_count: number
  read_at?: string
  downloaded_at?: string
  accepted_at?: string
  rejected_at?: string
}

export interface CreateDistributionRequest {
  title: string
  kind: DistributionKind
  price?: number
  content?: string
  file_name?: string
  content_type?: string
  file_base64?: string
  metadata?: Record<string, unknown>
  audience: DistributionAudienceInput
}

export interface DistributionAudiencePreview {
  count: number
  preview: Array<{ id: number; email: string; username: string }>
}

export interface DistributionBatchResult {
  succeeded: number
  failed: number
  items: Array<{
    index: number
    distribution_id?: number
    recipient_count?: number
    error?: string
  }>
}

export async function listMyDistributions(page = 1, pageSize = 20, unreadOnly = false) {
  const { data } = await apiClient.get<BasePaginationResponse<DistributionItem>>('/distributions', {
    params: { page, page_size: pageSize, unread_only: unreadOnly ? 1 : undefined }
  })
  return data
}

export async function getMyDistribution(id: number) {
  const { data } = await apiClient.get<DistributionItem>(`/distributions/${id}`)
  return data
}

export async function markDistributionRead(id: number) {
  const { data } = await apiClient.post<{ message: string }>(`/distributions/${id}/read`)
  return data
}

export async function acceptMyDistribution(id: number) {
  const { data } = await apiClient.post<{ message: string }>(`/distributions/${id}/accept`)
  return data
}

export async function rejectMyDistribution(id: number) {
  const { data } = await apiClient.post<{ message: string }>(`/distributions/${id}/reject`)
  return data
}

export async function downloadMyDistribution(id: number): Promise<Blob> {
  const response = await apiClient.get(`/distributions/${id}/download`, { responseType: 'blob' })
  return response.data as Blob
}

export async function listAdminDistributions(page = 1, pageSize = 20) {
  const { data } = await apiClient.get<BasePaginationResponse<DistributionItem>>('/admin/distributions', {
    params: { page, page_size: pageSize }
  })
  return data
}

export async function createDistribution(request: CreateDistributionRequest) {
  const { data } = await apiClient.post<DistributionItem>('/admin/distributions', request)
  return data
}

export async function batchCreateDistributions(items: CreateDistributionRequest[]) {
  const { data } = await apiClient.post<DistributionBatchResult>('/admin/distributions/batch', { items })
  return data
}

export async function previewDistributionAudience(audience: DistributionAudienceInput) {
  const { data } = await apiClient.post<DistributionAudiencePreview>('/admin/distributions/audience-preview', audience)
  return data
}

export async function deleteDistribution(id: number) {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/distributions/${id}`)
  return data
}

export async function downloadAdminDistribution(id: number): Promise<Blob> {
  const response = await apiClient.get(`/admin/distributions/${id}/download`, { responseType: 'blob' })
  return response.data as Blob
}
