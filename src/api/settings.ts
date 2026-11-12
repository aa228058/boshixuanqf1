import { api } from '@/utils/request'
import type { Tenant, Plan, ActivationCode, SupportConfig, Announcement, MessageTemplate, AccountGroup, ListResponse } from '@/types'

export interface TextDictEntry { entry_id: string; dict_id: string; word: string; weight: number; enabled: boolean }
export interface TextDict { dict_id: string; tenant_id: string; name: string; enabled: boolean; items: string[]; entries: TextDictEntry[]; created_at: string }

export const settingsApi = {
  getProfile: () => api.get<Tenant>('/tenant/profile'),
  getMessageDecoration: () => api.get<{ enabled: boolean }>('/tenant/message-decoration'),
  updateMessageDecoration: (enabled: boolean) => api.put<{ enabled: boolean }>('/tenant/message-decoration', { enabled }),
  activateCode: (code: string) => api.post<Tenant>('/tenant/activate', { code }),
  getPlans: () => api.get<Plan[]>('/plans'),
  getSupport: () => api.get<SupportConfig>('/support'),
  getAnnouncements: () => api.get<Announcement[]>('/announcements'),
  markAnnouncementRead: (id: string) => api.post<void>(`/announcements/${id}/read`),

  // Message Templates
  getTemplates: () => api.get<ListResponse<MessageTemplate>>('/templates'),
  createTemplate: (data: { name: string; content: string }) => api.post<MessageTemplate>('/templates', data),
  updateTemplate: (id: string, data: { name: string; content: string }) => api.put<void>(`/templates/${id}`, data),
  deleteTemplate: (id: string) => api.delete<void>(`/templates/${id}`),

  // Saved text collections (manual reuse)
  getTextDicts: () => api.get<ListResponse<TextDict>>('/text-dicts'),
  createTextDict: (data: { name: string; enabled: boolean; entries: Array<{ word: string; enabled: boolean }> }) => api.post<TextDict>('/text-dicts', data),
  updateTextDict: (id: string, data: { name: string; enabled: boolean; entries: Array<{ word: string; enabled: boolean }> }) => api.put<void>(`/text-dicts/${id}`, data),
  deleteTextDict: (id: string) => api.delete<void>(`/text-dicts/${id}`),

  // Account Groups
  getAccountGroups: () => api.get<ListResponse<AccountGroup>>('/account-groups'),
  createAccountGroup: (data: { name: string; color: string; account_ids: string[] }) => api.post<AccountGroup>('/account-groups', data),
  updateAccountGroup: (id: string, data: { name: string; color: string; account_ids: string[] }) => api.put<void>(`/account-groups/${id}`, data),
  deleteAccountGroup: (id: string) => api.delete<void>(`/account-groups/${id}`),

  // Export
  exportMembers: (taskId: string, format: 'csv' | 'json', groupId?: string) => {
    let url = `/scrape/${taskId}/export?format=${format}`
    if (groupId) url += `&group_id=${groupId}`
    return api.get(url)
  },

  // R119+R120: 通知中心(租户级 SMTP)
  getNotify: () => api.get<{ email: string; settings: Record<string, boolean>; events: Record<string, string>; smtp: Record<string, any>; smtp_configured: boolean }>('/tenant/notify'),
  updateNotify: (email: string, settings: Record<string, boolean>, smtp?: Record<string, any>) => api.put('/tenant/notify', { email, settings, smtp }),
  testNotify: () => api.post<{ message: string }>('/tenant/notify/test'),
}
