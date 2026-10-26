import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

/**
 * Guards the scroll layout's application-shell contract.
 *
 * The rail and the content must live in one grid with a single horizontal
 * rhythm. An earlier version centred a fixed-width container inside the main
 * column, which left a large dead gap between the rail and the content on wide
 * screens and made the page look like two panels stitched together — these
 * assertions exist so that cannot come back.
 */
const css = readFileSync(fileURLToPath(new URL('./main.css', import.meta.url)), 'utf8')

function ruleBody(selector: string): string {
  const match = css.match(new RegExp(`\\${selector}\\s*\\{([^}]*)\\}`, 's'))
  if (!match) throw new Error(`rule ${selector} not found in main.css`)
  return match[1]
}

describe('scroll layout shell', () => {
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

  it('lets the shell span the whole page', () => {
    const page = ruleBody('.page--rail')
    expect(page).toMatch(/width:\s*100%/)
    expect(page).toMatch(/max-width:\s*none/)
    expect(page).toMatch(/padding:\s*0/)
  })

  it('puts the rail and the main column in one grid', () => {
    const shell = ruleBody('.shell')
    expect(shell).toMatch(/display:\s*grid/)
    expect(shell).toMatch(/grid-template-columns:\s*var\(--rail-width\)\s+minmax\(0,\s*1fr\)/)
  })

  it('never re-centres the content inside the main column', () => {
    // The regression this guards: a centred, width-capped container in the main
    // column leaves dead space between the rail and the content.
    const content = ruleBody('.shell__content')
    expect(content).not.toMatch(/margin:\s*0\s+auto/)
    expect(content).not.toMatch(/margin-inline:\s*auto/)
    expect(content).not.toMatch(/max-width/)
    expect(content).not.toMatch(/width:\s*min\(100%/)
  })

  it('gives the top bar and the content the same horizontal gutter', () => {
    expect(ruleBody('.shell__topbar')).toMatch(/padding:\s*0\s+var\(--shell-gutter\)/)
    expect(ruleBody('.shell__content')).toMatch(/padding:\s*[^;]*var\(--shell-gutter\)/)
  })

  it('hides the rail on phones, shows the brand and the chip bar instead', () => {
    const mobile = css.slice(css.indexOf('@media (max-width: 700px)'))
    expect(mobile).toMatch(/\.rail\s*\{\s*display:\s*none/)
    expect(mobile).toMatch(/\.shell__brand\s*\{\s*display:\s*flex/)
    expect(mobile).toMatch(/\.chipbar\s*\{[^}]*display:\s*flex/s)
  })

  it('collapses the rail to icons on medium widths', () => {
    const tablet = css.slice(css.indexOf('@media (max-width: 1080px)'), css.indexOf('@media (max-width: 1000px)'))
    expect(tablet).toMatch(/--rail-width-collapsed/)
    expect(tablet).toMatch(/\.rail__label,[\s\S]*?display:\s*none/)
  })
})

describe('rail theming', () => {
  it('derives the rail from a translucent layer instead of a solid slab', () => {
    const dark = css.slice(css.indexOf(":root[data-theme='dark']"), css.indexOf(":root[data-theme='light']"))
    const light = css.slice(css.indexOf(":root[data-theme='light']"))
    // Both themes use a low-opacity overlay.
    expect(dark).toMatch(/--rail-bg:\s*rgba\([^)]*0\.\d+\)/)
    expect(light).toMatch(/--rail-bg:\s*rgba\(255,\s*255,\s*255/)
  })

  it('does not paint a dark block in the light theme', () => {
    const light = css.slice(css.indexOf(":root[data-theme='light']"), css.indexOf(':root {'))
    expect(light).not.toMatch(/--rail-bg:\s*rgba\((?:[0-9]|[1-9][0-9]),/)
  })

  it('uses a soft translucent highlight for the active category', () => {
    for (const theme of ["data-theme='dark'", "data-theme='light'"]) {
      const block = css.slice(css.indexOf(`:root[${theme}]`))
      expect(block).toMatch(/--rail-active:\s*rgba\([^)]*0\.\d+\)/)
    }
  })
})
