import { api } from '@/utils/request'
import type { SystemGroup } from '@/types/models'

export interface GroupLibraryResult {
  list: SystemGroup[]
  total: number
  page: number
  page_size: number
}

export interface GroupLibraryQuery {
  keyword?: string
  gtype?: string
  status?: string
  source?: string
  page?: number
  page_size?: number
}

// 平台级群组库(找群): 所有租户采集汇总, 登录即可见
export function getGroupLibrary(params: GroupLibraryQuery): Promise<GroupLibraryResult> {
  return api.get('/group-library', { params })
}

// 勾选批量加入: 对每个账号创建一个加群任务。
// opts.perAccountCount>0 时按账号轮流分批(每号只加自己那份);
// opts.intervalMin/MaxMinutes 为租户自定义加群间隔(分钟, 区间内随机)。
export interface JoinLibraryOptions {
  perAccountCount?: number
  intervalMinMinutes?: number
  intervalMaxMinutes?: number
}
export function joinLibraryGroups(accountIds: string[], sgIds: string[], opts: JoinLibraryOptions = {}): Promise<{ created: number; links: number; failed_err: string }> {
  return api.post('/group-library/join', {
    account_ids: accountIds,
    sg_ids: sgIds,
    per_account_count: opts.perAccountCount || 0,
    interval_min_minutes: opts.intervalMinMinutes || 0,
    interval_max_minutes: opts.intervalMaxMinutes || 0,
  })
}

// 链接批量导入: 文本里每行一个 t.me 链接/@用户名/邀请链接, 支持整段粘贴。
// 公开群/频道即时解析群名; 私密邀请后台自动补名(>200条直接进入后台队列)。
export function importLibraryGroups(text: string, accountId?: string): Promise<{ imported: number; failed: string[]; skipped: number }> {
  return api.post('/group-library/import', { text, account_id: accountId || undefined })
}

// 关键词搜群: 借本租户在线账号做 TG 全局搜索, 结果直接入库并返回
export function searchLibraryGroups(keyword: string, accountId?: string): Promise<{ list: SystemGroup[]; count: number }> {
  return api.post('/group-library/search', { keyword, account_id: accountId || undefined })
}
