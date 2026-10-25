import { describe, expect, it } from 'vitest'
import { ICON_SOURCE_LABELS, ICON_TYPE_LABELS, iconSourceLabel, iconTypeLabel, isManualIconType } from '@/utils/icons'

/**
 * The manual/automatic boundary is a product rule, not a display detail: the
 * automatic fetcher must never overwrite an icon the user chose.
 */
describe('isManualIconType', () => {
  it('protects every icon type the user can set', () => {
    for (const type of ['url', 'upload', 'simple', 'text'] as const) {
      expect(isManualIconType(type)).toBe(true)
    }
  })

  it('allows the automatic fetcher to work on the remaining types', () => {
    for (const type of ['none', 'auto', 'favicon'] as const) {
      expect(isManualIconType(type)).toBe(false)
    }
  })
})

describe('icon labels', () => {
  it('labels every icon type', () => {
    for (const type of Object.keys(ICON_TYPE_LABELS)) {
      expect(iconTypeLabel(type as keyof typeof ICON_TYPE_LABELS)).not.toBe('')
    }
  })

  it('marks the brand guess as a guess', () => {
    // The requirement forbids presenting an unverified icon as official.
    expect(ICON_SOURCE_LABELS['brand-guess']).toContain('推测')
    expect(iconSourceLabel('brand-guess')).toContain('推测')
  })

  it('falls back to the raw value for unknown input', () => {
    expect(iconSourceLabel(undefined)).toBe('')
    expect(iconSourceLabel('unknown-source' as never)).toBe('unknown-source')
  })
})
