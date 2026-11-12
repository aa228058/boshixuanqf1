// Parse a numeric value while tolerating common currency prefixes, grouping separators and units.
export function parseCountValue(value: unknown): number {
  if (typeof value === 'number') return Number.isFinite(value) ? value : 0
  const normalized = String(value ?? '').replace(/,(?=\d{3}(?:\D|$))/g, '')
  const match = normalized.match(/[+-]?(?:\d+\.?\d*|\.\d+)(?:e[+-]?\d+)?/i)
  if (!match) return 0
  const parsed = Number(match[0])
  return Number.isFinite(parsed) ? parsed : 0
}
