import type { VisualTheme } from '@/composables/useTheme'

/**
 * Lifecycle of the opt-in ASCII Pond background.
 *
 * The upstream engine is a plain ES module with side effects on `window` and a
 * `destroy()` that removes them again, so the only thing this project owns is
 * *when* an instance exists. That timing is the fragile part: the module is
 * fetched lazily, and a visitor can switch themes several times before the
 * fetch resolves. Keeping the generation counter here (instead of in the
 * component) makes the races testable without a DOM.
 */

export interface PondInstance {
  destroy: () => void
  canvas?: Element
}

export interface PondModule {
  mount: (target: Element, options: PondOptions) => PondInstance
}

export type PondOptions = Record<string, unknown>

export interface PondControllerDeps {
  /** Loads the engine module (a dynamic import in the app). */
  load: () => Promise<PondModule>
  /** Where to mount; called only once the module has loaded. */
  target: () => Element
  /** Mount options, resolved at mount time so they can follow the viewport. */
  options: () => PondOptions
  /** Called with every instance that survives the race, e.g. to tag its canvas. */
  onMount?: (instance: PondInstance) => void
}

export interface PondController {
  /** Apply a theme: mounts for `pond`, tears down for anything else. */
  sync: (theme: VisualTheme) => void
  /** Final teardown; later `sync` calls are ignored. */
  dispose: () => void
}

export function createPondController(deps: PondControllerDeps): PondController {
  let instance: PondInstance | undefined
  let generation = 0
  let disposed = false

  function teardown(): void {
    instance?.destroy()
    instance = undefined
  }

  function sync(theme: VisualTheme): void {
    const token = ++generation
    teardown()
    if (disposed || theme !== 'pond') return
    void deps
      .load()
      .then((module) => {
        // A newer sync (or unmount) already won: never mount a stale engine.
        if (disposed || token !== generation) return
        const next = module.mount(deps.target(), deps.options())
        if (disposed || token !== generation) {
          next.destroy()
          return
        }
        instance = next
        deps.onMount?.(next)
      })
      .catch((error: unknown) => {
        // Decorative only: a failed engine must never break the navigation.
        console.warn('ASCII Pond background could not start:', error)
      })
  }

  return {
    sync,
    dispose() {
      disposed = true
      ++generation
      teardown()
    },
  }
}
