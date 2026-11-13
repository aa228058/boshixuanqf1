/**
 * A pull-to-refresh gesture may only start when the active scroll surface is
 * already at its top edge. Reject non-finite positions defensively.
 */
export function canStartPullRefresh(scrollTop: number, refreshing: boolean): boolean {
  return !refreshing && Number.isFinite(scrollTop) && scrollTop <= 0
}
