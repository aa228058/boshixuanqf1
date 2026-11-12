export function normalizeRingSize(size: number): number {
  return Number.isFinite(size) ? Math.max(1, size) : 44
}

export function normalizeRingStroke(stroke: number, size: number): number {
  const safeSize = normalizeRingSize(size)
  const fallback = Math.min(4, safeSize)
  return Number.isFinite(stroke)
    ? Math.min(safeSize, Math.max(0, stroke))
    : fallback
}

export function normalizeRingPercent(percent: number): number {
  return Number.isFinite(percent) ? Math.min(100, Math.max(0, percent)) : 0
}

export function normalizeRingSuccess(success: number, percent: number): number {
  return Number.isFinite(success)
    ? Math.min(normalizeRingPercent(percent), Math.max(0, success))
    : 0
}
