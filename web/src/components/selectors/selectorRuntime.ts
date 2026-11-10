export function filterSelectorItems<T>(
  items: readonly T[],
  query: string,
  getSearchText: (item: T) => string,
): T[] {
  if (!query) return [...items]
  const normalizedQuery = query.toLowerCase()
  return items.filter(item => getSearchText(item).toLowerCase().includes(normalizedQuery))
}

export function areAllSelectorKeysSelected<T>(
  items: readonly T[],
  selectedKeys: ReadonlySet<string>,
  getKey: (item: T) => string,
): boolean {
  if (items.length === 0) return false
  for (const item of items) {
    if (!selectedKeys.has(getKey(item))) return false
  }
  return true
}

export function toggleAllSelectorKeys<T>(
  items: readonly T[],
  selectedKeys: ReadonlySet<string>,
  getKey: (item: T) => string,
): Set<string> {
  const next = new Set(selectedKeys)
  const remove = areAllSelectorKeysSelected(items, selectedKeys, getKey)
  for (const item of items) {
    const key = getKey(item)
    if (remove) next.delete(key)
    else next.add(key)
  }
  return next
}

export function createSelectorTitleLookup<
  TGroup extends { tg_chat_id: string | number; username?: string | null; title: string },
  TDialog extends { id: string; title: string },
>(
  groups: readonly TGroup[],
  dialogs: readonly TDialog[],
): Map<string, string> {
  const titles = new Map<string, string>()
  for (const group of groups) {
    titles.set(String(group.tg_chat_id), group.title)
    if (group.username) {
      titles.set(group.username, group.title)
      titles.set(`@${group.username}`, group.title)
    }
  }
  for (const dialog of dialogs) {
    if (!titles.has(dialog.id)) titles.set(dialog.id, dialog.title)
  }
  return titles
}
