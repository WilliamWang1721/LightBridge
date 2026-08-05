import { apiClient } from '../client'
import type { BackupRecord } from './backup'

export interface ReleaseInfo { name: string; body: string; published_at: string; html_url: string; prerelease?: boolean }
export interface VersionRelease { version: string; name: string; body: string; published_at: string; html_url: string; prerelease?: boolean; draft?: boolean; current?: boolean; latest?: boolean }
export interface UpdateCapabilities { deployment_type: 'binary' | 'container' | string; can_in_place_update: boolean; can_rollback: boolean; can_restart: boolean }
export const unavailableUpdateCapabilities: UpdateCapabilities = { deployment_type: 'unknown', can_in_place_update: false, can_rollback: false, can_restart: false }
export interface VersionReleasesResult { current_version: string; latest_version: string; build_type: string; capabilities: UpdateCapabilities; releases: VersionRelease[] }
export interface VersionInfo { current_version: string; latest_version: string; has_update: boolean; release_info?: ReleaseInfo; cached: boolean; warning?: string; build_type: string; capabilities: UpdateCapabilities }
export async function getVersion() { const { data } = await apiClient.get<{ version: string }>('/admin/system/version'); return data }
export async function checkUpdates(force = false) { const { data } = await apiClient.get<VersionInfo>('/admin/system/check-updates', { params: force ? { force: 'true' } : undefined }); return data }
export async function listVersionReleases(force = false) { const { data } = await apiClient.get<VersionReleasesResult>('/admin/system/versions', { params: force ? { force: 'true' } : undefined }); return data }
export interface UpdateResult { message: string; need_restart: boolean; backup?: BackupRecord; forced_without_backup?: boolean }
export interface UpdateOptions { version?: string; backup_current?: boolean; force_without_backup?: boolean }
export async function performUpdate(options: UpdateOptions = {}) { const payload = Object.keys(options).length ? options : undefined; const { data } = await apiClient.post<UpdateResult>('/admin/system/update', payload); return data }
export interface RollbackOptions { backup_current?: boolean }
export async function rollback(options: RollbackOptions = {}) { const payload = Object.keys(options).length ? options : undefined; const { data } = await apiClient.post<UpdateResult>('/admin/system/rollback', payload); return data }
export async function restartService() { const { data } = await apiClient.post<{ message: string }>('/admin/system/restart'); return data }
export const systemAPI = { getVersion, checkUpdates, listVersionReleases, performUpdate, rollback, restartService }
export default systemAPI
