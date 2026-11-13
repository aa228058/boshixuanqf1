import { afterEach, describe, expect, it } from 'vitest'
import { ask, confirmState, settle } from './confirm'

describe('confirm dialog runtime', () => {
  afterEach(() => {
    // Leave the module-level dialog state clean for the next test.
    settle(false)
  })

  it('resolves true when the user confirms', async () => {
    const result = ask('Delete this record?')
    expect(confirmState.visible).toBe(true)
    expect(confirmState.message).toBe('Delete this record?')

    settle(true)

    await expect(result).resolves.toBe(true)
    expect(confirmState.visible).toBe(false)
    expect(confirmState.resolve).toBeUndefined()
  })

  it('resolves false when the user cancels', async () => {
    const result = ask('Delete this record?')
    settle(false)

    await expect(result).resolves.toBe(false)
    expect(confirmState.visible).toBe(false)
  })

  it('cancels the previous request if another dialog is opened', async () => {
    const first = ask('First request')
    const second = ask('Second request')

    await expect(first).resolves.toBe(false)
    expect(confirmState.message).toBe('Second request')

    settle(true)

    await expect(second).resolves.toBe(true)
  })
})
