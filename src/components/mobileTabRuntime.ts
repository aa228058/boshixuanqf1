export type MobileTabKey = 'home' | 'broadcast' | 'accounts' | 'tasks' | 'more'

const staticPaths: Record<Exclude<MobileTabKey, 'broadcast' | 'tasks'>, string> = {
  home: '/dashboard',
  accounts: '/accounts',
  more: '/more',
}

export function mobileTabPath(key: MobileTabKey, accountId?: string | null): string {
  if (key === 'broadcast') return accountId ? `/workspace/${accountId}/broadcast` : '/accounts'
  if (key === 'tasks') return accountId ? `/workspace/${accountId}/tasks` : '/accounts'
  return staticPaths[key]
}

export function isMobileTabActive(key: MobileTabKey, pathname: string): boolean {
  if (key === 'home') return pathname === '/dashboard'
  if (key === 'accounts') return pathname === '/accounts' || pathname.startsWith('/accounts/')
  if (key === 'more') return pathname === '/more' || pathname.startsWith('/more/')
  if (key === 'broadcast') return /^\/workspace\/[^/]+\/broadcast(?:\/|$)/.test(pathname)
  return /^\/workspace\/[^/]+\/tasks(?:\/|$)/.test(pathname)
}
