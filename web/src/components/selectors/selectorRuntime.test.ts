import { describe, expect, it } from 'vitest'
import {
  areAllSelectorKeysSelected,
  createSelectorTitleLookup,
  filterSelectorItems,
  toggleAllSelectorKeys,
} from './selectorRuntime'

type Item = { id: string; title: string }

describe('selector runtime helpers', () => {
  it('filters case-insensitively without changing item order', () => {
    const items: Item[] = [
      { id: '1', title: 'Telegram Tools' },
      { id: '2', title: 'Product Discussion' },
      { id: '3', title: 'TELEGRAM Developers' },
    ]

    expect(filterSelectorItems(items, 'telegram', item => item.title).map(item => item.id))
      .toEqual(['1', '3'])
    expect(filterSelectorItems(items, '', item => item.title)).toEqual(items)
  })

  it('checks and toggles only the filtered keys while preserving outside selections', () => {
    const items: Item[] = [{ id: 'a', title: 'A' }, { id: 'b', title: 'B' }]
    const selected = new Set(['outside', 'a'])

    expect(areAllSelectorKeysSelected(items, selected, item => item.id)).toBe(false)
    const added = toggleAllSelectorKeys(items, selected, item => item.id)
    expect([...added]).toEqual(['outside', 'a', 'b'])
    expect(areAllSelectorKeysSelected(items, added, item => item.id)).toBe(true)

    const removed = toggleAllSelectorKeys(items, added, item => item.id)
    expect([...removed]).toEqual(['outside'])
  })

  it('returns false for empty results and handles 10,000 candidates without quadratic lookups', () => {
    const items = Array.from({ length: 10_000 }, (_, index) => ({
      id: `item-${index}`,
      title: `Group ${index}`,
    }))
    const selected = new Set(items.slice(0, 5_000).map(item => item.id))

    expect(areAllSelectorKeysSelected([], selected, item => item.id)).toBe(false)
    const filtered = filterSelectorItems(items, 'group 999', item => item.title)
    expect(filtered.map(item => item.id)).toEqual(['item-999'])

    const all = toggleAllSelectorKeys(items, selected, item => item.id)
    expect(all.size).toBe(10_000)
    expect(areAllSelectorKeysSelected(items, all, item => item.id)).toBe(true)

    const none = toggleAllSelectorKeys(items, all, item => item.id)
    expect(none.size).toBe(0)
  })

  it('indexes group chat IDs and usernames once, with group titles taking precedence', () => {
    const lookup = createSelectorTitleLookup(
      [{ tg_chat_id: '-100123', username: 'tg_group', title: 'Group title' }],
      [{ id: '42', title: 'Contact title' }, { id: '-100123', title: 'Conflicting contact' }],
    )

    expect(lookup.get('-100123')).toBe('Group title')
    expect(lookup.get('tg_group')).toBe('Group title')
    expect(lookup.get('@tg_group')).toBe('Group title')
    expect(lookup.get('42')).toBe('Contact title')
  })
})
