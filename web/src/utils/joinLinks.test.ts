import { describe, expect, it } from 'vitest'
import { findInvalidJoinLinks, normalizeJoinLinks } from './joinLinks'

describe('join link parsing', () => {
  it('normalizes usernames and public Telegram links', () => {
    expect(normalizeJoinLinks([
      '@sample_group',
      't.me/another_group',
      'https://t.me/third_group',
      'https://www.telegram.me/fourth_group',
    ].join('\n'))).toEqual([
      'https://t.me/sample_group',
      'https://t.me/another_group',
      'https://t.me/third_group',
      'https://t.me/fourth_group',
    ])
  })

  it('preserves private invite hashes from Telegram deep links', () => {
    expect(normalizeJoinLinks('tg://join?invite=AbCd_123-xy'))
      .toEqual(['https://t.me/+AbCd_123-xy'])
  })

  it('reports malformed lines without treating supported input as invalid', () => {
    const input = '@valid_group\nt.me/public_group\ntg://join?invite=valid_hash\nnot-a-link\nftp://t.me/group'
    expect(normalizeJoinLinks(input)).toEqual([
      'https://t.me/valid_group',
      'https://t.me/public_group',
      'https://t.me/+valid_hash',
    ])
    expect(findInvalidJoinLinks(input)).toEqual(['not-a-link', 'ftp://t.me/group'])
  })

  it('ignores blank lines and preserves input order', () => {
    expect(normalizeJoinLinks('\n@first_group\n\n@second_group\n'))
      .toEqual(['https://t.me/first_group', 'https://t.me/second_group'])
    expect(findInvalidJoinLinks('\n@first_group\n\n')).toEqual([])
  })
})
