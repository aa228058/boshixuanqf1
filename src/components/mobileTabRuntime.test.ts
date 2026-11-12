import { describe, expect, it } from 'vitest'
import { isMobileTabActive, mobileTabPath } from './mobileTabRuntime'

describe('mobile tab routes', () => {
  it('falls back to account management when no account is selected', () => {
    expect(mobileTabPath('broadcast', null)).toBe('/accounts')
    expect(mobileTabPath('tasks', '')).toBe('/accounts')
  })

  it('keeps workspace tabs active across account switches', () => {
    expect(isMobileTabActive('broadcast', '/workspace/old-account/broadcast')).toBe(true)
    expect(isMobileTabActive('broadcast', '/workspace/new-account/broadcast')).toBe(true)
    expect(isMobileTabActive('tasks', '/workspace/new-account/tasks')).toBe(true)
    expect(isMobileTabActive('tasks', '/workspace/new-account/tasks/123')).toBe(true)
  })

  it('does not mark workspace tabs active on account management fallback', () => {
    expect(isMobileTabActive('broadcast', '/accounts')).toBe(false)
    expect(isMobileTabActive('tasks', '/accounts')).toBe(false)
    expect(isMobileTabActive('accounts', '/accounts')).toBe(true)
  })

  it('matches fixed tabs exactly or by nested route segment', () => {
    expect(isMobileTabActive('home', '/dashboard')).toBe(true)
    expect(isMobileTabActive('home', '/dashboard/child')).toBe(false)
    expect(isMobileTabActive('more', '/more/help')).toBe(true)
  })
})
