import { ref } from 'vue'
import type { Theme } from '@/types/api'

type Resolved = 'light' | 'dark'

const STORAGE_KEY = 'xn-theme'
const current = ref<Resolved>('dark')

function systemPreference(): Resolved {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function resolve(theme: Theme): Resolved {
  if (theme === 'auto') return systemPreference()
  return theme
}

function apply(resolved: Resolved): void {
  current.value = resolved
  document.documentElement.dataset.theme = resolved
}

/** Applies a theme; a local override (if set) wins over the server setting. */
export function applyTheme(theme: Theme): void {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'light' || stored === 'dark') {
    apply(stored)
    return
  }
  apply(resolve(theme))
}

/** Toggles between light and dark and persists the choice locally. */
export function toggleTheme(): void {
  const next: Resolved = current.value === 'dark' ? 'light' : 'dark'
  localStorage.setItem(STORAGE_KEY, next)
  apply(next)
}

/** Clears the local override so the server setting takes effect again. */
export function clearThemeOverride(): void {
  localStorage.removeItem(STORAGE_KEY)
}

export function useTheme() {
  return { current, applyTheme, toggleTheme, clearThemeOverride }
}
