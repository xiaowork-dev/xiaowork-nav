import { describe, expect, it } from 'vitest'
import { resolveLayout } from '@/composables/useLayout'

/**
 * Layout resolution rules (requirement 4.6).
 *
 * Precedence: an explicit visitor choice, then the site default, then scroll.
 * Theme is a separate preference and is deliberately not involved here.
 */
describe('resolveLayout', () => {
  it('defaults new visitors to the scroll layout', () => {
    expect(resolveLayout(null, null)).toBe('scroll')
    expect(resolveLayout(null, undefined)).toBe('scroll')
    expect(resolveLayout(null, '')).toBe('scroll')
  })

  it('honours the site default when the visitor has not chosen', () => {
    expect(resolveLayout(null, 'tabs')).toBe('tabs')
    expect(resolveLayout(null, 'scroll')).toBe('scroll')
  })

  it('lets an explicit visitor choice win over the site default', () => {
    expect(resolveLayout('scroll', 'tabs')).toBe('scroll')
    expect(resolveLayout('tabs', 'scroll')).toBe('tabs')
  })

  it('ignores unknown stored values', () => {
    expect(resolveLayout('grid', 'tabs')).toBe('tabs')
    expect(resolveLayout('carousel', null)).toBe('scroll')
  })

  it('ignores unknown site defaults', () => {
    expect(resolveLayout(null, 'carousel')).toBe('scroll')
  })
})
