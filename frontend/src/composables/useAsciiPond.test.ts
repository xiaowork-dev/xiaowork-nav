import { describe, expect, it, vi } from 'vitest'
import { createPondController, type PondInstance, type PondModule } from '@/composables/useAsciiPond'

/**
 * Mount/unmount rules for the opt-in ASCII Pond background.
 *
 * The engine is loaded lazily, so a visitor can switch themes while the module
 * is still in flight. These tests pin down the guarantees: one live instance at
 * most, nothing mounted for classic themes, every instance destroyed when the
 * theme changes or the page unmounts.
 */

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

function fakeInstance(): PondInstance & { destroyed: number } {
  const instance = {
    destroyed: 0,
    canvas: undefined as Element | undefined,
    destroy() {
      instance.destroyed++
    },
  }
  return instance
}

function setup() {
  const instances: (PondInstance & { destroyed: number })[] = []
  const gate = deferred<PondModule>()
  const mount = vi.fn(() => {
    const instance = fakeInstance()
    instances.push(instance)
    return instance
  })
  const load = vi.fn(() => gate.promise)
  const attach = vi.fn()
  const controller = createPondController({
    load,
    target: () => ({}) as Element,
    options: () => ({ theme: 'dark' }),
    onMount: attach,
  })
  return { controller, gate, mount, load, instances, attach }
}

describe('createPondController', () => {
  it('does not load the engine for the classic themes', () => {
    const { controller, load } = setup()
    controller.sync('dark')
    controller.sync('light')
    expect(load).not.toHaveBeenCalled()
  })

  it('loads the engine and mounts one instance for pond', async () => {
    const { controller, gate, mount, load, attach } = setup()
    controller.sync('pond')
    gate.resolve({ mount })
    await gate.promise
    await Promise.resolve()
    expect(load).toHaveBeenCalledTimes(1)
    expect(mount).toHaveBeenCalledTimes(1)
    expect(attach).toHaveBeenCalledTimes(1)
  })

  it('mounts once even if the same theme is applied twice', async () => {
    const { controller, gate, mount } = setup()
    controller.sync('pond')
    controller.sync('pond')
    gate.resolve({ mount })
    await gate.promise
    await Promise.resolve()
    // Only the newest sync adopts the module; the earlier one is stale.
    expect(mount).toHaveBeenCalledTimes(1)
  })

  it('destroys the instance when the visitor leaves the pond theme', async () => {
    const { controller, gate, mount, instances } = setup()
    controller.sync('pond')
    gate.resolve({ mount })
    await gate.promise
    await Promise.resolve()
    expect(instances).toHaveLength(1)
    controller.sync('dark')
    expect(instances[0].destroyed).toBe(1)
  })

  it('survives rapid switching and ends with exactly one live instance', async () => {
    const { controller, gate, mount, instances } = setup()
    for (const theme of ['pond', 'dark', 'pond', 'light', 'pond'] as const) controller.sync(theme)
    gate.resolve({ mount })
    await gate.promise
    await Promise.resolve()
    expect(instances).toHaveLength(1)
    expect(instances.filter((i) => i.destroyed === 0)).toHaveLength(1)
  })

  it('never mounts a module that resolves after the last switch away', async () => {
    const { controller, gate, mount } = setup()
    controller.sync('pond')
    controller.sync('light')
    gate.resolve({ mount })
    await gate.promise
    await Promise.resolve()
    expect(mount).not.toHaveBeenCalled()
  })

  it('destroys the instance on unmount and ignores later syncs', async () => {
    const { controller, gate, mount, instances } = setup()
    controller.sync('pond')
    gate.resolve({ mount })
    await gate.promise
    await Promise.resolve()
    controller.dispose()
    expect(instances[0].destroyed).toBe(1)
    controller.sync('pond')
    await Promise.resolve()
    expect(instances).toHaveLength(1)
  })

  it('never mounts when the component unmounts before the module resolves', async () => {
    const { controller, gate, mount } = setup()
    controller.sync('pond')
    controller.dispose()
    gate.resolve({ mount })
    await gate.promise
    await Promise.resolve()
    expect(mount).not.toHaveBeenCalled()
  })

  it('stays silent when the engine fails to load', async () => {
    const { controller, gate, mount } = setup()
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    controller.sync('pond')
    gate.reject(new Error('404'))
    await gate.promise.catch(() => {})
    await Promise.resolve()
    expect(mount).not.toHaveBeenCalled()
    expect(warn).toHaveBeenCalled()
    warn.mockRestore()
  })
})
