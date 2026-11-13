import { describe, expect, it, vi } from 'vitest'
import { copyWithExecCommand } from './useCopy'

function createDocument(execResult: boolean | Error) {
  const textarea = {
    value: '',
    parentNode: null as null | { removeChild: (node: unknown) => void },
    style: {} as Record<string, string>,
    setAttribute: vi.fn(),
    select: vi.fn(),
  }
  const removeChild = vi.fn((node: unknown) => {
    if (node === textarea) textarea.parentNode = null
  })
  const body = {
    appendChild: vi.fn((node: typeof textarea) => {
      node.parentNode = { removeChild }
      return node
    }),
  }
  const doc = {
    createElement: vi.fn(() => textarea),
    body,
    execCommand: vi.fn(() => {
      if (execResult instanceof Error) throw execResult
      return execResult
    }),
  } as unknown as Document
  return { doc, textarea, body, removeChild }
}

describe('copyWithExecCommand', () => {
  it('copies text and removes the temporary textarea', () => {
    const { doc, textarea, body, removeChild } = createDocument(true)
    expect(copyWithExecCommand('secret', doc)).toBe(true)
    expect(textarea.value).toBe('secret')
    expect(textarea.setAttribute).toHaveBeenCalledWith('readonly', '')
    expect(body.appendChild).toHaveBeenCalledWith(textarea)
    expect(removeChild).toHaveBeenCalledWith(textarea)
  })

  it('removes the textarea when the browser reports copy failure', () => {
    const { doc, textarea, removeChild } = createDocument(false)
    expect(copyWithExecCommand('secret', doc)).toBe(false)
    expect(removeChild).toHaveBeenCalledWith(textarea)
  })

  it('removes the textarea when execCommand throws', () => {
    const { doc, textarea, removeChild } = createDocument(new Error('blocked'))
    expect(copyWithExecCommand('secret', doc)).toBe(false)
    expect(removeChild).toHaveBeenCalledWith(textarea)
  })

  it('returns false when selecting the textarea throws and still cleans up', () => {
    const { doc, textarea, removeChild } = createDocument(true)
    textarea.select.mockImplementation(() => { throw new Error('select failed') })
    expect(copyWithExecCommand('secret', doc)).toBe(false)
    expect(removeChild).toHaveBeenCalledWith(textarea)
  })
})
