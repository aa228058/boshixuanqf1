<template>
  <main class="max-w-2xl mx-auto space-y-6">
    <Breadcrumb :items="[{ path: '/admin/dashboard', label: '仪表盘' }, { path: '', label: '客服配置' }]" />
    <div class="glass rounded-[var(--t-r-md)] p-5">
      <h3 class="text-lg font-semibold mb-6">客服信息配置</h3>
      <div v-if="loadError" role="alert" class="mb-4 rounded-lg border border-rose/30 bg-rose/5 px-3 py-2 text-sm text-rose">
        配置读取失败，当前内容不能安全保存。请检查网络后重试读取。
      </div>
      <div v-else-if="loadingConfig" role="status" aria-live="polite" class="mb-4 text-sm text-text-secondary">
        正在读取客服配置…
      </div>
      <div class="space-y-4">
        <div>
          <span class="block text-sm text-text-secondary mb-1.5">联系方式类型</span>
          <div class="ui-segmented w-full" role="group" aria-label="联系方式类型">
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
          <label for="support-contact-value" class="block text-sm text-text-secondary mb-1.5">联系账号/链接</label>
          <input
            id="support-contact-value"
            v-model="form.contact_value"
            placeholder="如微信号、TG号、QQ号或邮箱"
            class="ui-input !text-sm" aria-label="如微信号、TG号、QQ号或邮箱" />
        </div>
        <div>
          <label for="support-display-name" class="block text-sm text-text-secondary mb-1.5">显示名称</label>
          <input
            id="support-display-name"
            v-model="form.display_name"
            placeholder="如：联系客服开通套餐"
            class="ui-input !text-sm" aria-label="如：联系客服开通套餐" />
        </div>
        <div>
          <span class="block text-sm text-text-secondary mb-1.5">二维码图片（可选）</span>
          <div class="flex items-center gap-3">
            <label class="px-3 py-1.5 rounded-lg border border-border text-xs cursor-pointer hover:border-[color:var(--t-brand)]/40 transition-colors">
              选择图片
              <input type="file" accept="image/*" class="hidden" @change="onQrFileChange" />
            </label>
            <span class="text-xs text-text-muted">支持 JPG、PNG 格式</span>
            <img v-if="form.qr_code_url" :src="form.qr_code_url" class="w-16 h-16 rounded-lg object-contain bg-white p-1" alt="二维码" loading="lazy">
            <t-button v-if="form.qr_code_url" type="button" @click="clearQr" class="ui-t-button--danger ui-t-button" aria-label="清除二维码图片">
              清除图片
            </t-button>
          </div>
        </div>
      </div>
      <div class="flex justify-end gap-3 mt-6 pt-6 border-t border-border">
        <t-button type="button" @click="fetchConfig" :disabled="loadingConfig || saving" class="ui-t-button--secondary ui-t-button">重新读取</t-button>
        <t-button theme="primary" size="sm" class="ui-t-button ui-t-button--primary ui-t-button--sm" @click="saveConfig" :disabled="!configLoaded || loadingConfig || saving" :loading="saving">保存配置</t-button>
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
import { ref, onMounted } from 'vue'
import { MessageCircle } from 'lucide-vue-next'
import { adminApiMod } from '@/api/admin'
import type { SupportConfig } from '@/types'

const contactTypes = [
  { label: '微信', value: 'wechat' },
  { label: 'Telegram', value: 'telegram' },
  { label: 'QQ', value: 'qq' },
  { label: '邮箱', value: 'email' },
  { label: '自定义', value: 'custom' },
]

const qrFile = ref<File | null>(null)
const loadingConfig = ref(false)
const configLoaded = ref(false)
const loadError = ref(false)
const saving = ref(false)
const form = ref<SupportConfig>({
  id: 0,
  contact_type: 'wechat',
  contact_value: '',
  display_name: '',
  qr_code_url: '',
  is_active: true,
})

async function fetchConfig() {
  loadingConfig.value = true
  configLoaded.value = false
  loadError.value = false
  try {
    // 空配置与读取失败必须区分：只有接口成功返回(包括首次部署返回空值)
    // 才允许保存，避免网络错误时把已有客服信息覆盖为空值。
    const EMPTY = { id: 0, contact_type: 'wechat', contact_value: '', display_name: '', qr_code_url: '', is_active: true }
    const config = await adminApiMod.getSupport()
    form.value = { ...EMPTY, ...(config || {}) }
    configLoaded.value = true
  } catch (e: unknown) {
    loadError.value = true
    toast({ type: 'error', title: '读取客服配置失败', message: e instanceof Error ? e.message : '请检查网络后重试' })
  } finally {
    loadingConfig.value = false
  }
}

async function saveConfig() {
  if (!configLoaded.value || loadingConfig.value || saving.value) return
  saving.value = true
  try {
    // qr_code_file 是 File 对象（仅本地预览用），JSON 序列化为空对象{}；
    // 剥离后提交，二维码内容走 qr_code_url（base64 data URL 或外链）。
    const { qr_code_file, ...payload } = form.value as any
    await adminApiMod.updateSupport(payload as any)
    toast({ type: 'success', title: '配置已保存' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '保存失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    saving.value = false
  }
}
// 清除已上传的二维码图片
function clearQr() {
  form.value.qr_code_url = ''
  form.value.qr_code_file = null as any
}

function onQrFileChange(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (f) handleQrUpload(f)
}

function handleQrUpload(file: File) {
  if (file.size > 512 * 1024) {
    toast({ type: 'warning', title: '图片过大', message: '二维码图片请小于 500KB' })
    return
  }
  // 半成品警示：这里只做 base64 预览并直接塞进 qr_code_url 提交——
  // File 本体从未上传到媒体服务，大图的 base64 会让保存报文臃肿；
  // 且 qr_code_file 字段（File 对象）JSON 序列化后为空对象，后端拿不到文件。
  // TODO: 走 mediaApi.upload 上传拿 URL，或后端支持 base64 落库并限制大小。
  const reader = new FileReader()
  reader.onload = () => { form.value.qr_code_url = reader.result as string }
  reader.readAsDataURL(file)
}

onMounted(fetchConfig)
</script>
