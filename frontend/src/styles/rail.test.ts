import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

/**
 * Guards the category rail's layout contract.
 *
 * The rail must read as part of the page shell (a full-height column on the
 * page edge), not as a floating card inside the centred content container, and
 * the widths must stay in the agreed range. These are easy to regress with a
 * stray CSS edit, so they are asserted here rather than only reviewed visually.
 */
const css = readFileSync(fileURLToPath(new URL('./main.css', import.meta.url)), 'utf8')

function ruleBody(selector: string): string {
  const match = css.match(new RegExp(`\\${selector}\\s*\\{([^}]*)\\}`, 's'))
  if (!match) throw new Error(`rule ${selector} not found in main.css`)
  return match[1]
}

describe('category rail layout', () => {
  it('keeps the expanded rail within the requested 200-220px range', () => {
    const width = Number(css.match(/--rail-width:\s*(\d+)px/)?.[1])
    expect(Number.isFinite(width)).toBe(true)
    expect(width).toBeGreaterThanOrEqual(200)
    expect(width).toBeLessThanOrEqual(220)
  })

  it('keeps the collapsed rail an icon-only strip', () => {
    const width = Number(css.match(/--rail-width-collapsed:\s*(\d+)px/)?.[1])
    expect(width).toBeGreaterThan(0)
    expect(width).toBeLessThan(80)
  })

  it('is a full-height sticky column rather than a floating card', () => {
    const rail = ruleBody('.rail')
    expect(rail).toMatch(/position:\s*sticky/)
    expect(rail).toMatch(/height:\s*100vh/)
    expect(rail).not.toMatch(/box-shadow/)
    expect(rail).not.toMatch(/border-radius/)
  })

  it('lets the scroll layout break out of the centred page container', () => {
    const page = ruleBody('.page--rail')
    expect(page).toMatch(/width:\s*100%/)
    expect(page).toMatch(/max-width:\s*none/)
    expect(page).toMatch(/padding:\s*0/)
  })

  it('re-centres the content column instead of leaving a gap', () => {
    const inner = ruleBody('.rail-main__inner')
    expect(inner).toMatch(/margin:\s*0 auto/)
    expect(inner).toMatch(/width:\s*min\(100%,\s*var\(--page-width\)\)/)
  })

  it('hides the rail on phones and shows the chip bar instead', () => {
    const mobile = css.slice(css.indexOf('@media (max-width: 700px)'))
    expect(mobile).toMatch(/\.rail\s*\{\s*display:\s*none/)
    expect(mobile).toMatch(/\.chipbar\s*\{[^}]*display:\s*flex/s)
  })

  it('collapses the rail to icons on medium widths', () => {
    const tablet = css.slice(css.indexOf('@media (max-width: 1080px)'), css.indexOf('@media (max-width: 1000px)'))
    expect(tablet).toMatch(/--rail-width-collapsed/)
    expect(tablet).toMatch(/\.rail__label,[\s\S]*?display:\s*none/)
  })
})
