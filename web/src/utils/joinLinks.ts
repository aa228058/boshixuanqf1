export function normalizeJoinLinks(input: string): string[] {
  return input.split('\n').map(line => line.trim()).filter(Boolean).map(line => {
    if (line.startsWith('@')) return 'https://t.me/' + line.slice(1)
    const invite = line.match(/^tg:\/\/join\?invite=([A-Za-z0-9_-]+)$/i)
    if (invite) return 'https://t.me/+' + invite[1]
    const link = line.match(/^(?:https?:\/\/)?(?:www\.)?(?:t\.me|telegram\.me)\/(.+)$/i)
    return link ? 'https://t.me/' + link[1] : ''
  }).filter(Boolean)
}

export function findInvalidJoinLinks(input: string): string[] {
  return input.split('\n').map(line => line.trim()).filter(Boolean).filter(line => {
    if (line.startsWith('@')) return false
    if (/^tg:\/\/join\?invite=[A-Za-z0-9_-]+$/i.test(line)) return false
    return !/^(?:https?:\/\/)?(?:www\.)?(?:t\.me|telegram\.me)\//i.test(line)
  })
}
