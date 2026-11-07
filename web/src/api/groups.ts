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
  return api.get('/group-library', { params: { ...params } as Record<string, string | number | boolean | undefined> })
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

export interface GroupLibraryImportJob {
  job_id: string
  raw_count: number
  valid_count: number
  duplicate_count: number
  invalid_count: number
  file_name?: string
  status: 'review' | 'pending' | 'processing' | 'cancelling' | 'cancelled' | 'completed' | 'failed'
  total_count: number
  processed_count: number
  imported_count: number
  failed_count: number
  skipped_count: number
  last_error?: string
  created_at: string
  finished_at?: string
}

// 文本或 txt/csv 文件先做链接预筛选，返回预览任务；确认后才进入后台解析。
export function importLibraryGroups(text: string, fileName = ''): Promise<GroupLibraryImportJob> {
  return api.post('/group-library/import', { text, file_name: fileName })
}

export function confirmImportLibraryGroups(jobId: string): Promise<GroupLibraryImportJob> {
  return api.post('/group-library/import-jobs/' + encodeURIComponent(jobId) + '/confirm')
}

export function listGroupLibraryImportJobs(): Promise<{ list: GroupLibraryImportJob[] }> {
  return api.get('/group-library/import-jobs')
}

export function getGroupLibraryImportJob(jobId: string): Promise<GroupLibraryImportJob> {
  return api.get('/group-library/import-jobs/' + encodeURIComponent(jobId))
}

export function cancelGroupLibraryImportJob(jobId: string): Promise<GroupLibraryImportJob> {
  return api.post('/group-library/import-jobs/' + encodeURIComponent(jobId) + '/cancel')
}

export function retryFailedGroupLibraryImportJob(jobId: string): Promise<GroupLibraryImportJob> {
  return api.post('/group-library/import-jobs/' + encodeURIComponent(jobId) + '/retry')
}

// 关键词搜群: 借本租户在线账号做 TG 全局搜索, 结果直接入库并返回
export function searchLibraryGroups(keyword: string, accountId?: string): Promise<{ list: SystemGroup[]; count: number }> {
  return api.post('/group-library/search', { keyword, account_id: accountId || undefined })
}
