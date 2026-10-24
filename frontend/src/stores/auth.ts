import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as authApi from '@/api/auth'
import type { User } from '@/types/api'

/** Authentication and session state. */
export const useAuthStore = defineStore('auth', () => {
  const loaded = ref(false)
  const initialized = ref(false)
  const authenticated = ref(false)
  const user = ref<User | null>(null)

  async function refresh(): Promise<void> {
    try {
      const status = await authApi.getAuthStatus()
      initialized.value = status.initialized
      authenticated.value = status.authenticated
      user.value = status.user ?? null
    } finally {
      loaded.value = true
    }
  }

  async function signIn(username: string, password: string): Promise<void> {
    const result = await authApi.login(username, password)
    user.value = result.user
    authenticated.value = true
    initialized.value = true
  }

  async function signOut(): Promise<void> {
    await authApi.logout()
    user.value = null
    authenticated.value = false
  }

  async function setupAdmin(username: string, password: string): Promise<void> {
    const result = await authApi.setup(username, password)
    user.value = result.user
    authenticated.value = true
    initialized.value = true
  }

  return { loaded, initialized, authenticated, user, refresh, signIn, signOut, setupAdmin }
})
