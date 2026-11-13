/** Whether a Lottie animation should advance under the current motion preference. */
export function shouldPlayLottie(autoplay: boolean, prefersReducedMotion: boolean): boolean {
  return autoplay && !prefersReducedMotion
}
