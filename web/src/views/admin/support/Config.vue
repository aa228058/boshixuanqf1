<template>
  <main class="max-w-2xl mx-auto space-y-6">
    <Breadcrumb :items="[{ path: '/admin/dashboard', label: '仪表盘' }, { path: '', label: '客服配置' }]" />
    <div class="glass rounded-[var(--t-r-md)] p-5">
      <h3 class="text-lg font-semibold mb-6">客服信息配置</h3>
      <div class="space-y-4">
        <div>
          <label for="support-type" class="block text-sm text-text-secondary mb-1.5">联系方式类型</label>
          <div class="ui-segmented w-full" id="support-type" role="group" aria-label="联系方式类型">
            <button
              v-for="type in contactTypes"
              :key="type.value"
              type="button"
              @click="form.contact_type = type.value as SupportConfig['contact_type']"
              class="ui-segmented__item"
              :class="{ 'is-active': form.contact_type === type.value }"
              :aria-pressed="form.contact_type === type.value"
            >
              {{ type.label }}
            </button>
          </div>
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-1.5">联系账号/链接</label>
          <input
            v-model="form.contact_value"
            placeholder="如微信号、TG号、QQ号或邮箱"
            class="ui-input !text-sm" aria-label="如微信号、TG号、QQ号或邮箱" />
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-1.5">显示名称</label>
          <input
            v-model="form.display_name"
            placeholder="如：联系客服开通套餐"
            class="ui-input !text-sm" aria-label="如：联系客服开通套餐" />
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-1.5">二维码图片（可选）</label>
          <div class="flex items-center gap-3">
            <label class="px-3 py-1.5 rounded-lg border border-border text-xs cursor-pointer hover:border-[color:var(--t-brand)]/40 transition-colors">
              选择图片
              <input type="file" accept="image/png,image/jpeg" class="hidden" @change="onQrFileChange" />
            </label>
            <span class="text-xs text-text-muted">仅支持 JPG、PNG，且小于 500KB</span>
            <span v-if="qrReading" class="text-xs text-text-secondary" role="status">正在读取二维码…</span>
            <img v-if="form.qr_code_url" :src="form.qr_code_url" class="w-16 h-16 rounded-lg object-cover" alt="二维码" loading="lazy">
            <t-button v-if="form.qr_code_url" type="button" @click="clearQr" class="ui-t-button--danger ui-t-button" aria-label="清除二维码图片">
              清除图片
            </t-button>
          </div>
        </div>
      </div>
      <div class="flex justify-end gap-3 mt-6 pt-6 border-t border-border">
        <t-button type="button" @click="resetConfig" :disabled="savingSupport" class="ui-t-button--secondary ui-t-button">重置</t-button>
        <t-button theme="primary" size="sm" class="ui-t-button ui-t-button--primary ui-t-button--sm" @click="saveConfig" :loading="savingSupport" :disabled="savingSupport || qrReading">保存配置</t-button>
      </div>
    </div>

    <!-- 预览 -->
    <div class="glass rounded-[var(--t-r-md)] p-5">
      <h3 class="text-sm font-medium text-text-secondary mb-4">租户端预览</h3>
      <div class="bg-bg-card rounded-lg p-4">
        <div class="flex items-center gap-3">
          <MessageCircle class="w-8 h-8 text-[var(--t-brand)]" />
          <div>
            <p class="font-medium">{{ form.display_name || '联系客服' }}</p>
            <p class="text-sm text-text-secondary font-mono max-w-[65ch]">{{ form.contact_value || '未配置' }}</p>
          </div>
        </div>
        <div v-if="form.qr_code_url" class="mt-3 flex justify-center">
          <img :src="form.qr_code_url" class="w-32 h-32 rounded-lg" alt="二维码" loading="lazy">
        </div>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import { toast } from '@/utils/toast'
import Breadcrumb from '@/components/ui/Breadcrumb.vue'
import { ref, onMounted, onUnmounted } from 'vue'
import { MessageCircle } from 'lucide-vue-next'
import { adminApiMod } from '@/api/admin'
import type { SupportConfig } from '@/types'
import { isSupportedQrImage } from '@/utils/adminPageGuards'

const contactTypes = [
  { label: '微信', value: 'wechat' },
  { label: 'Telegram', value: 'telegram' },
  { label: 'QQ', value: 'qq' },
  { label: '邮箱', value: 'email' },
  { label: '自定义', value: 'custom' },
]

const qrReading = ref(false)
let qrReader: FileReader | null = null
const savingSupport = ref(false)
let supportPageActive = false
let supportRequestId = 0
let supportController: AbortController | null = null
let qrReadRequestId = 0
const form = ref<SupportConfig>({
  id: 0,
  contact_type: 'wechat',
  contact_value: '',
  display_name: '',
  qr_code_url: '',
  is_active: true,
})

async function fetchConfig() {
  supportController?.abort()
  const controller = new AbortController()
  supportController = controller
  const requestId = ++supportRequestId
  try {
    // P0-04: 原兜底默认值用了第三套互不相同的字段名(contact_wechat/telegram/url),
    // 与后端真实返回(contact_type/contact_value/display_name/qr_code_url)完全对不上——
    // 配置永远存不进正确的字段。以 007 迁移 + repository 的真实字段为准统一。
    const EMPTY = { id: 0, contact_type: 'wechat', contact_value: '', display_name: '', qr_code_url: '', is_active: true }
    const config = await adminApiMod.getSupport({ signal: controller.signal })
    if (!supportPageActive || requestId !== supportRequestId) return
    form.value = { ...EMPTY, ...(config || {}), qr_code_file: undefined }
  } catch (e: unknown) {
    if (!supportPageActive || requestId !== supportRequestId) return
    if (controller.signal.aborted) return
    // 首次部署无客服配置时接口可能报错/返回空, 兜底默认空值避免页面异常
    form.value = { id: 0, contact_type: 'wechat', contact_value: '', display_name: '', qr_code_url: '', is_active: true }
  } finally {
    if (supportController === controller) supportController = null
  }
}

async function saveConfig() {
  if (savingSupport.value || qrReading.value) return
  if (!form.value.contact_value.trim()) {
    toast({ type: 'warning', title: '请填写联系方式', message: '联系账号或链接不能为空' })
    return
  }
  savingSupport.value = true
  try {
    // File 只用于本地预览，不进入 JSON 请求；文本字段统一去除首尾空格。
    const { qr_code_file: _localFile, ...payload } = form.value
    await adminApiMod.updateSupport({
      ...payload,
      contact_value: payload.contact_value.trim(),
      display_name: payload.display_name.trim(),
    })
    if (!supportPageActive) return
    await fetchConfig()
    if (supportPageActive) toast({ type: 'success', title: '配置已保存' })
  } catch (e: unknown) {
    if (supportPageActive) toast({ type: 'error', title: '保存失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    if (supportPageActive) savingSupport.value = false
  }
}

// 清除二维码时使未完成的读取失效，防止异步 onload 把图片重新写回来。
function clearQr() {
  qrReadRequestId++
  if (qrReader?.readyState === FileReader.LOADING) qrReader.abort()
  qrReader = null
  qrReading.value = false
  form.value.qr_code_url = ''
  form.value.qr_code_file = undefined
}

function resetConfig() {
  clearQr()
  void fetchConfig()
}

function onQrFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  // 允许再次选择同一个文件。
  input.value = ''
  if (file) handleQrUpload(file)
}

function handleQrUpload(file: File) {
  if (!isSupportedQrImage(file.type, file.size)) {
    toast({ type: 'warning', title: '二维码图片无效', message: '仅支持非空 JPG、PNG 图片，且文件需小于 500KB' })
    return
  }
  if (qrReader?.readyState === FileReader.LOADING) qrReader.abort()
  const reader = new FileReader()
  qrReader = reader
  const requestId = ++qrReadRequestId
  qrReading.value = true
  reader.onload = () => {
    if (!supportPageActive || requestId !== qrReadRequestId) return
    if (typeof reader.result !== 'string') {
      qrReading.value = false
      qrReader = null
      toast({ type: 'error', title: '二维码读取失败', message: '请重新选择图片' })
      return
    }
    form.value.qr_code_url = reader.result
    form.value.qr_code_file = file
    qrReading.value = false
    qrReader = null
  }
  reader.onerror = () => {
    if (!supportPageActive || requestId !== qrReadRequestId) return
    qrReading.value = false
    qrReader = null
    toast({ type: 'error', title: '二维码读取失败', message: '请重新选择图片' })
  }
  reader.onabort = () => {
    if (requestId === qrReadRequestId) qrReading.value = false
  }
  reader.readAsDataURL(file)
}

onMounted(() => { supportPageActive = true; void fetchConfig() })
onUnmounted(() => {
  supportPageActive = false
  supportRequestId++
  qrReadRequestId++
  if (qrReader?.readyState === FileReader.LOADING) qrReader.abort()
  qrReader = null
  supportController?.abort()
  supportController = null
})
</script>
