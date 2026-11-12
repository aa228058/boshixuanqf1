import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { downloadBlob, downloadCSV, escapeSpreadsheetCSVCell, escapeSpreadsheetHTMLCell, parseLinks } from '../export'

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
  const createObjectURL = vi.fn((_blob: Blob) => 'blob:test-export')
  const revokeObjectURL = vi.fn()
  const click = vi.fn()
  const setTimeout = vi.fn((_handler: () => void, _delay: number) => 0)
  const remove = vi.fn()
  const appendChild = vi.fn()
  let anchor: { href: string; download: string; click: typeof click; remove: typeof remove }

  beforeEach(() => {
    createObjectURL.mockClear()
    revokeObjectURL.mockClear()
    click.mockClear()
    setTimeout.mockClear()
    remove.mockClear()
    appendChild.mockClear()
    anchor = { href: '', download: '', click, remove }
    vi.stubGlobal('URL', { createObjectURL, revokeObjectURL })
    vi.stubGlobal('document', {
      createElement: vi.fn((tag: string) => {
        expect(tag).toBe('a')
        return anchor
      }),
      body: { appendChild },
    })
    vi.stubGlobal('window', { setTimeout })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('生成带 BOM 的 UTF-8 CSV，并防止公式注入及正确转义字段', async () => {
    const captured: Blob[] = []
    createObjectURL.mockImplementation((value: Blob) => {
      captured.push(value)
      return 'blob:test-export'
    })

    downloadCSV('members.csv', ['name', 'value', 'memo'], [
      ['=1+1', 'a,b', 'say "hi"\nnext'],
      ['+SUM(A1:A2)', '-2', '@cmd'],
    ])

    expect(anchor.download).toBe('members.csv')
    expect(anchor.href).toBe('blob:test-export')
    expect(appendChild).toHaveBeenCalledWith(anchor)
    expect(click).toHaveBeenCalledOnce()
    expect(remove).toHaveBeenCalledOnce()
    expect(captured).toHaveLength(1)
    const bytes = new Uint8Array(await captured[0].arrayBuffer())
    // Blob.text() 按规范会剥离 UTF-8 BOM；直接检查字节，避免误报。
    expect(Array.from(bytes.slice(0, 3))).toEqual([0xef, 0xbb, 0xbf])
    expect(new TextDecoder().decode(bytes.slice(3))).toBe(
      'name,value,memo\n\'=1+1,"a,b","say ""hi""\nnext"\n\'+SUM(A1:A2),\'-2,\'@cmd',
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

describe('spreadsheet cell sanitizers', () => {
  it.each(['=', '+', '-', '@'])('CSV 将 %s 前缀作为文本并正确引用', (prefix) => {
    expect(escapeSpreadsheetCSVCell(prefix + 'SUM(A1:A2)')).toBe('"\t' + prefix + 'SUM(A1:A2)"')
  })

  it('CSV 保留 Unicode、空值、逗号、双引号和换行', () => {
    expect(escapeSpreadsheetCSVCell(null)).toBe('""')
    expect(escapeSpreadsheetCSVCell('中文, "名字"\r\n第二行')).toBe('"中文, ""名字""\r\n第二行"')
  })

  it('Excel HTML 单元格防公式注入并转义 HTML 特殊字符', () => {
    expect(escapeSpreadsheetHTMLCell('=1+1')).toBe('\t=1+1')
    expect(escapeSpreadsheetHTMLCell(`<img src=x onerror="x">&'\n`)).toBe('&lt;img src=x onerror=&quot;x&quot;&gt;&amp;&#39;\n')
    expect(escapeSpreadsheetHTMLCell(undefined)).toBe('')
  })
})


describe('downloadBlob', () => {
  it('挂载临时下载锚点，点击后移除锚点并延迟释放 Object URL', () => {
    const click = vi.fn()
    const remove = vi.fn()
    const appendChild = vi.fn()
    const revokeObjectURL = vi.fn()
    const setTimeout = vi.fn((handler: () => void, delay: number) => 0)
    const anchor = { href: '', download: '', click, remove }
    vi.stubGlobal('URL', { createObjectURL: vi.fn(() => 'blob:shared-download'), revokeObjectURL })
    vi.stubGlobal('document', {
      createElement: vi.fn(() => anchor),
      body: { appendChild },
    })
    vi.stubGlobal('window', { setTimeout })

    downloadBlob('report.json', new Blob(['{}'], { type: 'application/json' }))

    expect(anchor.href).toBe('blob:shared-download')
    expect(anchor.download).toBe('report.json')
    expect(appendChild).toHaveBeenCalledWith(anchor)
    expect(click).toHaveBeenCalledOnce()
    expect(remove).toHaveBeenCalledOnce()
    expect(setTimeout).toHaveBeenCalledOnce()
    expect(setTimeout.mock.calls[0][1]).toBe(1000)
    expect(revokeObjectURL).not.toHaveBeenCalled()
    ;(setTimeout.mock.calls[0][0] as () => void)()
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:shared-download')
    vi.unstubAllGlobals()
  })

  it('创建下载锚点失败时立即释放已创建的 Object URL', () => {
    const revokeObjectURL = vi.fn()
    vi.stubGlobal('URL', { createObjectURL: vi.fn(() => 'blob:anchor-failed'), revokeObjectURL })
    vi.stubGlobal('document', { createElement: vi.fn(() => { throw new Error('anchor creation failed') }) })
    vi.stubGlobal('window', { setTimeout: vi.fn() })

    expect(() => downloadBlob('report.json', new Blob(['{}']))).toThrow('anchor creation failed')
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:anchor-failed')
    vi.unstubAllGlobals()
  })

  it('点击失败时立即释放 Object URL 并仍然移除临时锚点', () => {
    const remove = vi.fn()
    const revokeObjectURL = vi.fn()
    const click = vi.fn(() => { throw new Error('download click failed') })
    const anchor = { href: '', download: '', click, remove }
    vi.stubGlobal('URL', { createObjectURL: vi.fn(() => 'blob:failed-download'), revokeObjectURL })
    vi.stubGlobal('document', {
      createElement: vi.fn(() => anchor),
      body: { appendChild: vi.fn() },
    })
    vi.stubGlobal('window', { setTimeout: vi.fn() })

    expect(() => downloadBlob('report.json', new Blob(['{}']))).toThrow('download click failed')
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:failed-download')
    expect(remove).toHaveBeenCalledOnce()
    vi.unstubAllGlobals()
  })
})
