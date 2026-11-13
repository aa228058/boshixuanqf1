// downloadCSV 与页面级表格导出共用同一套公式注入防护和 CSV 引号转义策略。
export function downloadCSV(filename: string, headers: string[], rows: (string | number)[][]) {
  const csv = [headers.map(escapeSpreadsheetCSVCell).join(','), ...rows.map(r => r.map(escapeSpreadsheetCSVCell).join(','))].join('\n')
  const blob = new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8;' })
  downloadBlob(filename, blob)
}

/** 统一下载 Blob：将临时锚点挂载到 DOM，触发后清理锚点，并延迟释放 Object URL。 */
export function downloadBlob(filename: string, blob: Blob): void {
  const url = URL.createObjectURL(blob)
  let anchor: HTMLAnchorElement | null = null
  let failed = false
  let failure: unknown
  try {
    anchor = document.createElement('a')
    anchor.href = url
    anchor.download = filename
    document.body.appendChild(anchor)
    anchor.click()
  } catch (error) {
    failed = true
    failure = error
  }

  // 即使浏览器/测试环境的锚点清理抛错，也必须释放 URL，不能跳过后续回收。
  try {
    anchor?.remove()
  } catch (error) {
    if (!failed) {
      failed = true
      failure = error
    }
  }
  if (failed) {
    URL.revokeObjectURL(url)
    throw failure
  }
  try {
    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch (error) {
    // 定时器不可用时不能让刚创建的 Object URL 永久泄漏。
    URL.revokeObjectURL(url)
    throw error
  }
}

// R117: downloadJSON 已删(零外部引用; downloadCSV 自拼 CSV 不依赖它)


/** CSV 单元格：公式前缀强制按文本处理，并按 RFC 风格双引号包裹/转义。 */
export function escapeSpreadsheetCSVCell(value: unknown): string {
  let text = String(value ?? '')
  if (/^[\s\u0000-\u001f]*[=+\-@]/.test(text)) text = '\t' + text
  return '"' + text.replace(/"/g, '""') + '"'
}

/** Excel HTML 单元格：公式前缀保护后转义所有 HTML 特殊字符。 */
export function escapeSpreadsheetHTMLCell(value: unknown): string {
  let text = String(value ?? '')
  if (/^[\s\u0000-\u001f]*[=+\-@]/.test(text)) text = '\t' + text
  return text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;')
}

export function parseLinks(text: string): { valid: string[]; invalid: string[] } {
  const lines = text.split(/\n/).map(l => l.trim()).filter(Boolean)
  const valid: string[] = []
  const invalid: string[] = []
  const pattern = /^https?:\/\/(t\.me|telegram\.me)\/[-_a-zA-Z0-9+]+$/
  for (const line of lines) {
    if (pattern.test(line)) valid.push(line)
    else invalid.push(line)
  }
  return { valid, invalid }
}
