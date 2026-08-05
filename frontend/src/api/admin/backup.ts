import { apiClient } from '../client'

// A deterministic empty gzip member appended by the backend only after the
// database dump and primary gzip stream complete successfully.
export const LOCAL_BACKUP_COMPLETION_MARKER = new Uint8Array([
  0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00,
  0x00, 0xff, 0x01, 0x00, 0x00, 0xff, 0xff, 0x00,
  0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00
])

export interface BackupS3Config {
  endpoint: string
  region: string
  bucket: string
  access_key_id: string
  secret_access_key?: string
  prefix: string
  force_path_style: boolean
}

export interface BackupScheduleConfig {
  enabled: boolean
  cron_expr: string
  retain_days: number
  retain_count: number
}

/** Version Manager provenance attached only to its pre-action database snapshots. */
export interface VersionManagerBackupMetadata {
  source_version: string
  version_action: 'update' | 'rollback'
  target_version?: string
  system_operation_id: string
  initiating_admin_id: number
}

export interface BackupRecord {
  id: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  backup_type: string
  file_name: string
  s3_key: string
  size_bytes: number
  triggered_by: string
  error_message?: string
  started_at: string
  finished_at?: string
  expires_at?: string
  progress?: string
  restore_status?: string
  restore_error?: string
  restored_at?: string
  /** Present when Version Manager created this snapshot before an update or rollback. */
  metadata?: VersionManagerBackupMetadata
}

export interface CreateBackupRequest {
  expire_days?: number
  destination?: 's3' | 'local'
}

export interface TestS3Response {
  ok: boolean
  message: string
}

// S3 Config
export async function getS3Config(): Promise<BackupS3Config> {
  const { data } = await apiClient.get<BackupS3Config>('/admin/backups/s3-config')
  return data
}

export async function updateS3Config(config: BackupS3Config): Promise<BackupS3Config> {
  const { data } = await apiClient.put<BackupS3Config>('/admin/backups/s3-config', config)
  return data
}

export async function testS3Connection(config: BackupS3Config): Promise<TestS3Response> {
  const { data } = await apiClient.post<TestS3Response>('/admin/backups/s3-config/test', config)
  return data
}

// Schedule
export async function getSchedule(): Promise<BackupScheduleConfig> {
  const { data } = await apiClient.get<BackupScheduleConfig>('/admin/backups/schedule')
  return data
}

export async function updateSchedule(config: BackupScheduleConfig): Promise<BackupScheduleConfig> {
  const { data } = await apiClient.put<BackupScheduleConfig>('/admin/backups/schedule', config)
  return data
}

// Backup operations
export async function createBackup(req?: CreateBackupRequest): Promise<BackupRecord> {
  const { data } = await apiClient.post<BackupRecord>('/admin/backups', req || {})
  return data
}

export function readBlobAsArrayBuffer(blob: Blob): Promise<ArrayBuffer> {
  const modernBlob = blob as Blob & { arrayBuffer?: () => Promise<ArrayBuffer> }
  if (typeof modernBlob.arrayBuffer === 'function') {
    return modernBlob.arrayBuffer()
  }

  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      if (reader.result instanceof ArrayBuffer) {
        resolve(reader.result)
        return
      }
      reject(new Error('Failed to read backup data'))
    }
    reader.onerror = () => reject(reader.error || new Error('Failed to read backup data'))
    reader.readAsArrayBuffer(blob)
  })
}

async function readBlobAsText(blob: Blob): Promise<string> {
  const modernBlob = blob as Blob & { text?: () => Promise<string> }
  if (typeof modernBlob.text === 'function') {
    return modernBlob.text()
  }

  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '')
    reader.onerror = () => reject(reader.error || new Error('Failed to read error response'))
    reader.readAsText(blob)
  })
}

export async function validateLocalBackupDownload(envelope: Blob): Promise<Blob> {
  if (envelope.size <= LOCAL_BACKUP_COMPLETION_MARKER.byteLength) {
    throw new Error('Local backup stream is incomplete; no file was saved')
  }

  const suffix = new Uint8Array(
    await readBlobAsArrayBuffer(
      envelope.slice(envelope.size - LOCAL_BACKUP_COMPLETION_MARKER.byteLength)
    )
  )
  const completed = LOCAL_BACKUP_COMPLETION_MARKER.every((value, index) => suffix[index] === value)
  if (!completed) {
    throw new Error('Local backup stream is incomplete; no file was saved')
  }

  // The completion marker is itself a valid empty gzip member, so preserving it
  // keeps the downloaded file standards-compliant and independently verifiable.
  return envelope.slice(0, envelope.size, 'application/gzip')
}

export async function downloadLocalBackup(): Promise<string> {
  try {
    const response = await apiClient.post<Blob>(
      '/admin/backups',
      { destination: 'local' },
      {
        responseType: 'blob',
        timeout: 0,
        headers: { Accept: 'application/gzip' }
      }
    )

    const disposition = String(response.headers['content-disposition'] || '')
    const match = disposition.match(/filename="?([^";]+)"?/i)
    const fileName = match?.[1] || `lightbridge_${new Date().toISOString().replace(/[:.]/g, '-')}.sql.gz`
    const backupBlob = await validateLocalBackupDownload(response.data)
    const url = URL.createObjectURL(backupBlob)
    const link = document.createElement('a')
    link.href = url
    link.download = fileName
    link.style.display = 'none'
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
    return fileName
  } catch (error) {
    const err = error as {
      message?: string
      data?: unknown
      response?: { data?: unknown }
    }
    const responseData = err.response?.data ?? err.data
    if (responseData instanceof Blob) {
      try {
        const text = await readBlobAsText(responseData)
        const payload = JSON.parse(text) as { message?: string; detail?: string }
        const message = payload.message || payload.detail
        if (message) {
          throw new Error(message)
        }
      } catch (parseError) {
        if (parseError instanceof Error && parseError.name !== 'SyntaxError') {
          throw parseError
        }
      }
    }
    throw error
  }
}

export async function listBackups(): Promise<{ items: BackupRecord[] }> {
  const { data } = await apiClient.get<{ items: BackupRecord[] }>('/admin/backups')
  return data
}

export async function getBackup(id: string): Promise<BackupRecord> {
  const { data } = await apiClient.get<BackupRecord>(`/admin/backups/${id}`)
  return data
}

export async function deleteBackup(id: string): Promise<void> {
  await apiClient.delete(`/admin/backups/${id}`)
}

export async function getDownloadURL(id: string): Promise<{ url: string }> {
  const { data } = await apiClient.get<{ url: string }>(`/admin/backups/${id}/download-url`)
  return data
}

// Restore
export async function restoreBackup(id: string, password: string): Promise<BackupRecord> {
  const { data } = await apiClient.post<BackupRecord>(`/admin/backups/${id}/restore`, { password })
  return data
}

export const backupAPI = {
  getS3Config,
  updateS3Config,
  testS3Connection,
  getSchedule,
  updateSchedule,
  createBackup,
  downloadLocalBackup,
  listBackups,
  getBackup,
  deleteBackup,
  getDownloadURL,
  restoreBackup,
}

export default backupAPI
