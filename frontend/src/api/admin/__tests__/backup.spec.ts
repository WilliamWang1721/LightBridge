import { describe, expect, it } from 'vitest'
import {
  LOCAL_BACKUP_COMPLETION_MARKER,
  readBlobAsArrayBuffer,
  validateLocalBackupDownload
} from '../backup'

describe('local backup completion marker', () => {
  it('keeps a successful concatenated gzip download unchanged', async () => {
    const primaryGzipMember = LOCAL_BACKUP_COMPLETION_MARKER.slice()
    const envelope = new Blob([primaryGzipMember, LOCAL_BACKUP_COMPLETION_MARKER])

    const backup = await validateLocalBackupDownload(envelope)

    expect(backup.type).toBe('application/gzip')
    expect(backup.size).toBe(envelope.size)
    expect(Array.from(new Uint8Array(await readBlobAsArrayBuffer(backup)))).toEqual(
      Array.from(new Uint8Array(await readBlobAsArrayBuffer(envelope)))
    )
  })

  it('rejects a truncated backup without the completion marker', async () => {
    const truncated = new Blob([new Uint8Array([0x1f, 0x8b, 0x08, 0x00])])

    await expect(validateLocalBackupDownload(truncated)).rejects.toThrow('Local backup stream is incomplete')
  })

  it('rejects a backup with a partial completion marker', async () => {
    const partialMarker = LOCAL_BACKUP_COMPLETION_MARKER.slice(0, -4)
    const truncated = new Blob([new Uint8Array([0x1f, 0x8b]), partialMarker])

    await expect(validateLocalBackupDownload(truncated)).rejects.toThrow('Local backup stream is incomplete')
  })
})
