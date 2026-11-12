<template>
  <div class="w-full min-w-0 p-3 sm:p-5 md:p-6 max-w-3xl mx-auto">
    <div class="flex items-center justify-between gap-3 mb-4">
      <div class="min-w-0">
        <h1 class="text-title font-extrabold tracking-tight text-text-primary truncate">群发消息</h1>
        <!-- R109: 移除"当前账号"标签(群发已多账号池化, 单一账号标签误导) -->
        <p class="text-sm text-text-muted mt-1">执行账号在创建任务时选择, 可多选</p>
      </div>
      <t-button theme="primary" class="ui-t-button ui-t-button--primary shrink-0" @click="showCreate = true">新建群发任务</t-button>
    </div>

    <!-- 页面 tab: 任务列表 / 发送日历(发送日历合并自原独立页) -->
    <div class="flex items-center gap-1 border-b border-border mb-4">
      <button type="button" v-for="t in pageTabs" :key="t.key"
              @click="activeTab = t.key"
              class="px-4 py-2 text-sm font-medium transition-colors relative"
              :class="activeTab === t.key ? 'text-[var(--t-brand)]' : 'text-text-secondary hover:text-text-primary'">
        {{ t.label }}
        <span v-if="activeTab === t.key" class="tab-underline absolute bottom-0 left-0 right-0 h-0.5 bg-[var(--t-brand)] rounded-full" />
      </button>
    </div>

    <!-- 发送日历 tab: 每日成功发送量月历热力图 -->
    <SendCalendar v-if="activeTab === 'calendar'" />

    <!-- 任务列表 tab(默认): 原有全部内容 -->
    <div v-show="activeTab === 'tasks'">
    <!-- 实时进度面板: 创建任务后在此显示, 无需跳转 -->
    <!-- R110: 协议号协同任务记录块已删(isProtocol 恒 false 死代码; 统一走多号池任务) -->
    <!-- 进行中的任务在下方列表已显示成功/失败/等待进度, 不再重复展示实时卡片 -->

    <!-- 本账号的群发任务列表 -->
    <div v-if="tasksLoading" class="glass rounded-[var(--t-r-md)] p-4 space-y-3 mb-4">
      <div v-for="i in 3" :key="i" class="space-y-2.5">
        <div class="flex items-center justify-between">
          <Skeleton height="1.25rem" width="40%" rounded="rounded-md" />
          <Skeleton height="0.875rem" width="25%" rounded="rounded-md" />
        </div>
        <Skeleton height="0.5rem" rounded="rounded-full" />
      </div>
    </div>
    <template v-else>
      <EmptyState v-if="tasksError" icon="MessageSquare" :title="tasksError" description="点击重试">
        <t-button theme="default" variant="outline" size="sm" class="ui-t-button ui-t-button--secondary ui-t-button--sm mt-3" @click="fetchTasks">重新加载</t-button>
      </EmptyState>
      <EmptyState v-else-if="!tasks.length" icon="MessageSquare" title="暂无群发任务" description="点击右上角「新建群发任务」开始发送" />
      <div v-else class="space-y-3 mb-4">
        <pull-down-refresh v-if="tasks.length" v-model="pullRefreshing" @refresh="onPullRefresh">
        <div v-for="(t, ti) in tasks" :key="t.task_id" :class="['glass rounded-[var(--t-r-md)] p-3 sm:p-4 card-lift card-enter hover:border-[color:var(--t-brand-dim)] mb-3', t.status === 'running' ? 'card-running' : '']" :style="{ '--i': ti }">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <span class="px-2 py-0.5 rounded text-xs shrink-0 inline-flex items-center gap-1.5" :class="statusClass(t.status)">
                <span v-if="t.status === 'running'" class="w-1.5 h-1.5 rounded-full bg-current dot-pulse" />
                {{ statusLabel(t.status) }}
              </span>
              <span v-if="t.is_loop" class="text-[10px] px-1.5 py-0.5 rounded bg-[var(--t-brand-dim)] text-[var(--t-brand)] shrink-0">循环 · {{ loopScheduleLabel(t.payload) }}</span>
              <span v-else class="text-[10px] px-1.5 py-0.5 rounded bg-bg-hover text-text-muted shrink-0">单次</span>
              <span class="text-xs text-text-muted truncate">目标 {{ (t.target_ids || []).length || progOf(t).total || '-' }} 个</span>
            </div>
            <span class="text-xs text-text-muted shrink-0">{{ formatTime(t.created_at) }}</span>
          </div>
          <div v-if="t.is_loop && t.next_run_at && ['pending','paused','completed'].includes(t.status)" class="mt-2 text-[11px] text-text-muted">
            下次计划执行：{{ formatTime(t.next_run_at) }}
          </div>
          <div v-if="t.is_loop" class="mt-1 text-[11px] text-text-muted">
            已完成轮数：{{ Number((t.payload as any)?.round_count || 0) }}
          </div>
          <div v-if="t.is_loop && ((t.payload as any)?.max_rounds || (t.payload as any)?.end_at)" class="mt-1 text-[11px] text-text-muted">
            <span v-if="(t.payload as any)?.max_rounds">最多 {{ (t.payload as any).max_rounds }} 轮</span>
            <span v-if="(t.payload as any)?.max_rounds && (t.payload as any)?.end_at"> · </span>
            <span v-if="(t.payload as any)?.end_at">结束于 {{ formatTime((t.payload as any).end_at) }}</span>
          </div>
          <div v-if="t.is_loop && loopRoundHistory(t).length" class="mt-2 rounded-lg border border-border/60 bg-bg-hover/40 px-3 py-2">
            <div class="text-[10px] text-text-muted mb-1.5">最近执行轮次</div>
            <div v-for="round in loopRoundHistory(t)" :key="round.round" class="flex flex-wrap items-center gap-x-2 gap-y-1 py-1 text-[11px]">
              <span class="font-medium text-text-primary">第 {{ round.round }} 轮</span>
              <span class="text-[var(--t-ok)]">成功 {{ round.success || 0 }}</span>
              <span class="text-[var(--t-err)]">失败 {{ round.failed || 0 }}</span>
              <span class="text-text-muted ml-auto">{{ formatTime(round.completed_at) }}</span>
            </div>
          </div>
          <div class="mt-3">
            <div class="flex items-center justify-between text-xs mb-1.5">
              <span class="text-text-secondary tabular-nums">
                {{ progOf(t).done + (progOf(t).failed || 0) }} / {{ progOf(t).total }} 已处理
                <span class="text-[var(--t-ok)] ml-1.5">{{ progOf(t).done }} 成功</span>
                <span v-if="progOf(t).failed" class="text-[var(--t-err)] ml-1.5">{{ progOf(t).failed }} 失败</span>
              </span>
              </div>
            <!-- 进度环(交互示例28): 绿=成功段 青=其余进度, 44px 紧凑 -->
            <div class="flex items-center gap-2 mt-2">
              <ProgressRing :percent="percentOf(t)" :success="progOf(t).total ? Math.round((progOf(t).done / progOf(t).total) * 100) : 0" :size="44" :stroke="4" />
              <div class="flex-1">
                <!-- 分段进度条: 绿=成功 红=失败 灰底=剩余, 进行中/已完成都有 -->
                <div class="h-2 rounded-full bg-bg-hover overflow-hidden flex">
              <div class="h-full bg-[var(--t-ok)] transition-all duration-500"
                   :style="{ width: (progOf(t).total ? (progOf(t).done / progOf(t).total) * 100 : 0) + '%' }" />
              <div v-if="progOf(t).failed" class="h-full bg-[var(--t-err)] transition-all duration-500"
                   :style="{ width: (progOf(t).total ? (progOf(t).failed / progOf(t).total) * 100 : 0) + '%' }" />
                </div>
              </div>
            </div>
          </div>
          <div class="flex items-center justify-end gap-2 mt-3">
            <t-button v-if="['failed','cancelled','completed'].includes(t.status) && progOf(t).failed > 0" theme="default" variant="outline" size="sm" class="ui-t-button ui-t-button--secondary ui-t-button--sm" @click.stop="retryFailed(t.task_id)">重试失败 ({{ progOf(t).failed }})</t-button>
            <t-button v-if="['running','queued','pending'].includes(t.status)" theme="default" variant="outline" size="sm" class="ui-t-button ui-t-button--secondary ui-t-button--sm" @click.stop="pauseTask(t.task_id)">暂停</t-button>
            <t-button v-if="t.status === 'paused'" theme="primary" size="sm" class="ui-t-button ui-t-button--primary ui-t-button--sm" @click.stop="resumeTask(t.task_id)">继续</t-button>
            <t-button v-if="['running','queued','pending','paused'].includes(t.status)" theme="danger" variant="outline" size="sm" class="ui-t-button ui-t-button--danger ui-t-button--sm" @click.stop="stopTask(t.task_id)">停止</t-button>
            <t-button theme="default" variant="outline" size="sm" class="ui-t-button ui-t-button--secondary ui-t-button--sm" @click.stop="$router.push(`/workspace/${route.params.accountId}/tasks/${t.task_id}`)">查看详情</t-button>
          </div>
        </div>
        </pull-down-refresh>
      </div>
    </template>
    </div><!-- /v-show tasks tab 容器 -->

    <!-- 新建模态框: 表单全部内容, 统一全站弹窗组件(留在 tab 容器外, 任何 tab 可开) -->
    <Modal v-model="showCreate" title="新建群发任务">
    <div class="space-y-3 sm:space-y-4">
      <!-- R110: 协议号协同死分支已删(isProtocol 恒 false)。统一多号池流程见下 -->
      <!-- 目标选择 -->
      <div>
        <!-- R98 表单分组① -->
      <div class="flex items-center gap-2 mb-3 -mt-1"><span class="w-[18px] h-[18px] rounded-full grid place-items-center text-[10px] font-black text-white shrink-0" style="background: var(--t-brand);">1</span><span class="text-[11.5px] font-extrabold uppercase tracking-wider" style="color: var(--t-text-2);">发送目标</span><span class="flex-1 h-px" style="background: var(--t-line);"></span></div>
      <span class="block text-sm font-medium mb-2">发送目标</span>
        <div role="group" aria-label="发送目标" class="p-3 rounded-lg border border-border hover:border-[color:var(--t-brand)]/30 transition-colors">
          <div class="flex items-center gap-2">
            <button type="button" aria-label="选择发送目标" @click="showTargetSelector = true" class="flex min-w-0 flex-1 items-center gap-2 text-left">
              <MessageSquare class="w-5 h-5 text-text-muted shrink-0" />
              <span v-if="!selectedTargets.length" class="text-text-muted">点击选择发送目标</span>
              <span v-else class="text-sm">已选 {{ selectedTargets.length }} 个目标</span>
            </button>
            <t-button v-if="selectedTargets.length" type="button" theme="primary" variant="text" size="sm" class="ui-t-button ui-t-button--text ui-t-button--sm shrink-0" @click="showTargetNames = !showTargetNames">{{ showTargetNames ? '收起' : '查看群名' }}</t-button>
          </div>
          <!-- 群名 chips: 全部列出, 小字, 可滚动 -->
          <div v-if="selectedTargets.length && showTargetNames" class="flex flex-wrap gap-1 mt-2 max-h-40 overflow-y-auto">
            <span v-for="tid in selectedTargets" :key="tid" class="px-1.5 py-0.5 rounded text-[11px] bg-bg-secondary text-text-secondary border border-border/60">{{ targetTitle(tid) }}</span>
          </div>
        </div>
        <TargetSelector v-model="showTargetSelector" :groups="groups" :accountId="currentAccount?.account_id" @select="selectedTargets = $event" @refresh="onSelectorRefresh" />
        <!-- 执行账号多选；实际发送范围按每个账号已加入的群组映射计算 -->
        <span class="block text-sm font-medium mb-2 mt-4">执行账号（可多选）</span>
        <div class="flex flex-wrap gap-1.5" role="group" aria-label="选择执行账号">
          <button v-for="acc in onlineAccounts" :key="acc.account_id" type="button" @click="toggleExec(acc.account_id)" :aria-pressed="execAccounts.includes(acc.account_id)"
                  class="px-2.5 py-1.5 rounded-lg text-xs border transition-colors whitespace-nowrap"
                  :class="execAccounts.includes(acc.account_id) ? 'border-[color:var(--t-brand)]/50 bg-[color:var(--t-brand-dim)] text-[var(--t-brand)] font-medium' : 'border-border text-text-secondary hover:border-border-hover'">
            {{ acc.first_name || acc.phone }}
          </button>
        </div>
        <div class="mt-3">
          <span class="block text-xs font-semibold text-text-secondary mb-2">账号分配策略</span>
          <div class="grid grid-cols-2 gap-2">
            <button type="button" @click="sendStrategy = 'assign'" :aria-pressed="sendStrategy === 'assign'"
              class="rounded-lg border px-3 py-2 text-left text-xs transition-colors"
              :class="sendStrategy === 'assign' ? 'border-[color:var(--t-brand)] bg-[color:var(--t-brand-dim)] text-[var(--t-brand)]' : 'border-border text-text-secondary'">
              <span class="block font-semibold">均衡分配</span>
              <span class="block mt-1 opacity-75">每个群由一个已加入的账号负责</span>
            </button>
            <button type="button" @click="sendStrategy = 'all'" :aria-pressed="sendStrategy === 'all'"
              class="rounded-lg border px-3 py-2 text-left text-xs transition-colors"
              :class="sendStrategy === 'all' ? 'border-[color:var(--t-brand)] bg-[color:var(--t-brand-dim)] text-[var(--t-brand)]' : 'border-border text-text-secondary'">
              <span class="block font-semibold">全部符合条件账号</span>
              <span class="block mt-1 opacity-75">每个已加入该群的所选账号各执行一次</span>
            </button>
          </div>
          <p class="text-[10.5px] text-text-muted mt-1.5">{{ isLoop ? '循环任务每一轮沿用此账号分配关系。' : '单次任务按此分配关系执行一轮。' }}</p>
        </div>
        <p v-if="selectedTargets.length && execAccounts.length" class="text-[10.5px] text-text-muted mt-1.5">
          按账号入群情况预计发送 <strong class="text-text-primary">{{ coverage.ok }}</strong> 条
          <span v-if="coverage.unreachable.length"> · {{ coverage.unreachable.length }} 个群没有所选账号加入</span>
          · 约需 {{ estMinutes }} 分钟
        </p>
        <!-- R101 P1-第二刀: 覆盖预览 -->
        <div v-if="selectedTargets.length && execAccounts.length && (coverage.ok > 0 || coverage.unreachable.length > 0)"
             class="mt-3 rounded-lg border p-3 text-xs"
             :class="coverage.unreachable.length ? 'border-[color:var(--t-warn)]/30 bg-[var(--t-warn)]/5' : 'border-[color:var(--t-ok)]/30 bg-[var(--t-ok)]/5'">
          <div>
            <span class="text-[var(--t-ok)]">✅ 可发 {{ coverage.ok }} 条</span>
            <span v-if="coverage.unreachable.length" class="text-[var(--t-err)] ml-2">⛔ {{ coverage.unreachable.length }} 个群无账号加入</span>
          </div>
          <div v-if="coverage.unreachable.length" class="text-[10.5px] text-[var(--t-warn)]/90 mt-1.5 leading-relaxed">
            未覆盖：{{ coverage.unreachable.slice(0, 3).join('、') }}{{ coverage.unreachable.length > 3 ? ` 等 ${coverage.unreachable.length} 个` : '' }}——这些群发送会失败，建议先把账号拉进群
          </div>
        </div>
      </div>

      <div class="mt-2">
        <t-button type="button" theme="primary" variant="text" size="sm" class="ui-t-button ui-t-button--text ui-t-button--sm" @click="showImport = !showImport">批量导入目标（粘贴列表/上传文件）</t-button>
        <div v-if="showImport" class="mt-2 space-y-2">
          <textarea v-model="importText" rows="4" class="ui-input !text-sm" aria-label="批量导入群组目标"
            placeholder="每行一个目标, 支持:&#10;@用户名&#10;https://t.me/群名&#10;-100xxxxxxxxxx(群ID)&#10;也可粘贴CSV内容"></textarea>
          <div class="flex gap-2">
            <t-button theme="primary" size="sm" variant="outline" class="ui-t-button ui-t-button--secondary ui-t-button--sm" @click="applyImportText">解析添加</t-button>
            <label class="px-3 py-1.5 rounded-lg border border-border text-xs cursor-pointer hover:border-[color:var(--t-brand)] transition-colors">
              选择文件<input type="file" accept=".txt,.csv" class="hidden" @change="onImportFile">
            </label>
          </div>
        </div>
      </div>

      <!-- R98 表单分组② -->
      <div class="flex items-center gap-2 mb-3 mt-5"><span class="w-[18px] h-[18px] rounded-full grid place-items-center text-[10px] font-black text-white shrink-0" style="background: var(--t-brand);">2</span><span class="text-[11.5px] font-extrabold uppercase tracking-wider" style="color: var(--t-text-2);">消息内容</span><span class="flex-1 h-px" style="background: var(--t-line);"></span></div>
      <!-- 消息内容 -->
      <div>
      <label for="broadcast-message" class="block text-sm font-medium mb-2">消息内容</label>
        <span v-if="draftSaved" class="text-xs text-[var(--t-ok)] mb-2 block">草稿已自动保存</span>
        <t-textarea id="broadcast-message" v-model="message" :autosize="{ minRows: 4, maxRows: 10 }"
                    placeholder="输入消息内容；如需随机表达，请使用 {选项一|选项二} 格式。" />
        <div class="flex items-center justify-between mt-2">
          <span class="text-xs text-text-muted">{{ message.length }} 字符</span>
        </div>
        <div v-if="authStore.tenant?.plan_features?.dict && textDicts.length" class="mt-3 rounded-lg border border-border/70 p-3 space-y-2">
          <div class="flex items-center justify-between gap-2"><span class="text-xs font-medium text-text-secondary">我的文本素材</span><button type="button" class="text-xs text-[var(--t-brand)]" @click="showTextSnippets = !showTextSnippets">{{ showTextSnippets ? "收起" : "选择并插入" }}</button></div>
          <div v-if="showTextSnippets" class="space-y-2">
            <select v-model="selectedTextDictId" class="ui-input w-full text-sm"><option value="">选择素材分组</option><option v-for="d in textDicts.filter(item => item.enabled !== false)" :key="d.dict_id" :value="d.dict_id">{{ d.name }}</option></select>
            <div v-if="selectedTextDict" class="flex flex-wrap gap-2">
              <button v-for="(item, index) in selectedTextDict.items" :key="index" type="button" class="px-2.5 py-1.5 rounded-lg border border-border text-xs text-text-secondary hover:border-[color:var(--t-brand)] text-left max-w-full break-words" @click="insertTextSnippet(item)">{{ item }}</button>
            </div>
            <p v-else class="text-xs text-text-muted">选择一个分组后，点击素材即可插入到消息末尾。</p>
          </div>
        </div>
      </div>

      <!-- 富媒体(可选) -->
      <div>
        <label for="broadcast-media-type" class="block text-sm font-medium mb-2">附带媒体 <span class="text-text-muted font-normal">(可选)</span></label>
        <t-select id="broadcast-media-type" v-model="mediaType" class="mb-2" :options="[
          { label: '无', value: '' },
          { label: '图片', value: 'photo' },
          { label: '视频', value: 'video' },
        ]" />
        <div v-if="mediaType" class="space-y-2">
          <input v-model="mediaUrl" aria-label="媒体 URL" placeholder="媒体 URL (https://...)，或上传本地文件"
                 class="ui-input !text-sm" />
          <div class="flex items-center gap-2">
            <label class="px-3 py-1.5 rounded-lg border border-border text-xs cursor-pointer hover:border-[color:var(--t-brand)] transition-colors">
              上传文件<input type="file" accept=".jpg,.jpeg,.png,.webp,.gif,.mp4,.mov" class="hidden" @change="onMediaFile">
            </label>
            <span v-if="mediaPath" class="text-xs text-[var(--t-brand)]">已上传: {{ mediaPath }}</span>
            <span v-if="uploadingMedia" class="text-xs text-text-muted">上传中…</span>
            <t-button v-if="mediaPath || mediaUrl" theme="primary" variant="text" size="sm" class="ui-t-button ui-t-button--text ui-t-button--sm" @click="clearMedia">清除</t-button>
          </div>
        </div>
      </div>

      <!-- R98 表单分组③ -->
      <div class="flex items-center gap-2 mb-3 mt-5"><span class="w-[18px] h-[18px] rounded-full grid place-items-center text-[10px] font-black text-white shrink-0" style="background: var(--t-brand);">3</span><span class="text-[11.5px] font-extrabold uppercase tracking-wider" style="color: var(--t-text-2);">发送设置</span><span class="flex-1 h-px" style="background: var(--t-line);"></span></div>
      <!-- 发送间隔设置 -->
      <div>
        <span class="block text-sm font-medium mb-2">发送间隔（秒）</span>
        <!-- TDesign 骨架: 数字输入组件(品牌紫双主题) -->
        <div class="grid grid-cols-2 gap-2 sm:gap-3">
          <label class="min-w-0 block">
            <span class="block text-[11px] text-text-muted mb-1">最小间隔</span>
            <t-input-number v-model="intervalMin" :min="1" :max="300" theme="normal" class="w-full" />
          </label>
          <label class="min-w-0 block">
            <span class="block text-[11px] text-text-muted mb-1">最大间隔</span>
            <t-input-number v-model="intervalMax" :min="1" :max="300" theme="normal" class="w-full" />
          </label>
        </div>
        <p class="text-xs text-text-muted mt-1">系统会在最小和最大间隔之间随机取值，模拟真人发送</p>
      </div>

      <!-- 单次任务的定时启动；循环任务使用下方首次执行时间 -->
      <div v-if="!isLoop">
        <label for="broadcast-scheduled-at" class="block text-sm font-medium mb-2">定时启动 <span class="text-text-muted font-normal">(可选, 留空立即执行)</span></label>
        <t-date-picker id="broadcast-scheduled-at" v-model="scheduledAtTs" type="datetime" value-type="time-stamp" clearable
                       placeholder="选择定时启动时间（留空立即执行）" class="w-full" />
        <p class="text-xs text-text-muted mt-1">到设定时间后任务自动进入队列执行</p>
      </div>

      <!-- 发送模式(2026-10-20 双 tab): 只发一次 / 循环群发(套餐 feature 控制) -->
      <div>
        <span class="block text-sm font-medium mb-2">发送模式</span>
        <div class="flex rounded-lg bg-ink-01 p-1 gap-1 w-fit" role="group" aria-label="发送模式">
          <button type="button" class="ui-segmented__item"
                  :class="!isLoop ? 'bg-bg-card text-text-primary font-medium shadow-sm' : 'text-text-secondary'"
                  @click="isLoop = false" :aria-pressed="!isLoop">只发一次</button>
          <button type="button" class="ui-segmented__item relative"
                  :class="[isLoop ? 'bg-bg-card text-text-primary font-medium shadow-sm' : 'text-text-secondary', !loopFeature && 'opacity-60']"
                  @click="loopFeature ? (isLoop = true) : toast({ type: 'warning', title: '套餐未开通循环群发', message: '请联系管理员升级' })" :aria-pressed="isLoop">
            循环群发
            <span v-if="!loopFeature" class="absolute -top-1 -right-1 text-[9px] px-1 rounded bg-[color:var(--t-warn-dim)] text-[var(--t-warn)] border border-[color:var(--t-warn)]/40">锁</span>
          </button>
        </div>
        <div v-if="isLoop" class="mt-3 rounded-lg border border-border/70 p-3 space-y-2">
          <span class="block text-xs font-semibold text-text-secondary">循环方式</span>
          <div class="grid grid-cols-3 gap-1.5" role="group" aria-label="循环方式">
            <button type="button" @click="loopScheduleMode = 'interval'" :aria-pressed="loopScheduleMode === 'interval'" class="rounded-lg border px-2 py-2 text-xs"
              :class="loopScheduleMode === 'interval' ? 'border-[color:var(--t-brand)] bg-[color:var(--t-brand-dim)] text-[var(--t-brand)]' : 'border-border text-text-secondary'">间隔循环</button>
            <button type="button" @click="loopScheduleMode = 'daily'" :aria-pressed="loopScheduleMode === 'daily'" class="rounded-lg border px-2 py-2 text-xs"
              :class="loopScheduleMode === 'daily' ? 'border-[color:var(--t-brand)] bg-[color:var(--t-brand-dim)] text-[var(--t-brand)]' : 'border-border text-text-secondary'">每天固定时间</button>
            <button type="button" @click="loopScheduleMode = 'weekly'" :aria-pressed="loopScheduleMode === 'weekly'" class="rounded-lg border px-2 py-2 text-xs"
              :class="loopScheduleMode === 'weekly' ? 'border-[color:var(--t-brand)] bg-[color:var(--t-brand-dim)] text-[var(--t-brand)]' : 'border-border text-text-secondary'">每周固定时间</button>
          </div>
          <div v-if="loopScheduleMode === 'interval'" class="flex items-center gap-2">
            <span class="text-xs text-text-muted shrink-0">上一轮完成后，间隔</span>
            <t-input v-model="loopIntervalMin" type="number" :min="1" :max="10080" class="w-24" aria-label="循环间隔分钟数" />
            <span class="text-xs text-text-muted shrink-0">分钟开始下一轮</span>
          </div>
          <div v-else class="space-y-2">
            <div>
              <label for="broadcast-schedule-timezone" class="block text-xs font-semibold text-text-secondary mb-1">时区（IANA）</label>
              <t-input id="broadcast-schedule-timezone" v-model="scheduleTimezone" placeholder="例如 Asia/Shanghai 或 America/Los_Angeles" class="w-full" />
            </div>
            <div>
              <label for="broadcast-schedule-time" class="block text-xs font-semibold text-text-secondary mb-1">固定执行时间</label>
              <input id="broadcast-schedule-time" v-model="scheduleLocalTime" type="time" class="w-full rounded-lg border border-border bg-bg-card px-3 py-2 text-sm text-text-primary" />
            </div>
            <div v-if="loopScheduleMode === 'weekly'">
              <span class="block text-xs font-semibold text-text-secondary mb-1">执行星期</span>
              <div class="grid grid-cols-7 gap-1" role="group" aria-label="执行星期">
                <button v-for="day in weekDayOptions" :key="day.value" type="button" @click="toggleScheduleWeekday(day.value)" :aria-pressed="scheduleWeekdays.includes(day.value)"
                  class="rounded-md border py-2 text-[11px]"
                  :class="scheduleWeekdays.includes(day.value) ? 'border-[color:var(--t-brand)] bg-[color:var(--t-brand-dim)] text-[var(--t-brand)]' : 'border-border text-text-secondary'">{{ day.label }}</button>
              </div>
            </div>
            <p class="text-[10px] text-text-muted">按所选时区的当地时间执行；夏令时不存在的时刻会跳过当天，重复时刻只执行第一次，避免同一天重复发送。</p>
          </div>
          <div>
            <label for="broadcast-first-run" class="block text-xs font-semibold text-text-secondary mb-1">首次执行时间（可选）</label>
            <input id="broadcast-first-run" v-model="firstRunLocal" type="datetime-local" class="w-full rounded-lg border border-border bg-bg-card px-3 py-2 text-sm text-text-primary" />
            <p class="text-[10px] text-text-muted mt-1">留空时，日历模式自动计算下一次符合条件的时间；间隔模式立即开始。</p>
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
            <div>
              <label for="broadcast-loop-max-rounds" class="block text-xs font-semibold text-text-secondary mb-1">最大执行轮数</label>
              <t-input id="broadcast-loop-max-rounds" v-model="loopMaxRounds" type="number" :min="0" :max="10000" placeholder="0 表示不限" class="w-full" />
              <p class="text-[10px] text-text-muted mt-1">完成指定轮数后自动停止</p>
            </div>
            <div>
              <label for="broadcast-loop-end-at" class="block text-xs font-semibold text-text-secondary mb-1">自动结束时间（可选）</label>
              <input id="broadcast-loop-end-at" v-model="loopEndAt" type="datetime-local" class="w-full rounded-lg border border-border bg-bg-card px-3 py-2 text-sm text-text-primary" />
              <p class="text-[10px] text-text-muted mt-1">到期后不再安排下一轮</p>
            </div>
          </div>
          <p class="text-[11px] text-text-muted leading-relaxed">循环任务会重复执行相同的账号分配、群组和消息配置。可在任务列表中暂停或停止；暂停后可继续。</p>
        </div>
        <p v-else class="text-xs text-text-muted mt-1">本任务只执行一轮，完成后自动结束。</p>
      </div>

    </div>
    <template #footer>
      <div class="flex items-center justify-end gap-3">
        <t-button type="button" class="ui-t-button ui-t-button--secondary" variant="outline" theme="default" @click="showCreate = false">取消</t-button>
        <t-button type="button" class="ui-t-button ui-t-button--primary" theme="primary" :loading="submitting" :disabled="!canSubmit" @click="submit">创建任务</t-button>
      </div>
    </template>
    </Modal>

  </div>
</template>

<script setup lang="ts">
import { parseTargets } from '@/utils/importTargets'
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAccountStore } from '@/stores/account'
import { groupApi } from '@/api/group'
// R115: Group 死类型导入已删(protoGroupList 删除后零使用)
// R111 已删 ProtocolPicker / campaignApi / CAMPAIGN_STATUS 导入(协议协同死代码)
import { useGroupStore } from '@/stores/group'
import { taskApi } from '@/api/task'
import { settingsApi, type TextDict } from '@/api/settings'
import { useAuthStore } from '@/stores/auth'
import SendCalendar from '@/components/SendCalendar.vue'
import { mediaApi } from '@/api/media'
import { Loader2, MessageSquare, Plus } from 'lucide-vue-next'
import TargetSelector from '@/components/selectors/TargetSelector.vue'
import { toast } from '@/utils/toast'
// LiveTaskFeed 实时卡片已移除(与列表重复), 保留组件文件供其他页面使用
import Modal from '@/components/ui/Modal.vue'
import ProgressRing from '@/components/ui/ProgressRing.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { useTaskEvents } from '@/composables/useTaskEvents'

const route = useRoute()
const router = useRouter()
const accountStore = useAccountStore()
const currentAccount = computed(() => accountStore.currentAccount)
const accName = (a: any) => a?.first_name || a?.remark || a?.phone || (a?.account_id || '').slice(0, 8)

// ===== tab 与发送模式状态(2026-10-20: 此前多轮编辑事故把这块定义整体冲丢, 模板引用 undefined
// 导致 tab 栏/循环开关静默不显示。此处集中补回, 以后动这个文件先看这里) =====
const activeTab = ref<'tasks' | 'calendar'>('tasks')
const pageTabs = [{ key: 'tasks', label: '任务列表' }, { key: 'calendar', label: '发送日历' }] as const
const isLoop = ref(false)
const loopIntervalMin = ref(60)
const loopScheduleMode = ref<'interval' | 'daily' | 'weekly'>('interval')
const scheduleTimezone = ref(Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai')
const scheduleLocalTime = ref('09:00')
const scheduleWeekdays = ref<number[]>([1, 2, 3, 4, 5])
const firstRunLocal = ref('')
const weekDayOptions = [{ value: 1, label: '一' }, { value: 2, label: '二' }, { value: 3, label: '三' }, { value: 4, label: '四' }, { value: 5, label: '五' }, { value: 6, label: '六' }, { value: 7, label: '日' }]
function toggleScheduleWeekday(day: number) {
  scheduleWeekdays.value = scheduleWeekdays.value.includes(day)
    ? scheduleWeekdays.value.filter(value => value !== day)
    : [...scheduleWeekdays.value, day].sort((a, b) => a - b)
}
function loopScheduleLabel(payload: any) {
  const mode = payload?.schedule_mode
  if (mode === 'daily') return `每天 ${payload?.local_time || ''} · ${payload?.timezone || ''}`
  if (mode === 'weekly') {
    const days = (payload?.weekdays || []).map((value: number) => weekDayOptions.find(day => day.value === value)?.label).filter(Boolean).join('、')
    return `每周${days} ${payload?.local_time || ''} · ${payload?.timezone || ''}`
  }
  return `${payload?.loop_interval_minutes || 60} 分钟`
}
const loopMaxRounds = ref(0)
const loopEndAt = ref('')
const authStore = useAuthStore()
const loopFeature = computed(() => {
  const pf: any = (authStore.tenant as any)?.plan_features
  return pf?.loop_broadcast === true
})

const selectedTargets = ref<string[]>([])
// R118: 群名显示 —— 目标 ID → 群标题映射; 缓存缺失时回退 ID 尾号
const showTargetNames = ref(false)
const targetTitle = (id: string) => {
  const g = (groups.value || []).find((x: any) => String(x.tg_chat_id || x.id) === String(id))
  if (!g) return '…' + String(id).slice(-6)
  return g.title || ('群 ' + String(g.tg_chat_id || g.id).slice(-6))
}
// R101 P1-切片1: 多号执行 —— 后端广播任务原生支持 account_ids 数组(笛卡尔模式:
// 每个账号对全部目标各发一遍)。默认勾选当前账号, 可勾选多个在线账号。
const execAccounts = ref<string[]>([])

// R151: 自动同步群组 —— 执行账号就绪即 ensure(有缓存秒显; 无缓存/过期后台自动同步)
watch(execAccounts, (list) => {
  for (const id of list || []) groupStore.ensure(id)
}, { immediate: true })

// 选择器同步完成后回填(覆盖 mutation 总线擦除的空窗)
function onSelectorRefresh() {
  for (const id of execAccounts.value) groupStore.ensure(id)
}
watch(currentAccount, (acc: any) => {
  if (acc?.account_id && !execAccounts.value.includes(acc.account_id)) {
    execAccounts.value = [acc.account_id, ...execAccounts.value.filter(id => id !== acc.account_id)]
  }
}, { immediate: true })
function toggleExec(id: string) {
  const i = execAccounts.value.indexOf(id)
  if (i >= 0) execAccounts.value.splice(i, 1)
  else execAccounts.value.push(id)
}
const onlineAccounts = computed(() => (accountStore.accounts || []).filter((a: any) => a.status === 'online'))

// R101 P1-第二刀: 覆盖预览 —— 拉全租户群(带 account_id), 计算"每个目标有哪些账号已加入",
// 提交前摊开: 可发/预计失败/完全无账号加入的群。
const _allGroups = ref<any[]>([])
let allGroupsRequestId = 0
async function loadAllGroups() {
  const requestId = ++allGroupsRequestId
  // R182.4: request.ts 已把列表响应统一包装为 {list,total} 对象(P0-35 审计修复), 不能当数组直用;
  // 直接 for...of 会抛 "not iterable" 导致群发页整页爆红。兼容数组/包装两种形态。
  try {
    const res: any = await groupApi.list()
    if (requestId !== allGroupsRequestId || !showCreate.value) return
    _allGroups.value = Array.isArray(res) ? res : (res?.list ?? [])
  } catch {
    if (requestId === allGroupsRequestId && showCreate.value) _allGroups.value = []
  }
}
function groupNameOf(t: string): string {
  const g: any = _allGroups.value.find((x: any) =>
    String(x.group_id) === t || String(x.tg_chat_id) === t || x.username === t)
  return g ? (g.title || g.username || t) : t
}
const sendStrategy = ref<'assign' | 'all'>('assign')
const coverage = computed(() => {
  const joinedBy = new Map<string, Set<string>>()
  // 与选择器使用同一份按账号缓存合并后的群组数据。
  // 全租户群组接口不包含 account_id，不能拿它反推入群账号。
  for (const g of groups.value as any[]) {
    const keys = [g.group_id, g.tg_chat_id, g.username, g.tg_link].filter(Boolean).map(String)
    for (const k of keys) {
      if (!joinedBy.has(k)) joinedBy.set(k, new Set())
      for (const accountId of (g._joinedAccounts || [])) {
        joinedBy.get(k)!.add(String(accountId))
      }
    }
  }
  const selected = new Set(execAccounts.value)
  const targetsByAccount = new Map<string, string[]>()
  let ok = 0
  let assignmentCursor = 0
  const unreachable: string[] = []
  for (const target of selectedTargets.value) {
    const eligible = [...(joinedBy.get(String(target)) || [])]
      .filter(id => selected.has(id))
      .sort((a, b) => execAccounts.value.indexOf(a) - execAccounts.value.indexOf(b))
    if (!eligible.length) {
      unreachable.push(groupNameOf(target))
      continue
    }
    const assigned = sendStrategy.value === 'all'
      ? eligible
      : [eligible[assignmentCursor++ % eligible.length]]
    for (const accountId of assigned) {
      const list = targetsByAccount.get(accountId) || []
      list.push(target)
      targetsByAccount.set(accountId, list)
      ok++
    }
  }
  const mappings = [...targetsByAccount.entries()].map(([account_id, targets]) => ({ account_id, targets }))
  return { ok, unreachable, mappings }
})
// R102: 预计耗时 = 总条数 × 平均间隔(秒) / 60, 向上取整
const estMinutes = computed(() => {
  const total = coverage.value.ok
  if (!total) return 0
  const avgSec = (intervalMin.value + intervalMax.value) / 2
  return Math.max(1, Math.ceil((total * avgSec) / 60))
})
const showImport = ref(false)
const importText = ref('')

// 解析导入文本: 每行一个目标, 支持 @用户名 / t.me链接 / 纯ID
function applyImportText() {
  const items = parseTargets(importText.value)
  if (!items.length) { toast({ type: 'error', title: '没有可识别的目标' }); return }
  selectedTargets.value = [...new Set([...selectedTargets.value, ...items])]
  toast({ type: 'success', title: `已添加 ${items.length} 个目标` })
  importText.value = ''
  showImport.value = false
}

function onImportFile(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  const r = new FileReader()
  r.onload = () => { importText.value = String(r.result || ''); applyImportText() }
  r.readAsText(f)
}
const message = ref('')
const textDicts = ref<TextDict[]>([])
const selectedTextDictId = ref('')
const showTextSnippets = ref(false)
const selectedTextDict = computed(() => {
  const dict = textDicts.value.find(d => d.dict_id === selectedTextDictId.value && d.enabled !== false)
  if (!dict) return null
  const entries = Array.isArray(dict.entries) ? dict.entries.filter(entry => entry.enabled) : []
  return { ...dict, items: entries.length ? entries.map(entry => entry.word) : (dict.entries?.length ? [] : dict.items || []) }
})
async function loadTextSnippets() {
  if (!authStore.tenant?.plan_features?.dict) return
  try { const res = await settingsApi.getTextDicts(); textDicts.value = (res.list || []).map(d => ({ ...d, enabled: d.enabled !== false, entries: d.entries || [] })) }
  catch (e) { console.warn('加载文本素材失败', e) }
}
function insertTextSnippet(item: string) {
  if (!item) return
  message.value = message.value.trim() ? message.value.trimEnd() + String.fromCharCode(10) + item : item
}

// 消息草稿按租户 + 当前账号隔离，避免切换账号或登录其他租户时串用消息内容。
const draftSaved = ref(false)
// 草稿按租户 + 当前账号隔离，避免切换账号或登录其他租户时串用消息内容。
let draftTimer: ReturnType<typeof setTimeout> | null = null
const draftKeyFor = (account: any) => {
  const tenant: any = authStore.tenant
  const tenantId = String(tenant?.tenant_id ?? 'tenant')
  const accountId = String(account?.account_id ?? 'unselected')
  return 'broadcast_msg_draft:' + tenantId + ':' + accountId
}
const activeDraftKey = ref(draftKeyFor(currentAccount.value))
function persistDraft(key: string, value: string) {
  try {
    if (value.trim()) localStorage.setItem(key, value)
    else localStorage.removeItem(key)
  } catch {}
}
function restoreDraft(key: string, notify = false) {
  try {
    const saved = localStorage.getItem(key) || ''
    message.value = saved
    draftSaved.value = !!saved.trim()
    if (saved.trim() && notify) {
      toast({ type: 'info', title: '草稿已恢复', message: '已恢复当前账号上次未发送的草稿' })
    }
  } catch {
    message.value = ''
    draftSaved.value = false
  }
}
watch(message, (value) => {
  draftSaved.value = false
  if (draftTimer) clearTimeout(draftTimer)
  draftTimer = setTimeout(() => {
    persistDraft(activeDraftKey.value, value)
    draftSaved.value = !!value.trim()
    draftTimer = null
  }, 800)
})
// 切换账号前先把当前编辑内容写回旧账号，再恢复新账号自己的草稿。
watch([currentAccount, () => authStore.tenant?.tenant_id], ([account]) => {
  if (draftTimer) {
    clearTimeout(draftTimer)
    draftTimer = null
  }
  persistDraft(activeDraftKey.value, message.value)
  activeDraftKey.value = draftKeyFor(account)
  restoreDraft(activeDraftKey.value, true)
})
onMounted(() => restoreDraft(activeDraftKey.value, true))
onUnmounted(() => {
  if (draftTimer) clearTimeout(draftTimer)
  persistDraft(activeDraftKey.value, message.value)
})
// 默认 [5,12] 与后端强制下限及协议群发页一致；原先 [4,300] 会让协同任务
// 平均 2.5 分钟才发一条，看起来像"群发没反应、账号全待机"
const intervalMin = ref(5)
const intervalMax = ref(12)

// 定时启动(可选): datetime-local 值原样提交, 后端支持 "2006-01-02T15:04" 格式
const scheduledAtTs = ref<number | null>(null)
// ===== 协议号协同模式: 勾选多个号, 走协同引擎(分工/限流换人/阵亡替补) =====
// R111 已删 isProtocol 计算属性(accountStore.view 为死常量, 恒 'phone')
// R111: protocolIds 等协议协同变量已删(死代码, view 恒 'phone')
// R111 已删 protoGroups
// R111 已删 protoGroupList / toggleProtoGroup / protocolTargets / protocolKind(协议协同死代码)
// R111 已删 campaigns / loadCampaigns(协议协同死代码)
const submitting = ref(false)
// 任务优先级: 5=普通 8=优先插队(队列按 priority DESC 领取, 紧急任务可插队)
const priority = ref(5)

// P1-35: 创建成功后重置整个表单(消息/目标/媒体/定时/循环全部归默认),
// 防止下次创建误带上次内容
function resetCreateForm() {
  message.value = ''
  selectedTargets.value = []
  showTargetNames.value = false
  showImport.value = false
  importText.value = ''
  intervalMin.value = 5
  intervalMax.value = 12
  priority.value = 5
  isLoop.value = false
  loopIntervalMin.value = 60
  loopScheduleMode.value = 'interval'
  scheduleTimezone.value = Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai'
  scheduleLocalTime.value = '09:00'
  scheduleWeekdays.value = [1, 2, 3, 4, 5]
  firstRunLocal.value = ''
  loopMaxRounds.value = 0
  loopEndAt.value = ''
  scheduledAtTs.value = null
  clearMedia()
  sendStrategy.value = 'assign'
  // 执行账号选择保留为下一次创建的默认账号池；目标和消息等一次性内容全部清空。
}
const validating = ref(false)
let precheckRequestId = 0

// 发送前目标预检：只允许结果作用于发起时的同一组目标，避免用户编辑目标期间旧结果误删新选择。
async function precheckTargets() {
  if (validating.value || !selectedTargets.value.length) return
  const requestId = ++precheckRequestId
  const targetSnapshot = [...selectedTargets.value]
  validating.value = true
  try {
    const res = await taskApi.validateTargets(targetSnapshot)
    const unchanged = targetSnapshot.length === selectedTargets.value.length &&
      targetSnapshot.every((target, index) => target === selectedTargets.value[index])
    if (requestId !== precheckRequestId || !unchanged) return
    if (res.invalid?.length) {
      const invalidSet = new Set(res.invalid.map(i => i.target))
      selectedTargets.value = targetSnapshot.filter(t => !invalidSet.has(t))
      toast({ type: 'warning', title: `已剔除 ${res.invalid.length} 个无效目标`, message: res.invalid.slice(0, 5).map(i => `${i.target}(${i.reason})`).join('；') + (res.invalid.length > 5 ? '…' : '') })
    } else {
      toast({ type: 'success', title: `全部 ${res.total} 个目标有效` })
    }
  } catch {
    if (requestId === precheckRequestId) toast({ type: 'error', title: '预检失败', message: '可忽略预检直接创建' })
  } finally {
    if (requestId === precheckRequestId) validating.value = false
  }
}

// ===== 模态框化: 本页任务列表 + 新建弹窗 =====
const showCreate = ref(false)
// R101 P1-第二刀: 打开创建弹窗即预拉全租户群(覆盖预览数据源)
watch(showCreate, (v) => {
  if (v) void loadAllGroups()
  else allGroupsRequestId++
})
const tasks = ref<any[]>([])
const tasksLoading = ref(true)
const tasksError = ref('')

let tasksRequestId = 0
async function fetchTasks() {
  const requestId = ++tasksRequestId
  const accountId = currentAccount.value?.account_id
  if (!accountId) {
    tasks.value = []
    tasksError.value = ''
    tasksLoading.value = false
    return
  }
  tasksLoading.value = true
  try {
    const res: any = await taskApi.list({ type: 'broadcast', account_id: accountId, page: 1, page_size: 20 } as any)
    if (requestId !== tasksRequestId || currentAccount.value?.account_id !== accountId) return
    tasks.value = res.list || []
    tasksError.value = ''
  } catch (e: any) {
    if (requestId === tasksRequestId && currentAccount.value?.account_id === accountId) {
      tasksError.value = e?.message || '加载失败, 点击重试'
    }
  } finally {
    if (requestId === tasksRequestId && currentAccount.value?.account_id === accountId) {
      tasksLoading.value = false
    }
  }
}
useAutoRefresh(fetchTasks, 15000)

// SSE 实时: 进度事件即时更新对应卡片, 状态事件刷新列表——亚秒级, 不再等轮询
useTaskEvents((evt: any) => {
  if (evt.type === 'task_status') { if (!evt.task_type || evt.task_type === 'broadcast') fetchTasks(); return }
  if (evt.progress && evt.task_id) {
    const t = tasks.value.find((x: any) => x.task_id === evt.task_id)
    if (t) t.progress = evt.progress
  }
})
watch(currentAccount, fetchTasks)

const statusLabelMap: Record<string, string> = {
  pending: '等待调度', queued: '排队中', running: '运行中',
  paused: '已暂停', completed: '已完成', failed: '失败', cancelled: '已取消',
}
function statusLabel(s: string) { return statusLabelMap[s] || s }
function statusClass(s: string): string {
  return ({
    pending: 'bg-[color:var(--t-warn-dim)] text-[var(--t-warn)]', queued: 'bg-[color:var(--t-warn-dim)] text-[var(--t-warn)]',
    running: 'bg-[color:var(--t-brand-dim)] text-[var(--t-brand)]', paused: 'bg-[color:var(--t-warn-dim)] text-[var(--t-warn)]', completed: 'bg-[color:var(--t-ok-dim)] text-[var(--t-ok)]',
    failed: 'bg-[color:var(--t-err-dim)] text-[var(--t-err)]', cancelled: 'bg-text-muted/10 text-text-muted',
  } as any)[s] || 'bg-bg-hover text-text-muted'
}
function progOf(t: any) {
  let p = t.progress
  if (typeof p === 'string') { try { p = JSON.parse(p) } catch { p = {} } }
  return { total: p?.total || 0, done: p?.done || 0, failed: p?.failed || 0 }
}
function loopRoundHistory(t: any) {
  let payload = t?.payload
  if (typeof payload === 'string') {
    try { payload = JSON.parse(payload) } catch { payload = {} }
  }
  const history = payload?.round_history
  return Array.isArray(history) ? history.slice(-3).reverse() : []
}
// 2026-10-21 修正: 进度按"已处理"(成功+失败)——失败也是处理结果,
// 旧口径 done/total 会让"已完成"任务停在 51%(48成功+47失败=95 全处理完却只显示一半)
function percentOf(t: any) {
  const pr = progOf(t)
  return pr.total ? Math.min(100, Math.round(((pr.done + (pr.failed || 0)) / pr.total) * 100)) : 0
}
function formatTime(iso: string) { return iso ? new Date(iso).toLocaleString('zh-CN') : '' }
const pullRefreshing = ref(false)
async function onPullRefresh() {
  await fetchTasks()
  pullRefreshing.value = false
}

async function retryFailed(id: string) {
  try { await taskApi.retry(id, 'failed'); fetchTasks() }
  catch (e: any) { toast({ type: 'error', title: '重试失败', message: e?.message }) }
}
async function stopTask(id: string) {
  try { await taskApi.cancel(id); fetchTasks() }
  catch (e: any) { toast({ type: 'error', title: '停止失败', message: e?.message }) }
}
async function pauseTask(id: string) {
  try {
    await taskApi.pause(id)
    toast({ type: 'success', title: '任务已暂停' })
    await fetchTasks()
  } catch (e: any) { toast({ type: 'error', title: '暂停失败', message: e?.message }) }
}
async function resumeTask(id: string) {
  try {
    await taskApi.resume(id)
    toast({ type: 'success', title: '任务已继续' })
    await fetchTasks()
  } catch (e: any) { toast({ type: 'error', title: '继续失败', message: e?.message }) }
}
const showTargetSelector = ref(false)
// 响应式绑定 groupStore：后台同步完成后写缓存即自动刷新选择器
// （原实现是快照赋值：页面加载后同步完成的群组永远不会出现，除非整页刷新）
// R108: 群并集 —— 取"所选执行账号"的群合并去重(按 group_id), 标注哪些号已加入。
// 顶栏切换器唯一剩余作用(给选群提供账号上下文)由此消除, 之后可删切换器。
const groups = computed<any[]>(() => {
  const byId = new Map<string, any>()
  for (const accId of execAccounts.value) {
    const accountGroups = (groupStore.get(accId) || []) as any[]
    accountGroups.forEach((g: any, index: number) => {
      // 无 Telegram ID/username 的异常记录也必须有稳定 key；Math.random 会导致每次
      // computed 重算都改变 key，让选择器整批卸载重建并丢失焦点/滚动位置。
      const fallback = [accId, g.title || g.name || 'group', index].join(':')
      const key = String(g.group_id || g.tg_chat_id || g.username || fallback)
      if (byId.has(key)) {
        const ex = byId.get(key)
        if (!ex._joinedAccounts.includes(accId)) ex._joinedAccounts.push(accId)
      } else {
        byId.set(key, { ...g, _joinedAccounts: [accId] })
      }
    })
  }
  return [...byId.values()]
})
// 所选账号变化时预拉每个号的群缓存(并集的数据源)
watch(execAccounts, (ids) => { ids.forEach(id => groupStore.ensure(id)) }, { deep: true })

// 富媒体与链接按钮
const mediaType = ref('')
const mediaUrl = ref('')
const mediaPath = ref('')
const uploadingMedia = ref(false)

let mediaUploadRequestId = 0
async function onMediaFile(e: Event) {
  const input = e.target as HTMLInputElement
  const f = input.files?.[0]
  if (!f) return
  const requestId = ++mediaUploadRequestId
  uploadingMedia.value = true
  try {
    const res: any = await mediaApi.uploadMedia(f)
    // 用户可能在上传期间点了清除或选择了另一份文件；旧响应不得复活旧媒体。
    if (requestId !== mediaUploadRequestId) return
    mediaPath.value = res.path || res
    toast({ type: 'success', title: '媒体上传成功' })
  } catch (err: any) {
    if (requestId === mediaUploadRequestId) {
      toast({ type: 'error', title: '上传失败', message: err?.message })
    }
  } finally {
    if (requestId === mediaUploadRequestId) uploadingMedia.value = false
    input.value = ''
  }
}
function clearMedia() {
  mediaUploadRequestId++
  uploadingMedia.value = false
  mediaType.value = ''
  mediaUrl.value = ''
  mediaPath.value = ''
}

// R110: 协议号协同死分支已删(view 为死常量, 该判断恒 false)。统一多号池校验。
const canSubmit = computed(() => {
  if (!message.value.trim().length) return false
  return selectedTargets.value.length > 0 && execAccounts.value.length > 0
})

const groupsLoading = ref(false)
// 群组走缓存 store: 有缓存零请求, 只有选择器里点"同步"才调接口
const groupStore = useGroupStore()

let groupsRequestId = 0
async function fetchData() {
  const accountId = currentAccount.value?.account_id
  const requestId = ++groupsRequestId
  if (!accountId) {
    groupsLoading.value = false
    return
  }
  groupsLoading.value = true
  try {
    // ensure 会把结果写入 groupStore 缓存，上方 computed 自动同步到选择器
    await groupStore.ensure(accountId)
  } catch {
    // 缓存加载失败由选择器空态/手动同步承接。
  } finally {
    if (requestId === groupsRequestId && currentAccount.value?.account_id === accountId) {
      groupsLoading.value = false
    }
  }
}

// 首次加载(空列表+加载中)视为正在同步群组, 与监控规则选择器的提示保持一致
const groupsSyncStatus = computed(() =>
  groupsLoading.value && groups.value.length === 0 ? 'syncing' : 'idle'
)

async function submit() {
  // loading 状态不应只依赖按钮组件；并发点击/键盘触发也必须在业务层幂等拦截。
  if (submitting.value || uploadingMedia.value || !canSubmit.value) return
  const accountIds = [...new Set(execAccounts.value)]
  const targetIds = [...new Set(selectedTargets.value)]
  const messageText = message.value
  const minSeconds = Number(intervalMin.value)
  const maxSeconds = Number(intervalMax.value)
  const loopMinutes = Number(loopIntervalMin.value)
  const mediaSource = mediaPath.value || mediaUrl.value.trim()

  if (!accountIds.length || !targetIds.length) {
    toast({ type: 'warning', title: '请至少选择一个执行账号和一个发送目标' })
    return
  }
  if (coverage.value.mappings.length === 0) {
    toast({ type: 'warning', title: '没有可发送的账号群组组合', message: '请确认已加载群组数据，并至少有一个所选账号加入目标群组' })
    return
  }
  if (!Number.isInteger(minSeconds) || !Number.isInteger(maxSeconds) ||
      minSeconds < 1 || maxSeconds > 300 || minSeconds > maxSeconds) {
    toast({ type: 'error', title: '发送间隔无效', message: '最小间隔和最大间隔须为 1–300 秒，且最小值不能大于最大值' })
    return
  }
  if (!isLoop.value && scheduledAtTs.value && scheduledAtTs.value <= Date.now()) {
    toast({ type: 'error', title: '定时启动时间无效', message: '请选择一个未来时间，或清空定时立即执行' })
    return
  }
  if (isLoop.value && !loopFeature.value) {
    toast({ type: 'warning', title: '套餐未开通循环群发' })
    return
  }
  if (isLoop.value && loopScheduleMode.value === 'interval' && (!Number.isInteger(loopMinutes) || loopMinutes < 1 || loopMinutes > 10080)) {
    toast({ type: 'error', title: '循环间隔无效', message: '循环间隔须为 1–10080 分钟' })
    return
  }
  const maxRounds = Number(loopMaxRounds.value)
  if (isLoop.value && (!Number.isInteger(maxRounds) || maxRounds < 0 || maxRounds > 10000)) {
    toast({ type: 'error', title: '最大轮数无效', message: '请输入 0–10000 的整数，0 表示不限' })
    return
  }
  if (isLoop.value && loopEndAt.value && new Date(loopEndAt.value).getTime() <= Date.now()) {
    toast({ type: 'error', title: '结束时间无效', message: '结束时间必须晚于当前时间' })
    return
  }
  if (isLoop.value && loopScheduleMode.value !== 'interval') {
    if (!scheduleTimezone.value.trim() || !/^([01]\d|2[0-3]):[0-5]\d$/.test(scheduleLocalTime.value)) {
      toast({ type: 'error', title: '日历排期无效', message: '请填写有效的 IANA 时区和 HH:MM 执行时间' })
      return
    }
    if (loopScheduleMode.value === 'weekly' && scheduleWeekdays.value.length === 0) {
      toast({ type: 'error', title: '请选择执行星期' })
      return
    }
  }
  if (!mediaType.value && mediaSource) {
    toast({ type: 'warning', title: '请选择媒体类型', message: '已提供媒体文件或链接，但未选择图片或视频类型' })
    return
  }
  if (mediaType.value && !mediaSource) {
    toast({ type: 'error', title: '缺少媒体文件', message: '请填写媒体链接或上传文件，或将媒体类型改回「无」' })
    return
  }

  submitting.value = true
  try {
    let payload: Record<string, any>
    if (isLoop.value) {
      // 后端 CreateScheduledBroadcast 单独处理循环任务；普通 broadcast 的 DTO 不识别 is_loop。
      payload = {
        type: 'scheduled_broadcast',
        account_ids: accountIds,
        target_ids: targetIds,
        payload: {
          message: messageText,
          ...(mediaSource ? { media: { type: mediaType.value, source: mediaSource } } : {}),
        },
        mappings: coverage.value.mappings,
        is_loop: true,
        loop_interval_minutes: loopMinutes,
        schedule_mode: loopScheduleMode.value,
        ...(loopScheduleMode.value !== 'interval' ? {
          timezone: scheduleTimezone.value.trim(),
          local_time: scheduleLocalTime.value,
          ...(loopScheduleMode.value === 'weekly' ? { weekdays: scheduleWeekdays.value } : {}),
        } : {}),
        ...(firstRunLocal.value ? (loopScheduleMode.value === 'interval'
          ? { first_run_at: new Date(firstRunLocal.value).toISOString() }
          : { first_run_local: firstRunLocal.value }) : {}),
        min_interval_seconds: minSeconds,
        max_interval_seconds: maxSeconds,
        max_rounds: maxRounds,
        ...(loopEndAt.value ? { end_at: new Date(loopEndAt.value).toISOString() } : {}),
      }
    } else {
      payload = {
        type: 'broadcast',
        account_ids: accountIds,
        target_ids: targetIds,
        message: messageText,
        interval_min: minSeconds,
        interval_max: maxSeconds,
        ...(coverage.value.mappings.length ? { mappings: coverage.value.mappings } : {}),
      }
      if (mediaSource) payload.media = { type: mediaType.value, source: mediaSource }
      if (scheduledAtTs.value) payload.scheduled_at = new Date(scheduledAtTs.value).toISOString()
    }

    await taskApi.create(payload)
    showCreate.value = false
    void fetchTasks()
    resetCreateForm()
    toast({ type: 'success', title: '任务创建成功', message: '实时进度见下方任务列表' })
  } catch (e: any) {
    // 失败不清空任何输入，用户可直接修正后重试。
    toast({ type: 'error', title: '创建失败', message: e?.message || '请稍后重试' })
  } finally {
    submitting.value = false
  }
}

onMounted(() => { fetchData(); fetchTasks(); loadTextSnippets() })
// 侧边栏切换账号: 重新拉取群组缓存
watch(currentAccount, () => { fetchData() })
</script>
