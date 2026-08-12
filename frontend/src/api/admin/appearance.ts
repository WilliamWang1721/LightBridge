import { apiClient } from '../client'
import type { RuntimeAppearanceSettings } from '@/types'

interface AppearanceResponse {
  appearance: RuntimeAppearanceSettings | null
}

export async function getRuntimeAppearance(): Promise<RuntimeAppearanceSettings | null> {
  const { data } = await apiClient.get<AppearanceResponse>('/admin/settings/appearance')
  return data?.appearance || null
}

export async function updateRuntimeAppearance(presetCode: string): Promise<RuntimeAppearanceSettings> {
  const { data } = await apiClient.put<AppearanceResponse>('/admin/settings/appearance', { preset_code: presetCode })
  if (!data?.appearance) throw new Error('Runtime appearance was not returned')
  return data.appearance
}

export async function resetRuntimeAppearance(): Promise<void> {
  await apiClient.delete('/admin/settings/appearance')
}

const appearanceAPI = {
  getRuntimeAppearance,
  updateRuntimeAppearance,
  resetRuntimeAppearance,
}

export default appearanceAPI
