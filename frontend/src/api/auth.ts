/** Authentication endpoints. */
import { api } from './client'
import type { AuthStatus, User } from '@/types/api'

export function getAuthStatus(): Promise<AuthStatus> {
  return api.get<AuthStatus>('/api/v1/auth/status')
}

export function setup(username: string, password: string): Promise<{ user: User }> {
  return api.post<{ user: User }>('/api/v1/setup', { username, password })
}

export function login(username: string, password: string): Promise<{ user: User }> {
  return api.post<{ user: User }>('/api/v1/auth/login', { username, password })
}

export function logout(): Promise<void> {
  return api.post<void>('/api/v1/auth/logout')
}

export function me(): Promise<User> {
  return api.get<User>('/api/v1/auth/me')
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return api.put<void>('/api/v1/auth/password', {
    current_password: currentPassword,
    new_password: newPassword,
  })
}
