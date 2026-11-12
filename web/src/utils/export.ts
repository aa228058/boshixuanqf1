// P2-15: 原实现不防公式注入——单元格以 = + - @ 开头时, Excel/WPS 打开会当公式
// 执行(=cmd|'/c...' 类 CSV 注入)。导出是备份/交接场景, 中招面真实存在。
// 处理: 危险前缀前加单引号(Excel 约定为纯文本), 再照常做引号转义。
function escapeCSV(value: string | number): string {
  let str = String(value)
  if (/^[=+\-@]/.test(str)) {
    str = `'${str}`
  }
  // 如果包含逗号、引号或换行符，需要用引号包裹
  if (/[",\n\r]/.test(str)) {
    return `"${str.replace(/"/g, '""')}"`
  }
  return str
}

export function downloadCSV(filename: string, headers: string[], rows: (string | number)[][]) {
  const csv = [headers.map(escapeCSV).join(','), ...rows.map(r => r.map(escapeCSV).join(','))].join('\n')
  const blob = new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  // Defer revocation until the browser has consumed the download URL.
  window.setTimeout(() => URL.revokeObjectURL(url), 1000)
}

/** 统一下载 Blob：将临时锚点挂载到 DOM，触发后清理锚点，并延迟释放 Object URL。 */
export function downloadBlob(filename: string, blob: Blob): void {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  try {
    document.body.appendChild(anchor)
    anchor.click()
  } catch (error) {
    URL.revokeObjectURL(url)
    throw error
  } finally {
    anchor.remove()
  }
  window.setTimeout(() => URL.revokeObjectURL(url), 1000)
}

// R117: downloadJSON 已删(零外部引用; downloadCSV 自拼 CSV 不依赖它)


/** CSV 单元格：公式前缀强制按文本处理，并按 RFC 风格双引号包裹/转义。 */
export function escapeSpreadsheetCSVCell(value: unknown): string {
  let text = String(value ?? '')
  if (/^[=+\-@]/.test(text)) text = '\t' + text
  return '"' + text.replace(/"/g, '""') + '"'
}

/** Excel HTML 单元格：公式前缀保护后转义所有 HTML 特殊字符。 */
export function escapeSpreadsheetHTMLCell(value: unknown): string {
  let text = String(value ?? '')
  if (/^[=+\-@]/.test(text)) text = '\t' + text
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
