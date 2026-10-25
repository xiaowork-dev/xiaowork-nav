/**
 * Pure helpers behind the scroll layout's category highlight.
 *
 * They are kept free of DOM access so they can be unit tested: the component
 * only measures element positions and feeds the numbers in.
 */

/**
 * Picks the category that owns the current reading position.
 *
 * @param tops     Distance of each category heading from the viewport top, in document order.
 * @param offset   Distance from the viewport top that counts as "being read".
 * @param atBottom When the page is scrolled to the very end, the last category wins
 *                 (otherwise a short final section could never become active).
 * @returns The index of the active category, or -1 when there is none.
 */
export function pickActiveIndex(tops: number[], offset: number, atBottom = false): number {
  if (tops.length === 0) return -1
  if (atBottom) return tops.length - 1

  let current = 0
  for (let i = 0; i < tops.length; i += 1) {
    if (tops[i] <= offset) current = i
    else break
  }
  return current
}

/** Reports whether the viewport is within `tolerance` pixels of the document end. */
export function isAtBottom(
  scrollY: number,
  innerHeight: number,
  scrollHeight: number,
  tolerance = 4,
): boolean {
  return innerHeight + scrollY >= scrollHeight - tolerance
}

/** Clamps a target scroll position so a jump never scrolls above the document start. */
export function clampScrollTop(top: number): number {
  return Math.max(0, top)
}
