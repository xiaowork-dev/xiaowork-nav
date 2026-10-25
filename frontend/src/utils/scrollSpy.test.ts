import { describe, expect, it } from 'vitest'
import { clampScrollTop, isAtBottom, pickActiveIndex } from '@/utils/scrollSpy'

/**
 * The highlight rule used by the scroll layout's category rail.
 *
 * `tops` are the distances from the viewport top, in document order; headings
 * above the reading offset have negative values.
 */
describe('pickActiveIndex', () => {
  it('returns -1 when there are no categories', () => {
    expect(pickActiveIndex([], 120)).toBe(-1)
  })

  it('selects the last heading that has passed the offset', () => {
    // Only the first two headings are above the reading line.
    expect(pickActiveIndex([-500, -120, 200], 120)).toBe(1)
    expect(pickActiveIndex([-500, -40, 200], 120)).toBe(1)
  })

  it('selects the last heading when every heading has passed the offset', () => {
    expect(pickActiveIndex([-500, -120, 40], 120)).toBe(2)
  })

  it('keeps the first category while the page is still at the top', () => {
    expect(pickActiveIndex([300, 900, 1500], 120)).toBe(0)
  })

  it('treats a heading exactly at the offset as reached', () => {
    expect(pickActiveIndex([-10, 120, 400], 120)).toBe(1)
  })

  it('prefers the last category when the page is scrolled to the bottom', () => {
    // Without this rule a short final section could never become active.
    expect(pickActiveIndex([-900, -400, 200], 120, true)).toBe(2)
  })

  it('never returns an out-of-range index', () => {
    const tops = [-100, -50, 10]
    const index = pickActiveIndex(tops, 120)
    expect(index).toBeGreaterThanOrEqual(0)
    expect(index).toBeLessThan(tops.length)
  })
})

describe('isAtBottom', () => {
  it('detects the end of the document within the tolerance', () => {
    expect(isAtBottom(1000, 800, 1800)).toBe(true)
    expect(isAtBottom(996, 800, 1800)).toBe(true)
    expect(isAtBottom(1800, 800, 1800)).toBe(true)
  })

  it('reports false while there is more to scroll', () => {
    expect(isAtBottom(0, 800, 1800)).toBe(false)
    expect(isAtBottom(500, 800, 1800)).toBe(false)
  })

  it('handles documents shorter than the viewport', () => {
    expect(isAtBottom(0, 900, 600)).toBe(true)
  })
})

describe('clampScrollTop', () => {
  it('never scrolls above the document start', () => {
    expect(clampScrollTop(-40)).toBe(0)
    expect(clampScrollTop(0)).toBe(0)
    expect(clampScrollTop(320)).toBe(320)
  })
})
