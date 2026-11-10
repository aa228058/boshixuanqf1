export function toggleSelectionByKey<T>(
  selected: readonly T[],
  candidates: readonly T[],
  getKey: (item: T) => string,
): T[] {
  if (candidates.length === 0) return [...selected]

  const candidateKeys = new Set(candidates.map(getKey))
  const allSelected = candidates.every(item =>
    selected.some(current => getKey(current) === getKey(item)),
  )

  if (allSelected) {
    return selected.filter(item => !candidateKeys.has(getKey(item)))
  }

  const selectedKeys = new Set(selected.map(getKey))
  const next = [...selected]
  for (const item of candidates) {
    const key = getKey(item)
    if (!selectedKeys.has(key)) {
      next.push(item)
      selectedKeys.add(key)
    }
  }
  return next
}
