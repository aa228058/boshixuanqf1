import { beforeEach, describe, expect, it, vi } from 'vitest'
import { emitMutation, onMutation } from './mutationBus'

describe('mutationBus subscriptions', () => {
  beforeEach(() => {
    // Each test owns and releases its subscriptions; no module-level reset is needed.
  })

  it('global and sticky regular expressions match every broadcast', () => {
    const invalidate = vi.fn()
    const unsubscribe = onMutation(/^\/accounts\//g, invalidate)

    emitMutation('POST', '/accounts/1')
    emitMutation('PUT', '/accounts/2')

    expect(invalidate).toHaveBeenCalledTimes(2)
    unsubscribe()
  })

  it('unsubscribe prevents future callbacks', () => {
    const invalidate = vi.fn()
    const unsubscribe = onMutation(/\/tasks\//, invalidate)

    emitMutation('POST', '/tasks/1')
    unsubscribe()
    emitMutation('DELETE', '/tasks/1')

    expect(invalidate).toHaveBeenCalledTimes(1)
  })

  it('a callback can unsubscribe itself during a broadcast', () => {
    const invalidate = vi.fn()
    let unsubscribe = () => {}
    unsubscribe = onMutation(/.*/, () => {
      invalidate()
      unsubscribe()
    })

    emitMutation('PATCH', '/resource')
    emitMutation('PATCH', '/resource')

    expect(invalidate).toHaveBeenCalledTimes(1)
  })

  it('one throwing subscriber does not block later subscribers', () => {
    const broken = onMutation(/.*/, () => { throw new Error('subscriber failed') })
    const invalidate = vi.fn()
    const unsubscribe = onMutation(/.*/, invalidate)

    expect(() => emitMutation('POST', '/resource')).not.toThrow()
    expect(invalidate).toHaveBeenCalledTimes(1)
    broken()
    unsubscribe()
  })

  it('GET requests do not broadcast', () => {
    const invalidate = vi.fn()
    const unsubscribe = onMutation(/.*/, invalidate)

    emitMutation('GET', '/resource')

    expect(invalidate).not.toHaveBeenCalled()
    unsubscribe()
  })
})
