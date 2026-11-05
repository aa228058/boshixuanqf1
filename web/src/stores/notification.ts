import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
// 本地 UI 通知由前端事件产生，不等同于服务端持久化 Notification。
interface LocalNotification {
  id: string
  type: string
  title: string
  message: string
  read: boolean
  tenant_id?: string

  created_at: string
}

export const useNotificationStore = defineStore('notification', () => {
  const notifications = ref<LocalNotification[]>([])
  const unreadCount = computed(() => notifications.value.filter(n => !n.read).length)

  function addNotification(n: Omit<LocalNotification, 'id' | 'created_at'> & { notification_id?: string; is_read?: boolean }) {
    notifications.value.unshift({
      ...n,
      read: n.is_read ?? n.read,
      id: Math.random().toString(36).slice(2) + Date.now().toString(36),
      created_at: new Date().toISOString(),
    })
    if (notifications.value.length > 100) notifications.value.pop()
  }

  function markRead(id: string) {
    const n = notifications.value.find(x => x.id === id)
    if (n) n.read = true
  }

  function markAllRead() {
    notifications.value.forEach(n => n.read = true)
  }

  function remove(id: string) {
    notifications.value = notifications.value.filter(n => n.id !== id)
  }

  return { notifications, unreadCount, addNotification, markRead, markAllRead, remove }
})
