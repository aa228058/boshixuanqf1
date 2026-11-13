import { describe, expect, it } from 'vitest'
import { shouldPlayLottie } from './lottiePlayback'

describe('shouldPlayLottie', () => {
  it('plays only when autoplay is enabled and reduced motion is not requested', () => {
    expect(shouldPlayLottie(true, false)).toBe(true)
    expect(shouldPlayLottie(true, true)).toBe(false)
    expect(shouldPlayLottie(false, false)).toBe(false)
    expect(shouldPlayLottie(false, true)).toBe(false)
  })
})
