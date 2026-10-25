import { ref } from 'vue'

/**
 * Navigation layout preference.
 *
 * Mirrors `useTheme` on purpose: a module-level singleton plus a localStorage
 * override, so switching layouts is instant, survives reloads, and stays
 * completely independent of the light/dark theme.
 */

export type NavLayout = 'scroll' | 'tabs'

const STORAGE_KEY = 'xn-layout'
const DEFAULT_LAYOUT: NavLayout = 'scroll'

const current = ref<NavLayout>(DEFAULT_LAYOUT)

/** True when the visitor has chosen a layout explicitly. */
export function hasLayoutOverride(): boolean {
  const stored = localStorage.getItem(STORAGE_KEY)
  return stored === 'scroll' || stored === 'tabs'
}

/**
 * Resolves the effective layout.
 *
 * Precedence: an explicit local choice, then the site default from the admin
 * settings, then the built-in default (scroll).
 */
export function resolveLayout(stored: string | null, preferred?: string | null): NavLayout {
  if (stored === 'scroll' || stored === 'tabs') return stored
  if (preferred === 'tabs') return 'tabs'
  if (preferred === 'scroll') return 'scroll'
  return DEFAULT_LAYOUT
}

/** Applies a layout without persisting it (used when loading site settings). */
export function applyLayout(preferred?: string | null): void {
  current.value = resolveLayout(localStorage.getItem(STORAGE_KEY), preferred)
}

/** Switches the layout and remembers the choice locally. */
export function setLayout(layout: NavLayout): void {
  localStorage.setItem(STORAGE_KEY, layout)
  current.value = layout
}

/** Clears the local choice so the site default applies again. */
export function clearLayoutOverride(): void {
  localStorage.removeItem(STORAGE_KEY)
}

export function useLayout() {
  return { current, applyLayout, setLayout, clearLayoutOverride, hasLayoutOverride }
}
