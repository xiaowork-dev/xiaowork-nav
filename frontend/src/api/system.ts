/** System endpoints: health, version, auth status. */
import { api } from './client'
import type { AuthStatus, HealthResponse, VersionInfo } from '@/types/api'

export function getHealth(): Promise<HealthResponse> {
  return api.get<HealthResponse>('/healthz')
}

export function getVersion(): Promise<VersionInfo> {
  return api.get<VersionInfo>('/api/v1/version')
}

export function getAuthStatus(): Promise<AuthStatus> {
  return api.get<AuthStatus>('/api/v1/auth/status')
}
