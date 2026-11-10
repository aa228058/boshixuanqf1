import { describe, expect, it } from 'vitest'
import { toggleSelectionByKey } from './selection'

type Item = { id: string; label: string }

describe('toggleSelectionByKey', () => {
  it('adds every filtered candidate without duplicating existing selections', () => {
    const selected: Item[] = [{ id: 'a', label: 'A' }, { id: 'outside', label: 'Outside' }]
    const candidates: Item[] = [
      { id: 'a', label: 'A (filtered copy)' },
      { id: 'b', label: 'B' },
      { id: 'c', label: 'C' },
    ]

    expect(toggleSelectionByKey(selected, candidates, item => item.id).map(item => item.id))
      .toEqual(['a', 'outside', 'b', 'c'])
  })

  it('removes only the currently filtered candidates when all are selected', () => {
    const selected: Item[] = [
      { id: 'a', label: 'A' },
      { id: 'b', label: 'B' },
      { id: 'outside', label: 'Outside' },
    ]

    expect(toggleSelectionByKey(selected, selected.slice(0, 2), item => item.id).map(item => item.id))
      .toEqual(['outside'])
  })

  it('leaves selection unchanged for an empty filtered result', () => {
    const selected: Item[] = [{ id: 'a', label: 'A' }]
    expect(toggleSelectionByKey(selected, [], item => item.id)).toEqual(selected)
  })
})
