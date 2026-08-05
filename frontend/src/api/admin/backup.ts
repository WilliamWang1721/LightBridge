import { apiClient } from '../client'
export interface BackupS3Config { endpoint: string; region: string; bucket: string; access_key_id: string; secret_access_key?: string; prefix: string; force_path_style: boolean }
export interface BackupScheduleConfig { enabled: boolean; cron_expr: string; retain_days: number; retain_count: number }
export interface VersionManagerBackupMetadata { source_version: string; version_action: 'update' | 'rollback'; target_version?: string; system_operation_id: string; initiating_admin_id: number }
export interface BackupRecord { id: string; status: 'pending' | 'running' | 'completed' | 'failed'; backup_type: string; file_name: string; s3_key: string; size_bytes: number; triggered_by: string; error_message?: string; started_at: string; finished_at?: string; expires_at?: string; progress?: string; restore_status?: string; restore_error?: string; restored_at?: string; metadata?: VersionManagerBackupMetadata }
export interface CreateBackupRequest { expire_days?: number }
export interface TestS3Response { ok: boolean; message: string }
export async function getS3Config() { const { data } = await apiClient.get<BackupS3Config>('/admin/backups/s3-config'); return data }
export async function updateS3Config(config: BackupS3Config) { const { data } = await apiClient.put<BackupS3Config>('/admin/backups/s3-config', config); return data }
export async function testS3Connection(config: BackupS3Config) { const { data } = await apiClient.post<TestS3Response>('/admin/backups/s3-config/test', config); return data }
export async function getSchedule() { const { data } = await apiClient.get<BackupScheduleConfig>('/admin/backups/schedule'); return data }
export async function updateSchedule(config: BackupScheduleConfig) { const { data } = await apiClient.put<BackupScheduleConfig>('/admin/backups/schedule', config); return data }
export async function createBackup(req?: CreateBackupRequest) { const { data } = await apiClient.post<BackupRecord>('/admin/backups', req || {}); return data }
export async function downloadLocalBackup(): Promise<string> {
  const response = await apiClient.post<Blob>('/admin/backups', { destination: 'local' }, { responseType: 'blob', timeout: 0, headers: { Accept: 'application/gzip' } })
  const disposition = String(response.headers['content-disposition'] || '')
  const match = disposition.match(/filename="?([^";]+)"?/i)
  const fileName = match?.[1] || `lightbridge_${new Date().toISOString().replace(/[:.]/g, '-')}.sql.gz`
  const url = URL.createObjectURL(response.data)
  const a = document.createElement('a')
  a.href = url
  a.download = fileName
  a.style.display = 'none'
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 0)
  return fileName
}
export async function listBackups() { const { data } = await apiClient.get<{ items: BackupRecord[] }>('/admin/backups'); return data }
export async function getBackup(id: string) { const { data } = await apiClient.get<BackupRecord>(`/admin/backups/${id}`); return data }
export async function deleteBackup(id: string) { await apiClient.delete(`/admin/backups/${id}`) }
export async function getDownloadURL(id: string) { const { data } = await apiClient.get<{ url: string }>(`/admin/backups/${id}/download-url`); return data }
export async function restoreBackup(id: string, password: string) { const { data } = await apiClient.post<BackupRecord>(`/admin/backups/${id}/restore`, { password }); return data }
export const backupAPI = { getS3Config, updateS3Config, testS3Connection, getSchedule, updateSchedule, createBackup, downloadLocalBackup, listBackups, getBackup, deleteBackup, getDownloadURL, restoreBackup }
export default backupAPI
