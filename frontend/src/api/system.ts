/** System endpoints: health and version. */
import { api } from './client'
import type { HealthResponse, VersionInfo } from '@/types/api'

export function getHealth(): Promise<HealthResponse> {
  return api.get<HealthResponse>('/healthz')
}

export function getVersion(): Promise<VersionInfo> {
  return api.get<VersionInfo>('/api/v1/version')
}
