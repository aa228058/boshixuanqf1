export function toggleSelectionByKey<T>(
  selected: readonly T[],
  candidates: readonly T[],
  getKey: (item: T) => string,
): T[] {
  if (candidates.length === 0) return [...selected]

  // Build the lookup once: using selected.some() inside candidates.every()
  // becomes quadratic for large group/contact libraries.
  const selectedKeys = new Set(selected.map(getKey))
  const candidateKeys = new Set(candidates.map(getKey))
  const allSelected = candidates.every(item => selectedKeys.has(getKey(item)))

  if (allSelected) {
    return selected.filter(item => !candidateKeys.has(getKey(item)))
  }

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
