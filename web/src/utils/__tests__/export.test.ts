import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { downloadCSV, parseLinks } from '../export'

describe('parseLinks', () => {
  it('提取并去除首尾空白的 Telegram 群链接', () => {
    const result = parseLinks('  https://t.me/abc  \nhttps://telegram.me/xyz123\n')
    expect(result).toEqual({
      valid: ['https://t.me/abc', 'https://telegram.me/xyz123'],
      invalid: [],
    })
  })

  it('非 Telegram 链接与非法行进入 invalid', () => {
    const result = parseLinks('https://example.com/a\n随便一行字\nftp://t.me/group')
    expect(result.valid).toEqual([])
    expect(result.invalid).toEqual([
      'https://example.com/a',
      '随便一行字',
      'ftp://t.me/group',
    ])
  })

  it('忽略空行并保留重复链接的输入顺序', () => {
    const result = parseLinks('\nhttps://t.me/a\n\nhttps://t.me/a\n')
    expect(result.valid).toEqual(['https://t.me/a', 'https://t.me/a'])
  })
})

describe('downloadCSV', () => {
  const createObjectURL = vi.fn(() => 'blob:test-export')
  const revokeObjectURL = vi.fn()
  const click = vi.fn()
  const setTimeout = vi.fn()
  let anchor: { href: string; download: string; click: typeof click }
  let blob: Blob

  beforeEach(() => {
    createObjectURL.mockClear()
    revokeObjectURL.mockClear()
    click.mockClear()
    setTimeout.mockClear()
    anchor = { href: '', download: '', click }
    vi.stubGlobal('URL', { createObjectURL, revokeObjectURL })
    vi.stubGlobal('document', {
      createElement: vi.fn((tag: string) => {
        expect(tag).toBe('a')
        return anchor
      }),
    })
    vi.stubGlobal('window', { setTimeout })
    // Capture the Blob passed to createObjectURL without depending on a browser DOM.
    createObjectURL.mockImplementation((value?: never) => {
      void value
      return 'blob:test-export'
    })
    const originalCreateObjectURL = createObjectURL
    originalCreateObjectURL.mockImplementation(() => 'blob:test-export')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('生成带 BOM 的 UTF-8 CSV，并防止公式注入及正确转义字段', async () => {
    const originalCreate = URL.createObjectURL
    const captured: Blob[] = []
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn((value: Blob) => {
        captured.push(value)
        return originalCreateObjectURL()
      }),
      revokeObjectURL,
    })

    downloadCSV('members.csv', ['name', 'value', 'memo'], [
      ['=1+1', 'a,b', 'say "hi"\nnext'],
      ['+SUM(A1:A2)', '-2', '@cmd'],
    ])

    expect(anchor.download).toBe('members.csv')
    expect(anchor.href).toBe('blob:test-export')
    expect(click).toHaveBeenCalledOnce()
    expect(captured).toHaveLength(1)
    expect(await captured[0].text()).toBe(
      '\uFEFFname,value,memo\n\'=1+1,"a,b","say ""hi""\nnext"\n\'+SUM(A1:A2),\'-2,\'@cmd',
    )
  })

  it('下载触发后延迟 1000ms 释放 Object URL，而非立即释放', () => {
    downloadCSV('links.csv', ['url'], [['https://t.me/group']])

    expect(setTimeout).toHaveBeenCalledOnce()
    const [release, delay] = setTimeout.mock.calls[0] as [() => void, number]
    expect(delay).toBe(1000)
    expect(revokeObjectURL).not.toHaveBeenCalled()
    release()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:test-export')
  })
})
