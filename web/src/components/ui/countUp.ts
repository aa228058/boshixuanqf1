// Parse a numeric value while tolerating common currency prefixes, grouping separators and units.
const NUMBER_PATTERN = /[+-]?(?:\d{1,3}(?:,\d{3})+|\d+|\.\d+)(?:\.\d*)?(?:e[+-]?\d+)?/i

export function parseCountValue(value: unknown): number {
  if (typeof value === 'number') return Number.isFinite(value) ? value : 0
  const match = String(value ?? '').match(NUMBER_PATTERN)
  if (!match) return 0
  const parsed = Number(match[0].replace(/,/g, ''))
  return Number.isFinite(parsed) ? parsed : 0
}

// Keep the caller's currency/sign prefix and unit suffix while the numeric part animates.
export function formatCountValue(value: unknown, amount: number): string {
  if (typeof value === 'number') return String(amount)
  const source = String(value ?? '')
  const match = source.match(NUMBER_PATTERN)
  if (!match) return source
  const raw = match[0]
  const fractionDigits = Math.min(6, (raw.split(/[eE]/)[0].split('.')[1] || '').length)
  const grouped = raw.includes(',')
  const number = grouped
    ? new Intl.NumberFormat('en-US', {
        useGrouping: true,
        minimumFractionDigits: fractionDigits,
        maximumFractionDigits: fractionDigits,
      }).format(amount)
    : String(Number(amount.toFixed(fractionDigits)))
  return source.slice(0, match.index) + number + source.slice((match.index || 0) + raw.length)
}
