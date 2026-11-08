// 共享模型类型(与后端 internal/domain 对齐)
export interface SystemGroup {
  sg_id: string
  source: string // scraped | joined | bio | import
  contributor_name?: string
  telegram_chat_id?: number
  resolution_status?: string
  entry_count?: number
  entries?: Array<{ link_id?: string; sg_id?: string; tme_link?: string; username?: string; source: string; status: string; link_type?: string }>
  title: string
  username?: string
  tme_link?: string
  gtype: string // group | supergroup | channel | unknown; legacy: public/private
  member_count: number
  free_speech_status: string // allowed | restricted | unknown
  status: string // valid | invalid | pending
  ref_count: number
  last_seen_at: string
}
