import { apiClient } from './client'
import type { UIProfileOverrides } from '@/ui-platform/types'

interface UIProfileResponse {
  profile: UIProfileOverrides
}

export async function getUIProfile(): Promise<UIProfileOverrides> {
  const { data } = await apiClient.get<UIProfileResponse>('/user/ui-profile')
  return data.profile || {}
}

export async function updateUIProfile(profile: UIProfileOverrides): Promise<UIProfileOverrides> {
  const { data } = await apiClient.put<UIProfileResponse>('/user/ui-profile', profile)
  return data.profile || {}
}

export async function resetUIProfile(): Promise<void> {
  await apiClient.delete('/user/ui-profile')
}

export default {
  getUIProfile,
  updateUIProfile,
  resetUIProfile,
}
