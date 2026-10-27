import { ref } from 'vue'
import type { Theme } from '@/types/api'

export type VisualTheme = 'light' | 'dark' | 'pond'
type Resolved = 'light' | 'dark'
const STORAGE_KEY = 'xn-theme'
const current = ref<VisualTheme>('dark')

function systemPreference(): Resolved {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}
function apply(resolved: VisualTheme): void {
  current.value = resolved
  document.documentElement.dataset.theme = resolved
}
/** Browser-only Pond theme is independent of the server API's light/dark/auto settings. */
export function applyTheme(theme: Theme): void {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'light' || stored === 'dark' || stored === 'pond') {
    apply(stored)
  } else {
    apply(theme === 'auto' ? systemPreference() : theme)
  }
}
export function setTheme(theme: VisualTheme): void {
  localStorage.setItem(STORAGE_KEY, theme)
  apply(theme)
}
/** Keeps the original two-way toggle behavior for legacy consumers. */
export function toggleTheme(): void {
  setTheme(current.value === 'dark' ? 'light' : 'dark')
}
export function clearThemeOverride(): void {
  localStorage.removeItem(STORAGE_KEY)
}
export function useTheme() {
  return { current, applyTheme, setTheme, toggleTheme, clearThemeOverride }
}
