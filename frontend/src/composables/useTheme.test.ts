import { beforeEach, describe, expect, it, vi } from 'vitest'
import { applyTheme, clearThemeOverride, setTheme, toggleTheme, useTheme } from '@/composables/useTheme'

/**
 * Theme preference rules.
 *
 * Three visual themes share one storage key: an explicit local choice always
 * wins over the server setting, and the two-theme values written by earlier
 * versions keep working.
 */

const store = new Map<string, string>()
const localStorageStub = {
  getItem: (key: string) => store.get(key) ?? null,
  setItem: (key: string, value: string) => void store.set(key, value),
  removeItem: (key: string) => void store.delete(key),
  clear: () => store.clear(),
}

const dataset: Record<string, string> = {}
let systemDark = false

beforeEach(() => {
  store.clear()
  for (const key of Object.keys(dataset)) delete dataset[key]
  vi.stubGlobal('localStorage', localStorageStub)
  vi.stubGlobal('document', { documentElement: { dataset } })
  vi.stubGlobal('window', {
    matchMedia: (query: string) => ({ matches: query.includes('dark') ? systemDark : !systemDark }),
  })
  systemDark = false
})

describe('useTheme', () => {
  it('starts from the legacy default of dark', () => {
    expect(useTheme().current.value).toBe('dark')
  })

  it('applies and persists the pond theme', () => {
    setTheme('pond')
    expect(document.documentElement.dataset.theme).toBe('pond')
    expect(localStorage.getItem('xn-theme')).toBe('pond')
    expect(useTheme().current.value).toBe('pond')
  })

  it('restores the stored pond theme on the next visit', () => {
    localStorage.setItem('xn-theme', 'pond')
    applyTheme('light')
    expect(document.documentElement.dataset.theme).toBe('pond')
    expect(useTheme().current.value).toBe('pond')
  })

  it('keeps an explicit local choice ahead of the server setting', () => {
    setTheme('light')
    applyTheme('dark')
    expect(document.documentElement.dataset.theme).toBe('light')
  })

  it('honours the server setting when nothing is stored', () => {
    applyTheme('light')
    expect(document.documentElement.dataset.theme).toBe('light')
    applyTheme('dark')
    expect(document.documentElement.dataset.theme).toBe('dark')
  })

  it('resolves auto from the system preference', () => {
    systemDark = true
    applyTheme('auto')
    expect(document.documentElement.dataset.theme).toBe('dark')
    systemDark = false
    applyTheme('auto')
    expect(document.documentElement.dataset.theme).toBe('light')
  })

  it('ignores stored values it does not understand', () => {
    localStorage.setItem('xn-theme', 'sepia')
    applyTheme('dark')
    expect(document.documentElement.dataset.theme).toBe('dark')
  })

  it('keeps the legacy two-theme toggle working', () => {
    setTheme('dark')
    toggleTheme()
    expect(document.documentElement.dataset.theme).toBe('light')
    toggleTheme()
    expect(document.documentElement.dataset.theme).toBe('dark')
  })

  it('leaves pond through the legacy toggle without keeping a stale value', () => {
    setTheme('pond')
    toggleTheme()
    const value = localStorage.getItem('xn-theme')
    expect(value).not.toBe('pond')
    expect(document.documentElement.dataset.theme).toBe(value)
  })

  it('drops a cleared override so the server setting wins again', () => {
    setTheme('pond')
    clearThemeOverride()
    applyTheme('dark')
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(localStorage.getItem('xn-theme')).toBeNull()
  })

  it('lands on the last choice after rapid switching', () => {
    for (const theme of ['pond', 'dark', 'light', 'pond', 'light', 'pond'] as const) setTheme(theme)
    expect(document.documentElement.dataset.theme).toBe('pond')
    expect(localStorage.getItem('xn-theme')).toBe('pond')
    expect(useTheme().current.value).toBe('pond')
  })
})
