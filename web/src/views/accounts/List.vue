<template>
  <main class="min-w-0 w-full space-y-3 sm:space-y-4">
    <PageHeader
      eyebrow="运营控制中心 · 账号"
      title="账号管理"
      :subtitle="`共 ${(accountStore.accounts || []).length} 个账号`"
    />
    <!-- 操作栏(R141 新皮肤适配): 第一行双按钮+多选入口, 第二行搜索独占整行 -->
    <div class="grid gap-2 mb-3.5">
      <div class="flex flex-wrap gap-2">
        <t-button theme="primary" class="flex-1 min-w-0 ui-t-button ui-t-button--primary" @click="openAddModal">添加账号</t-button>
        <!-- 导入协议号入口已整合进"添加账号"弹窗(手机号登录|导入协议号 双模式), 独立按钮删除 -->
        <t-button theme="default" class="flex-1 min-w-0 ui-t-button ui-t-button--secondary" @click="showGroupModal = true">分组管理</t-button>
        <!-- 多选模式入口(桌面端; 移动端可长按列表行进入) -->
        <t-button theme="default" variant="outline" size="sm" class="ui-t-button ui-t-button--secondary ui-t-button--sm whitespace-nowrap ml-auto"
                @click="enterSelecting">多选</t-button>
      </div>
      <t-input v-model="search" placeholder="搜索手机号/备注" class="w-full">
        <template #prefix>
          <Search class="w-4 h-4" style="color: var(--t-text-3);" />
        </template>
      </t-input>
    </div>

    <!-- 全选行(仅多选模式显示; 操作按钮在下方悬浮栏) -->
    <div v-if="selecting && filteredAccounts.length" class="flex items-center gap-2 mb-2 px-1">
      <label class="flex items-center gap-1.5 text-xs cursor-pointer select-none" style="color: var(--t-text-2);">
        <input ref="selectAllRef" type="checkbox" :checked="allSelected"
               class="accent-[var(--t-brand)] w-4 h-4" @change="toggleSelectAll" />
        全选
      </label>
      <span v-if="selectedIds.size" class="text-xs" style="color: var(--t-text-3);">已选 {{ selectedIds.size }} 个</span>
      <t-button theme="primary" variant="text" size="sm" class="ui-t-button ui-t-button--text ui-t-button--sm ml-auto" @click="exitSelecting">完成</t-button>
    </div>

    <!-- 批量操作悬浮栏(R141 新皮肤适配): 实底浮层(深色模式不透字) + 令牌阴影 + 主题感知按钮底 -->
    <Transition name="batch-bar">
      <!-- 避开底部 Tab: 移动端 bottom-[92px](Tab 上方留隙), 桌面端 bottom-5 -->
      <div v-if="selectedIds.size" class="fixed bottom-[calc(92px_+_env(safe-area-inset-bottom,0px))] md:bottom-5 left-1/2 -translate-x-1/2 z-40 flex items-center gap-1.5 pl-4 pr-2 py-2 rounded-[var(--t-r-lg)] whitespace-nowrap max-w-[calc(100vw-32px)] overflow-x-auto scrollbar-hide"
           style="background: var(--t-card); border: 1px solid var(--t-line-strong); box-shadow: var(--t-shadow-pop);">
        <span class="text-xs font-semibold whitespace-nowrap" style="color: var(--t-text-1);">已选 <b class="tabular-nums" style="color: var(--t-brand);">{{ selectedIds.size }}</b> 项</span>
        <span class="w-px h-4 shrink-0" style="background: var(--t-line-strong);" />
        <t-button theme="default" variant="outline" size="sm" @click="batchSetPool('standby')" :disabled="batchingPool"
                class="ui-t-button ui-t-button--secondary ui-t-button--sm whitespace-nowrap">
          转备用
        </t-button>
        <t-button theme="primary" size="sm" @click="batchSetPool('active')" :disabled="batchingPool"
                class="ui-t-button ui-t-button--primary ui-t-button--sm whitespace-nowrap">
          启用
        </t-button>
        <t-button theme="default" variant="outline" size="sm" @click="openBatchProfile"
                :disabled="batchingProfile"
                class="ui-t-button ui-t-button--secondary ui-t-button--sm whitespace-nowrap">
          批量修改资料
        </t-button>
        <t-button theme="default" variant="outline" size="sm" @click="openClone"
                :disabled="cloning"
                class="ui-t-button ui-t-button--secondary ui-t-button--sm whitespace-nowrap">
          克隆资料
        </t-button>
        <button type="button" @click="exitSelecting" title="退出多选" aria-label="退出多选"
                class="ui-icon-btn shrink-0">
          ✕
        </button>
      </div>
    </Transition>

    <!-- 账号列表 -->
    <div v-if="loading" class="py-12">
      <t-loading size="small" loading class="inline-flex" />
    </div>
    <div v-else-if="!filteredAccounts.length" class="py-12">
      <EmptyState
        icon="Users"
        title="暂无账号"
        description="点击下方按钮添加你的第一个 Telegram 账号吧"
      >
        <t-button theme="primary" class="mt-4 ui-t-button ui-t-button--primary" @click="openAddModal">
          添加账号
        </t-button>
      </EmptyState>
    </div>
    <template v-else>
      <!-- 统计瓦片(R141 新皮肤适配): 横向滑动条(移动端不再折行挤瘪), 实底卡 + 令牌字号/状态色 -->
      <div class="stat-strip scrollbar-hide mb-3">
        <div class="ui-card px-4 py-3">
          <div class="flex items-center gap-1.5 text-[11px] font-medium" style="color: var(--t-text-3);"><span class="w-1.5 h-1.5 rounded-full" style="background: var(--t-text-3);" />已绑定账号</div>
          <div class="mt-1.5 leading-none font-extrabold tabular-nums" style="font-size: var(--t-fs-display); color: var(--t-text-1);" v-countup="(accountStore.accounts || []).length">0</div>
        </div>
        <div class="ui-card px-4 py-3">
          <div class="flex items-center gap-1.5 text-[11px] font-medium" style="color: var(--t-text-3);"><span class="w-1.5 h-1.5 rounded-full" style="background: var(--t-brand);" />正常号</div>
          <div class="mt-1.5 leading-none font-extrabold tabular-nums" style="font-size: var(--t-fs-display); color: var(--t-brand);" v-countup="typeCount.phone">0</div>
        </div>
        <div class="ui-card px-4 py-3">
          <div class="flex items-center gap-1.5 text-[11px] font-medium" style="color: var(--t-text-3);"><span class="w-1.5 h-1.5 rounded-full" style="background: var(--t-warn);" />协议号</div>
          <div class="mt-1.5 leading-none font-extrabold tabular-nums" style="font-size: var(--t-fs-display); color: var(--t-warn);" v-countup="typeCount.protocol">0</div>
        </div>
        <div class="ui-card px-4 py-3">
          <div class="flex items-center gap-1.5 text-[11px] font-medium" style="color: var(--t-text-3);"><span class="w-1.5 h-1.5 rounded-full" style="background: var(--t-ok);" />在线账号</div>
          <div class="mt-1.5 leading-none font-extrabold tabular-nums" style="font-size: var(--t-fs-display); color: var(--t-ok);" v-countup="statusCount.online">0</div>
        </div>
        <div class="ui-card px-4 py-3">
          <div class="flex items-center gap-1.5 text-[11px] font-medium" style="color: var(--t-text-3);"><span class="w-1.5 h-1.5 rounded-full" style="background: var(--t-err);" />失败 / 风控</div>
          <div class="mt-1.5 leading-none font-extrabold tabular-nums" style="font-size: var(--t-fs-display); color: var(--t-err);">{{ statusCount.restricted + statusCount.failed }}</div>
        </div>
      </div>

      <!-- 风控轻提示(R141: 弱化, 无红底块, 小号陶橙文字) -->
      <div v-if="statusCount.online === 0" class="risk-hint">⚠ 无在线账号，任务无法执行</div>

      <!-- 账号列表(R141 新皮肤适配): 单容器 + 极细分割线, 两行信息, 长按多选 -->
      <div class="acc-group" :class="{ selecting }">


      <!-- R125: v-for 无模板层守卫 —— 防御全在数据层(pagedAccounts 元素过滤) + 42 处可选链。
           教训: template 包裹两次嵌套交错炸构建, v-if/v-for 同元素 Vue 读不到 item, 都不碰。 -->
      <div
        v-for="(acc, index) in pagedAccounts"
        :key="acc?.account_id ?? index"
        class="acc-row"
        :class="{ standby: acc?.pool_status === 'standby' }"
        :style="{ animationDelay: `${Math.min(index * 40, 300)}ms` }"
        role="group"
        :aria-label="`账号 ${acc.remark || acc.first_name || acc.phone || '未命名'}`"
        @touchstart.passive="onRowTouchStart($event, acc)"
        @touchmove.passive="onRowTouchMove"
        @touchend="onRowTouchEnd"
        @click="onRowClick(acc)">
        <!-- 复选框列(R141): 默认收起(width:0 不占位), 进入多选模式从左侧滑出 -->
        <div class="check-col">
          <input type="checkbox" :checked="selectedIds.has(acc?.account_id)"
                 class="acc-check"
                 @click.stop="toggleSelect(acc)" :aria-label="'选择 ' + (acc.remark || acc.phone)" />
        </div>
            <div class="relative group/avatar shrink-0">
              <div class="w-9 h-9 rounded-full bg-[color:var(--t-brand-dim)] flex items-center justify-center overflow-hidden">
                <img
                  v-if="avatarOf(acc)"
                  :src="avatarOf(acc)!"
                  :alt="acc.remark || acc.phone"
                  class="avatar-zoom w-full h-full object-cover"
                />
                <!-- R87-M6: 无TG头像时用"首字+品牌淡底"文字头像, 不再显示灰色小人图标 -->
                <div v-if="!avatarOf(acc)" class="w-full h-full bg-[color:var(--t-brand-dim)] flex items-center justify-center text-[var(--t-brand)] text-sm font-semibold">
                  {{ (acc.first_name || acc.phone || '?').slice(0, 1) }}
                </div>
              </div>
              <!-- 头像刷新: hover 浮现, 清路径→重连→上线钩子自动重拉 -->
              <button type="button" v-if="acc?.pool_status !== 'standby'"
                      @click.stop="refreshAvatar(acc)"
                      :disabled="avatarRefreshingId === acc?.account_id"
                      class="ui-icon-btn ui-icon-btn--xs absolute -bottom-0.5 -right-0.5 opacity-0 group-hover/avatar:opacity-100"
                      title="刷新头像" aria-label="刷新头像">
                <RefreshCw class="w-3 h-3" :class="avatarRefreshingId === acc?.account_id ? 'animate-spin' : ''" />
              </button>
            </div>
        <!-- 两行信息区(R141): 第一行名字/标签, 第二行状态+号码+发送报告 -->
        <div class="acc-main">
          <!-- 第一行: 备注名 + 类型标签 + 备用标签 + 任务摘要 -->
          <div class="acc-title">
            <span class="font-medium truncate" style="font-size: var(--t-fs-small);">{{ acc.remark || acc.first_name || acc.phone }}</span>
            <!-- R131-2 用户拍板: 健康分标签从卡片删除(信息噪音, 详情里仍有) -->
            <span class="text-[10px] px-1.5 py-0.5 rounded font-medium shrink-0"
                  :class="isProtocolAcc(acc) ? 'bg-[color:var(--t-warn-dim)] text-[var(--t-warn)]' : 'bg-[color:var(--t-brand-dim)] text-[var(--t-brand)]'">{{ isProtocolAcc(acc) ? '协议号' : '正常号' }}</span>
            <span v-if="acc?.pool_status === 'standby'"
                  class="text-[10px] px-1.5 py-0.5 rounded shrink-0"
                  style="background: var(--t-surface-2); color: var(--t-text-3); border: 1px solid var(--t-line);">备用</span>
            <!-- R153: 登录验证码(10分钟内有效, 点击复制) -->
            <button type="button" v-if="loginCodeFresh(acc)"
                   class="text-[10px] px-1.5 py-0.5 rounded font-bold shrink-0 cursor-pointer"
                   style="background: color-mix(in srgb, var(--t-ok) 14%, transparent); color: var(--t-ok);"
                   :title="'复制验证码 ' + acc.last_login_code"
                   :aria-label="'复制账号 ' + (acc.remark || acc.phone) + ' 的登录验证码'"
                   @click.stop="copyLoginCode(acc)">&#128273; {{ acc.last_login_code }} {{ codeAge(acc) }}</button>
            <!-- 任务摘要: 该账号正在执行的任务(最多3个) -->
            <span v-if="taskSummary[acc?.account_id]?.length"
                  class="text-[11px] px-1.5 py-0.5 rounded bg-[color:var(--t-brand-dim)] text-[var(--t-brand)] whitespace-nowrap"
                  :title="taskSummary[acc?.account_id].join('、')">
              ⚡ {{ taskSummary[acc?.account_id].slice(0, 1).join('') }}<template v-if="taskSummary[acc?.account_id].length > 1"> 等{{ taskSummary[acc?.account_id].length }}个</template>
            </span>
          </div>
          <!-- 第二行(R141): 状态 + 手机号(完整显示) + 能力/Premium/分组 + 发送报告(右对齐) -->
          <div class="acc-meta">
            <i class="w-1.5 h-1.5 rounded-full shrink-0" :class="statusDotClass(acc?.status)"></i>
            <span>{{ accountStatusLabel(acc?.status) }}</span>
            <!-- R133 兜底: 协议号未回读前显示"TG 协议号", 正常号永有手机号; 完整显示不遮罩 -->
            <span class="tabular-nums truncate">{{ acc.phone || (acc.username ? '@' + acc.username : (isProtocolAcc(acc) ? 'TG 协议号' : '未绑定手机')) }}</span>
            <!-- 能力直查(v1.22): 体检结果落库, hover 显示能做什么/不能做什么 -->
            <span v-if="acc.capabilities && Object.keys(acc.capabilities).length > 1"
                  class="text-[10px] px-1.5 py-0.5 rounded whitespace-nowrap cursor-help"
                  style="background: var(--t-surface-2); color: var(--t-text-3); border: 1px solid var(--t-line);"
                  :title="capsSummary(acc.capabilities)">能力</span>
            <!-- Premium 徽标: 体检时从 TG 抓取并落库 capabilities -->
            <span v-if="acc.capabilities?.is_premium" title="Telegram Premium 会员"
                  class="text-[10px] px-1.5 py-0.5 rounded whitespace-nowrap"
                  style="background: color-mix(in srgb, var(--t-warn) 12%, transparent); color: var(--t-warn);">★ Premium</span>
            <span v-if="getGroupForAccount(acc?.account_id)"
                  class="text-[10px] px-1.5 py-0.5 rounded-full whitespace-nowrap"
                  :style="{ backgroundColor: getGroupForAccount(acc?.account_id)?.color + '20', color: getGroupForAccount(acc?.account_id)?.color }">
              {{ getGroupForAccount(acc?.account_id)?.name }}
            </span>
            <!-- 发送报告(R141): 该号累计成功/失败总数, 紧凑格式(1.5k / 2.3万); hover 看精确值(含跳过) -->
            <span v-if="acc?.account_id" class="send-report" :class="accStats[acc?.account_id] ? '' : 'opacity-60'"
                  :title="`累计成功 ${accStats[acc?.account_id]?.success || 0} · 失败 ${accStats[acc?.account_id]?.failed || 0} · 跳过 ${accStats[acc?.account_id]?.skipped || 0}`">
              <i>✓</i>{{ fmtCompact(accStats[acc?.account_id]?.success || 0) }}
              <i class="f">✗</i>{{ fmtCompact(accStats[acc?.account_id]?.failed || 0) }}
            </span>
          </div>
        </div>

        <!-- 卡片固定三个主操作：备用、检测、更多；其他低频操作统一收纳 -->
        <div class="flex items-center gap-1 shrink-0 self-center acc-actions">
          <t-button theme="default" variant="outline" size="sm" @click.stop="togglePool(acc)"
                  class="ui-t-button ui-t-button--secondary ui-t-button--sm whitespace-nowrap acc-action-btn">
            <component :is="acc?.pool_status === 'standby' ? Play : Archive" class="w-3.5 h-3.5" />
            {{ acc?.pool_status === 'standby' ? '启用' : '备用' }}
          </t-button>
          <t-button theme="default" variant="outline" size="sm" @click.stop="openProbe(acc)"
                  class="ui-t-button ui-t-button--secondary ui-t-button--sm whitespace-nowrap acc-action-btn">
            <ShieldCheck class="w-3.5 h-3.5" /> 检测
          </t-button>
          <div class="relative ml-auto">
            <!-- R130 "更多"文字按钮(用户: 比 ⋯ 图标更易发现), 胶囊底常显 -->
            <t-button theme="default" variant="outline" size="sm" @click.stop="toggleMenu(acc?.account_id)"
                    class="ui-t-button ui-t-button--secondary ui-t-button--sm shrink-0 more-btn"
                    aria-label="更多操作">更多</t-button>
            <!-- 移动端遮罩(点空白关闭面板); 桌面端无遮罩 -->
            <div v-if="openMenuId === acc?.account_id" class="fixed inset-0 z-[2490] sm:hidden"
                 style="background: rgba(0, 0, 0, 0.4);" @click.stop="openMenuId = ''"></div>
            <!-- 更多菜单(R141): 移动端 = 底部 ActionSheet(圆角上提, 遮罩关闭); 桌面端 = 向上弹出菜单 -->
            <div v-if="openMenuId === acc?.account_id"
                 class="acc-menu fixed z-[2500] py-1.5 rounded-xl sm:absolute sm:right-0 sm:bottom-full sm:mb-1"
                 style="background: var(--t-modal-bg); border: 1px solid var(--t-line-light);"
                 @click.stop>
              <div class="acc-menu-head">
                <div class="min-w-0">
                  <div class="text-sm font-semibold text-text-primary">账号操作</div>
                  <div class="text-[11px] text-text-muted truncate">{{ acc?.remark || acc?.first_name || acc?.phone || '当前账号' }}</div>
                </div>
                <button type="button" class="acc-menu-close" aria-label="关闭更多菜单"
                        @click.stop="openMenuId = ''">×</button>
              </div>

              <div class="acc-menu-section">
                <div class="acc-menu-label">连接与登录</div>
                <button type="button" v-if="acc?.status === 'online'"
                        @click="openMenuId = ''; disconnect(acc)" :disabled="disconnecting === acc?.account_id"
                        class="ui-menu-item w-full disabled:opacity-50">
                  <Unplug class="w-3.5 h-3.5" />断开连接
                </button>
                <button type="button" v-else-if="(acc?.status === 'pending_auth' || acc?.status === 'session_revoked') && acc?.source !== 'protocol'"
                        @click="openMenuId = ''; showAuthModal(acc)"
                        class="ui-menu-item w-full">
                  <KeyRound class="w-3.5 h-3.5" />{{ acc?.status === 'session_revoked' ? '重新登录' : '输入验证码' }}
                </button>
                <button type="button" v-if="acc?.pool_status !== 'standby' && (acc?.status === 'offline' || acc?.status === 'banned')"
                        @click="openMenuId = ''; reconnectAccount(acc)" :disabled="reconnectingId === acc?.account_id"
                        class="ui-menu-item w-full disabled:opacity-50">
                  <RefreshCw class="w-3.5 h-3.5" :class="reconnectingId === acc?.account_id ? 'animate-spin' : ''" />重新连接
                </button>
              </div>

              <div class="acc-menu-section">
                <div class="acc-menu-label">账号维护</div>
                <button type="button" @click="openMenuId = ''; refreshAvatar(acc)" :disabled="avatarRefreshingId === acc?.account_id"
                        class="ui-menu-item w-full disabled:opacity-50">
                  <RefreshCw class="w-3.5 h-3.5" :class="avatarRefreshingId === acc?.account_id ? 'animate-spin' : ''" />刷新头像
                </button>
                <button type="button" @click="openMenuId = ''; leaveAllGroups(acc)" :disabled="leavingAllId === acc?.account_id"
                        class="ui-menu-item w-full disabled:opacity-50">
                  <LogOut class="w-3.5 h-3.5" />退群
                </button>
                <button type="button" @click="openMenuId = ''; openTransfer(acc)" :disabled="transferRunning && transferSource?.account_id === acc?.account_id"
                        class="ui-menu-item w-full disabled:opacity-50">
                  <ArrowRightLeft class="w-3.5 h-3.5" />转移群
                </button>
              </div>

              <div class="acc-menu-section">
                <div class="acc-menu-label">资料与备注</div>
                <button type="button" @click="openMenuId = ''; openProfile(acc)"
                        class="ui-menu-item w-full">
                  <UserRound class="w-3.5 h-3.5" />修改资料
                </button>
                <button type="button" @click="openMenuId = ''; editRemark(acc)"
                        class="ui-menu-item w-full">
                  <Pencil class="w-3.5 h-3.5" />备注与代理
                </button>
              </div>

              <div class="acc-menu-danger">
                <button type="button" @click="openMenuId = ''; confirmDelete(acc)"
                        class="ui-menu-item ui-menu-item--danger w-full">
                  <Trash2 class="w-3.5 h-3.5" />删除账号
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
      </div>
    </template>

    <!-- 添加账号弹窗（步骤化） -->
    <Modal v-model="showAddModal" :title="addModalTitle">
      <!-- 步骤1：输入手机号 -->
      <div v-if="addStep === 'phone'" class="space-y-4">
        <!-- 添加方式切换: 手机号登录 | 协议号导入(把原 sessions 页导入功能整合进添加弹窗) -->
        <div class="ui-segmented w-full">
          <button type="button" class="ui-segmented__item flex-1" :class="{ 'is-active': addMode === 'phone' }" @click="addMode = 'phone'">手机号登录</button>
          <button type="button" class="ui-segmented__item flex-1" :class="{ 'is-active': addMode === 'protocol' }" @click="addMode = 'protocol'">导入协议号</button>
        </div>

        <template v-if="addMode === 'phone'">
        <div>
          <label for="add-phone" class="block text-sm text-text-secondary mb-1.5">手机号</label>
          <div class="flex items-center">
            <span class="inline-flex items-center px-3 h-[44px] rounded-l-lg bg-bg-card border border-r-0 border-border text-text-secondary text-sm">+</span>
            <input
              id="add-phone"
              v-model="newPhone"
              placeholder="86 13800138000"
              class="ui-input !w-auto flex-1 !min-h-[44px] !px-3 !py-2 !rounded-l-none !text-sm"
              @keyup.enter="getVerificationCode" aria-label="手机号" />
          </div>
          <p class="text-xs text-text-secondary mt-1">输入国际格式手机号，如 86 13800138000</p>
        </div>
        <div>
          <label for="add-remark" class="block text-sm text-text-secondary mb-1.5">备注（可选）</label>
          <input
            id="add-remark"
            v-model="newRemark"
            placeholder="如：主号-客服"
            class="ui-input !min-h-[44px] !text-sm" aria-label="备注" />
        </div>
        <div>
          <label for="add-group" class="block text-sm text-text-secondary mb-1.5">分组（可选）</label>
          <t-select v-model="newGroupId" :options="groupOptions" placeholder="不分组" />
        </div>
        </template>

        <!-- 协议号导入面板 -->
        <template v-else>
          <div>
            <label class="block text-sm text-text-secondary mb-1.5">Session 字符串（每行一个）</label>
            <textarea v-model="importText" rows="5" class="ui-input !text-sm font-mono" placeholder="每行粘贴一个 session 字符串，可批量导入" aria-label="session 字符串"></textarea>
          </div>
          <div>
            <label class="block text-sm text-text-secondary mb-1.5">或上传 session 文件（可多选）</label>
            <input type="file" multiple accept=".session,.see,.tdata,.zip,.rar,.7z,.json" class="w-full text-sm text-text-secondary file:mr-3 file:px-3 file:py-1.5 file:rounded-lg file:border-0 file:bg-bg-hover file:text-text-primary" @change="onImportFiles" aria-label="session 文件" />
            <p v-if="importFileList.length" class="text-xs text-[var(--t-brand)] mt-1">已选 {{ importFileList.length }} 个文件</p>
          </div>
          <p class="text-xs text-text-secondary">导入成功后账号自动上线（协议号保活模式）；备注对本次全部导入生效。</p>
        </template>
      </div>
      <!-- 步骤2：输入验证码 -->
      <div v-else-if="addStep === 'code'" class="space-y-4 text-center">
        <p class="text-text-secondary">验证码已发送到 <span class="font-medium text-text-primary">{{ newPhone }}</span></p>
        <input
          v-model="authCode"
          placeholder="请输入验证码"
          inputmode="numeric"
          pattern="[0-9]*"
          
          class="ui-input !min-h-[48px] !px-4 !py-3 !text-center !text-lg font-mono tracking-widest"
          maxlength="6"
          aria-label="验证码"
          @keyup.enter="submitVerificationCode"
        />
        <p v-if="codeError" class="text-sm text-[var(--t-err)]">{{ codeError }}</p>
        <t-button theme="primary" variant="text" size="sm" @click="resendCode" class="ui-t-button ui-t-button--text ui-t-button--sm">
          没收到？重新获取
        </t-button>
      </div>
      <!-- 步骤3：两步验证 -->
      <div v-else-if="addStep === '2fa'" class="space-y-4 text-center">
        <div class="w-12 h-12 rounded-full icon-glow flex items-center justify-center mx-auto">
          <Shield class="w-6 h-6 text-[var(--t-warn)]" />
        </div>
        <p class="text-text-secondary">该账号开启了两步验证</p>
        <input
          v-model="twoFAPassword"
          type="password"
          placeholder="请输入两步验证密码"
          
          class="ui-input !min-h-[48px] !px-4 !py-3 !text-center"
          aria-label="两步验证密码"
          @keyup.enter="submit2FA"
        />
        <p v-if="twoFAError" class="text-sm text-[var(--t-err)]">{{ twoFAError }}</p>
      </div>
      <template #footer>
        <div class="flex gap-3">
          <t-button theme="default" variant="outline" class="ui-t-button ui-t-button--secondary" @click="closeAddModal" aria-label="关闭">取消</t-button>
          <t-button class="ui-t-button ui-t-button--primary" theme="primary"
            v-if="addStep === 'phone' && addMode === 'phone'"
            :disabled="!newPhone || adding"
            @click="getVerificationCode"
          >
            <Loader2 v-if="adding" class="w-4 h-4 animate-spin" />
            {{ adding ? '发送中...' : '获取验证码' }}
          </t-button>
          <t-button class="ui-t-button ui-t-button--primary" theme="primary"
            v-if="addStep === 'phone' && addMode === 'protocol'"
            :disabled="importing || (!importText.trim() && importFileList.length === 0)"
            @click="doProtocolImport"
          >
            <Loader2 v-if="importing" class="w-4 h-4 animate-spin" />
            {{ importing ? '导入中...' : '导入协议号' }}
          </t-button>
          <t-button class="ui-t-button ui-t-button--primary" theme="primary"
            v-else-if="addStep === 'code'"
            
            :disabled="!authCode || submitting"
            @click="submitVerificationCode"
          >
            {{ submitting ? '提交中...' : '提交验证码' }}
          </t-button>
          <t-button class="ui-t-button ui-t-button--primary" theme="primary"
            v-else-if="addStep === '2fa'"
            
            :disabled="!twoFAPassword || submitting"
            @click="submit2FA"
          >
            {{ submitting ? '验证中...' : '完成登录' }}
          </t-button>
        </div>
      </template>
    </Modal>

    <!-- 分组管理弹窗 -->
    <Modal v-model="showGroupModal" title="账号分组管理">
      <div class="space-y-3">
        <div v-for="g in accountGroups" :key="g.group_id" class="flex items-center gap-2 p-3 rounded-lg bg-bg-card">
          <div class="w-3 h-3 rounded-full" :style="`background-color:${g.color || 'var(--t-text-3)'}`" />
          <span class="flex-1 text-sm">{{ g.name }}</span>
          <span class="text-xs text-text-secondary">{{ (g.account_ids || []).length }} 个账号</span>
          <button type="button" @click="editGroup(g)" class="ui-icon-btn" aria-label="编辑">
            <Pencil class="w-3.5 h-3.5 text-text-secondary" />
          </button>
          <button type="button" @click="deleteGroup(g)" class="ui-icon-btn" aria-label="删除分组">
            <Trash2 class="w-3.5 h-3.5 text-text-secondary hover:text-[var(--t-err)]" />
          </button>
        </div>
        <div v-if="!accountGroups.length" class="text-center text-sm text-text-secondary py-4">
          暂无分组，点击下方按钮添加~
        </div>
        <div class="flex gap-2 mt-3">
          <input
            id="group-name" v-model="newGroupName"
            placeholder="新分组名称..."
            class="ui-input !w-auto flex-1 !min-h-[36px] !px-2.5 !py-1.5 !text-sm" aria-label="新分组名称..." />
          <input
            id="group-color" v-model="newGroupColor"
            type="color"
            class="w-10 h-9 rounded cursor-pointer" aria-label="选择颜色" />
          <t-button class="ui-t-button ui-t-button--primary" theme="primary" @click="createGroup">添加</t-button>
        </div>
      </div>
    </Modal>

    <!-- 编辑分组弹窗 -->
    <Modal v-model="showEditGroup" title="编辑分组">
      <div class="space-y-3">
        <div>
          <label for="group-name" class="block text-sm text-text-secondary mb-1">分组名称</label>
          <input id="group-name" v-model="editGroupName" class="ui-input !min-h-[36px] !px-2.5 !py-1.5 !text-sm" aria-label="群组名称" />
        </div>
        <div>
          <label for="group-color" class="block text-sm text-text-secondary mb-1">颜色</label>
          <input id="group-color" v-model="editGroupColor" type="color" class="w-full h-10 rounded cursor-pointer" aria-label="选择颜色" />
        </div>
      </div>
      <template #footer>
        <div class="flex gap-3">
          <t-button theme="default" variant="outline" class="ui-t-button ui-t-button--secondary" @click="showEditGroup = false" aria-label="编辑">取消</t-button>
          <t-button class="ui-t-button ui-t-button--primary" theme="primary" @click="saveGroup">保存</t-button>
        </div>
      </template>
    </Modal>

    <!-- 验证码弹窗 -->
    <Modal v-model="showAuth" title="重新登录">
      <div class="space-y-4 text-center">
        <p v-if="oldAuthStarting" class="text-text-secondary">正在连接 Telegram 并申请验证码，请稍候…</p>
        <p v-else-if="need2FA" class="text-text-secondary">请输入 Telegram 两步验证密码</p>
        <p v-else class="text-text-secondary">验证码发送目标：{{ authAccount?.phone }}</p>
        <input
          v-if="!need2FA"
          v-model="oldAuthCode"
          :disabled="oldAuthStarting || oldAuthSubmitting"
          placeholder="请输入验证码"
          inputmode="numeric"
          pattern="[0-9]*"
          class="ui-input !min-h-[48px] !px-4 !py-3 !text-center !text-lg font-mono tracking-widest"
          maxlength="6" aria-label="请输入验证码" />
        <div v-if="need2FA" class="space-y-2">
          <input
            v-model="oldTwoFAPassword"
            :disabled="oldAuthSubmitting"
            type="password"
            placeholder="两步验证密码"
            class="ui-input !min-h-[48px] !px-4 !py-3 !text-center" aria-label="两步验证密码" />
        </div>
      </div>
      <template #footer>
        <t-button theme="default" variant="outline" :disabled="oldAuthStarting || oldAuthSubmitting" class="ui-t-button ui-t-button--secondary" @click="showAuth = false">取消</t-button>
        <t-button class="ui-t-button ui-t-button--primary" theme="primary" :disabled="oldAuthStarting || oldAuthSubmitting || (!need2FA && !oldAuthCode.trim()) || (need2FA && !oldTwoFAPassword)" @click="need2FA ? submitOld2FA() : submitOldCode()">
          {{ oldAuthStarting ? '正在启动…' : oldAuthSubmitting ? '正在验证…' : '确认' }}
        </t-button>
      </template>
    </Modal>

    <!-- 编辑备注弹窗(含代理设置) -->
    <Modal v-model="showEditRemark" title="编辑账号">
      <div class="space-y-3">
        <div>
          <label class="block text-sm text-text-secondary mb-1.5">备注</label>
          <input
            id="edit-remark" v-model="editRemarkText"
            placeholder="输入备注..."
            class="ui-input !min-h-[44px] !text-sm" aria-label="输入备注..." />
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-1.5">代理（SOCKS5，留空 = 直连）</label>
          <input v-model="editProxyText" placeholder="socks5://账号:密码@IP:端口"
            class="ui-input !min-h-[44px] !text-sm font-mono" aria-label="代理地址" />
          <div class="flex items-center gap-2 mt-2">
            <t-button class="ui-t-button ui-t-button--secondary ui-t-button--sm" size="small" variant="outline" :disabled="proxyTesting" @click="doTestProxy">
              <Loader2 v-if="proxyTesting" class="w-3.5 h-3.5 animate-spin mr-1" />
              测试连通
            </t-button>
            <span v-if="proxyTestResult" class="text-xs" :class="proxyTestResult.ok ? 'text-[var(--t-ok)]' : 'text-[var(--t-err)]'">
              {{ proxyTestResult.ok ? `连通正常 · 延迟 ${proxyTestResult.latency_ms}ms` : (proxyTestResult.error || '连接失败') }}
            </span>
          </div>
        </div>
      </div>
      <template #footer>
        <t-button theme="default" variant="outline" class="ui-t-button ui-t-button--secondary" @click="showEditRemark = false">取消</t-button>
        <t-button class="ui-t-button ui-t-button--primary" theme="primary" @click="saveRemark">保存</t-button>
      </template>
    </Modal>

    <!-- R181 群转移: 选目标账号 + 可调随机间隔(分钟), 结果显示可转移数与跳过清单 -->
    <Modal v-model="showTransfer" :title="`转移群 — ${transferSource?.remark || transferSource?.phone || ''}`" width="440px">
      <div class="space-y-3">
        <div>
          <label class="block text-sm text-text-secondary mb-1.5">目标账号（把这些群转移到它）</label>
          <select v-model="transferTargetId" class="ui-input !min-h-[44px] !text-sm">
            <option value="" disabled>选择目标账号…</option>
            <option v-for="a in transferCandidates" :key="a.account_id" :value="a.account_id">
              {{ a.remark || a.phone }}<template v-if="a.status === 'offline'">（离线）</template>
            </option>
          </select>
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-1.5">加群随机间隔（分钟/群）</label>
          <div class="flex items-center gap-2">
            <input v-model.number="transferMin" type="number" min="1" max="60"
              class="ui-input !w-20 !min-h-[36px] !px-3 !py-2 !text-sm" aria-label="最小间隔" />
            <span class="text-text-secondary text-sm">~</span>
            <input v-model.number="transferMax" type="number" min="1" max="120"
              class="ui-input !w-20 !min-h-[36px] !px-3 !py-2 !text-sm" aria-label="最大间隔" />
          </div>
          <p class="text-xs text-text-secondary mt-1.5 leading-relaxed">
            每加一个群在区间内随机等待。默认 2~3 分钟；新注册/低权重号建议 5~8 分钟。
            公开群走 t.me 链接；私有群需要源号是管理员才能导出邀请链接，否则进下方跳过清单。
          </p>
        </div>
        <!-- 结果区: 创建任务后展示可转移数 + 跳过原因 -->
        <div v-if="transferResult" class="rounded-lg border border-border bg-bg-card/50 p-3 space-y-2">
          <div class="text-sm">
            <span class="text-[var(--t-ok)] font-medium">已创建转移任务：{{ transferResult.links_total }} 个群</span>
            <template v-if="transferResult.skipped?.length">
              <span class="text-text-secondary">；{{ transferResult.skipped.length }} 个群无法转移</span>
            </template>
          </div>
          <ul v-if="transferResult.skipped?.length" class="max-h-36 overflow-y-auto space-y-1 text-xs text-text-secondary">
            <li v-for="(s, i) in transferResult.skipped" :key="i" class="leading-relaxed">
              <span class="text-text-primary">{{ s.title }}</span> — {{ s.reason }}
            </li>
          </ul>
        </div>
      </div>
      <template #footer>
        <t-button theme="default" variant="outline" class="ui-t-button ui-t-button--secondary" @click="showTransfer = false">关闭</t-button>
        <t-button class="ui-t-button ui-t-button--primary" theme="primary" :disabled="transferRunning || !transferTargetId" @click="doTransfer">
          <Loader2 v-if="transferRunning" class="w-3.5 h-3.5 animate-spin mr-1" />
          {{ transferRunning ? '采集中(私群逐个导链接)…' : (transferResult ? '再次转移' : '开始转移') }}
        </t-button>
      </template>
    </Modal>

    <ConfirmDialog
      v-model="showDeleteConfirm"
      title="删除账号"
      :message="`确定删除账号 ${deletingAccount?.remark || deletingAccount?.phone || ''} 吗？`"
      detail="删除后该账号的所有任务记录也会消失，且无法恢复。"
      confirm-text="确认删除"
      danger
      @confirm="doDelete"
    />

    <!-- 账号检测/2FA改密 弹层已按租户要求移除(2026-10-20), 相关state与函数见下方注释 -->


    <!-- 分页控件(v1.19): 上一页/页码/下一页 -->
    <div v-if="accTotalPages > 1" class="flex items-center justify-center gap-2 mt-4">
      <t-button theme="default" variant="outline" size="sm" @click="accPage = Math.max(1, accPage - 1)" :disabled="accPage <= 1"
              class="ui-t-button ui-t-button--secondary ui-t-button--sm">上一页</t-button>
      <span class="text-xs text-text-muted tabular-nums">{{ accPage }} / {{ accTotalPages }}</span>
      <t-button theme="default" variant="outline" size="sm" @click="accPage = Math.min(accTotalPages, accPage + 1)" :disabled="accPage >= accTotalPages"
              class="ui-t-button ui-t-button--secondary ui-t-button--sm">下一页</t-button>
    </div>

    <!-- 批量改资料弹窗(批次2): 留空字段不变, 随机用户名为每号生成不同用户名 -->
    <Modal v-model="showBatchProfile" title="批量修改资料" width="480px">
      <div class="space-y-3">
        <p class="text-xs text-text-muted">对选中的 {{ selectedIds.size }} 个账号逐个应用(留空的字段不变)。改资料有 2-4 秒间隔防风控, 大批量请耐心等待。</p>
        <div class="grid grid-cols-2 gap-3">
          <t-input v-model="batchProfile.first_name" placeholder="名字(留空不变)" />
          <t-input v-model="batchProfile.last_name" placeholder="姓氏(留空不变)" />
        </div>
        <t-textarea v-model="batchProfile.about" placeholder="个人简介(留空不变)" :rows="2" />
        <div class="flex items-center gap-2">
          <t-input v-model="batchProfile.username" placeholder="用户名(留空不变)" class="flex-1" />
          <label class="flex items-center gap-1.5 text-xs text-text-secondary cursor-pointer shrink-0">
            <input type="checkbox" v-model="batchProfile.random_username" class="accent-[var(--t-brand)] w-4 h-4" />
            随机生成
          </label>
        </div>
        <t-input v-model="batchProfile.remark" placeholder="备注(留空不变)" />
        <!-- R119: 头像上传 —— 仅勾选单个账号时可用, 选图即传 -->
        <div class="flex items-center gap-3">
          <label class="flex-1 flex items-center gap-2 px-3 py-2.5 rounded-lg border border-dashed cursor-pointer transition-colors"
                 :class="selectedIds.size === 1 ? 'border-border hover:border-[color:var(--t-brand)] text-text-secondary' : 'border-border/50 text-text-muted cursor-not-allowed'">
            <svg class="w-4 h-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m17 8-5-5-5 5"/><path d="M12 3v12" stroke-linecap="round" stroke-linejoin="round"/></svg>
            <span class="text-xs truncate">{{ avatarFileName || (selectedIds.size === 1 ? '选择头像图片(JPG/PNG, 选图即上传)' : '头像上传需先勾选 1 个账号') }}</span>
            <input type="file" accept="image/jpeg,image/png" class="hidden" :disabled="selectedIds.size !== 1" @change="onAvatarPick" />
          </label>
          <span v-if="avatarUploading" class="text-xs text-[var(--t-brand)] shrink-0">上传中…</span>
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <t-button theme="default" variant="outline" class="ui-t-button ui-t-button--secondary" @click="showBatchProfile = false">取消</t-button>
          <t-button theme="primary" size="sm" @click="runBatchProfile" :disabled="batchingProfile"
                  class="ui-t-button ui-t-button--primary ui-t-button--sm disabled:opacity-50">
            {{ batchingProfile ? '执行中…' : '开始批量修改' }}
          </t-button>
        </div>
      </div>
    </Modal>

    <!-- 克隆他人资料(2026-10-21): 输入对方 @用户名 → 名/姓/签名套用到所有勾选号 -->
    <Modal v-model="showClone" title="克隆他人资料" width="420px">
      <div class="space-y-3">
        <div class="rounded-lg border border-border bg-bg-card/60 p-3 space-y-1">
          <div class="text-sm font-medium text-text-primary">复制 Telegram 公开资料</div>
          <p class="text-xs text-text-secondary leading-relaxed">从来源用户名读取名字、姓氏和个人简介，再应用到已选账号。用户名具有唯一性，头像不会在本次操作中覆盖。</p>
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-1.5">来源账号用户名</label>
          <input v-model="cloneSource" placeholder="@username"
                 class="ui-input !text-sm" aria-label="资料来源用户名"
                 autocomplete="off" autocapitalize="none" spellcheck="false"
                 @keyup.enter="runClone" />
          <p class="mt-1 text-[11px] text-text-muted">可输入带 @ 或不带 @ 的公开用户名。</p>
        </div>
        <div class="rounded-lg border border-border p-3 space-y-2">
          <div class="flex items-center justify-between text-xs">
            <span class="font-medium text-text-primary">目标账号</span>
            <span class="text-text-muted">已选 {{ cloneTargets.length }} 个</span>
          </div>
          <div v-if="cloneTargets.length" class="flex flex-wrap gap-1.5">
            <span v-for="(target, index) in cloneTargets.slice(0, 6)" :key="target?.account_id || index"
                  class="max-w-full truncate rounded-md px-2 py-1 text-[11px] bg-bg-hover text-text-secondary">
              {{ target?.remark || target?.first_name || target?.phone || '未命名账号' }}
            </span>
            <span v-if="cloneTargets.length > 6" class="text-[11px] text-text-muted self-center">另有 {{ cloneTargets.length - 6 }} 个</span>
          </div>
          <p v-else class="text-xs text-[var(--t-err)]">当前没有选中的目标账号，请关闭窗口后先多选账号。</p>
        </div>
        <div class="flex justify-end gap-2">
          <t-button theme="default" variant="outline" size="sm" @click="showClone = false"
                  class="ui-t-button ui-t-button--secondary ui-t-button--sm">取消</t-button>
          <t-button theme="primary" size="sm" @click="runClone"
                  :disabled="cloning || !cloneSource.trim() || cloneTargets.length === 0"
                  class="ui-t-button ui-t-button--primary ui-t-button--sm disabled:opacity-50">
            <Loader2 v-if="cloning" class="w-3.5 h-3.5 animate-spin" />
            {{ cloning ? '克隆中…' : '应用到所选账号' }}
          </t-button>
        </div>
      </div>
    </Modal>

    <!-- R119: 能力检测结果弹窗 —— 8 项矩阵 + 分级结论(截图同款) -->
    <Modal v-model="showProbe" :title="`能力检测 — ${probeAccount?.first_name || probeAccount?.phone || ''}`" width="420px">
      <div class="space-y-3">
        <!-- 冷却/未配置提示 -->
        <div v-if="probeCooldownText" class="flex items-start gap-2 rounded-lg border border-[color:var(--t-warn)]/30 bg-[color:var(--t-warn-dim)] p-3 text-xs text-[var(--t-warn)] leading-relaxed">
          <ShieldAlert class="w-4 h-4 shrink-0 mt-0.5" />{{ probeCooldownText }}
        </div>
        <!-- 结果 -->
        <template v-if="probeResult">
          <!-- 结论横幅 -->
          <div class="rounded-lg border px-3.5 py-2.5 text-sm font-medium"
               :class="probeResult.verdict.includes('正常') ? 'border-[color:var(--t-ok)]/30 bg-[var(--t-ok)]/10 text-[var(--t-ok)]' : 'border-[color:var(--t-err)]/30 bg-[var(--t-err)]/10 text-[var(--t-err)]'">
            {{ probeResult.verdict }}
          </div>
          <!-- 8 项逐项 -->
          <div class="divide-y divide-border/60 rounded-lg border border-border overflow-hidden">
            <div v-for="it in probeResult.items" :key="it.key" class="flex items-center justify-between px-3.5 py-2.5">
              <div class="min-w-0">
                <div class="text-sm" :class="it.ok ? 'text-text-primary' : 'text-text-secondary'">{{ it.label }}</div>
                <div v-if="!it.ok && it.detail" class="text-[11px] text-text-muted truncate" :title="it.detail">{{ it.detail }}</div>
              </div>
              <span class="shrink-0 ml-3" :class="it.ok ? 'text-[var(--t-ok)]' : 'text-[var(--t-err)]'">
                <svg v-if="it.ok" class="w-4.5 h-4.5 w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="m5 13 4 4L19 7" stroke-linecap="round" stroke-linejoin="round"/></svg>
                <svg v-else class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M18 6 6 18M6 6l12 12" stroke-linecap="round"/></svg>
              </span>
            </div>
          </div>
          <div class="text-[11px] text-text-muted text-right">检测于 {{ probeResult.checked_at?.slice(5, 16).replace('T', ' ') }}</div>
        </template>
        <div v-else-if="!probeCooldownText" class="py-6 text-center text-xs text-text-muted">
          尚未检测过 —— 点击下方按钮执行 8 项真实操作探测<br/><span class="text-[11px]">(私聊/加群/发言均真实执行后立即撤回, 探测目标由管理端统一配置)</span>
        </div>
        <!-- 操作行 -->
        <div class="flex items-center justify-between pt-1">
          <span class="text-[11px] text-text-muted">同一账号 24 小时内限测 2 次</span>
          <t-button theme="primary" size="sm" @click="runProbe" :disabled="probeRunning"
                  class="ui-t-button ui-t-button--primary ui-t-button--sm disabled:opacity-50">
            <Loader2 v-if="probeRunning" class="w-3.5 h-3.5 animate-spin" />
            {{ probeRunning ? '检测中…(约 30 秒)' : '开始检测' }}
          </t-button>
        </div>
      </div>
    </Modal>

  </main>
</template>

<script setup lang="ts">

import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { useAccountAvatar } from '@/composables/useAccountAvatar'
import {
  Plus, FolderPlus, User, Pencil, Trash2, Plug, Unplug,
  KeyRound, ShieldAlert, Shield, Loader2, Search,
  RefreshCw, Play, Archive, LogOut, MoreHorizontal,
  ShieldCheck, ArrowRightLeft, UserRound,
} from 'lucide-vue-next'
import { useAccountStore } from '@/stores/account'
import { accountApi, importSessions, importSessionFiles, setAccountProxy, testAccountProxy, uploadAvatar } from '@/api/account'
import { taskApi } from '@/api/task'
import { settingsApi } from '@/api/settings'
import Modal from '@/components/ui/Modal.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import PageHeader from '@/components/ui/PageHeader.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import { getHealthBg } from '@/utils/health'

// ===== 数字滚动(交互示例21): 统计卡数值从旧值缓动到新值 =====
// 同一元素快速收到新值时取消旧动画，避免旧 RAF 覆盖新目标；卸载时释放帧回调。
const countUpFrames = new WeakMap<HTMLElement, number>()
function runCountUp(el: HTMLElement, to: number) {
  const previousFrame = countUpFrames.get(el)
  if (previousFrame !== undefined) cancelAnimationFrame(previousFrame)
  const from = Number(el.textContent) || 0
  if (!Number.isFinite(to) || from === to) {
    el.textContent = String(Number.isFinite(to) ? to : 0)
    countUpFrames.delete(el)
    return
  }
  const start = performance.now(), dur = 600
  const tick = (now: number) => {
    const p = Math.min(1, (now - start) / dur)
    const eased = 1 - Math.pow(1 - p, 3)
    el.textContent = String(Math.round(from + (to - from) * eased))
    if (p < 1) {
      countUpFrames.set(el, requestAnimationFrame(tick))
    } else {
      countUpFrames.delete(el)
    }
  }
  countUpFrames.set(el, requestAnimationFrame(tick))
}
const vCountup = {
  mounted(el: HTMLElement, b: any) { runCountUp(el, Number(b.value) || 0) },
  updated(el: HTMLElement, b: any) { if (b.value !== b.oldValue) runCountUp(el, Number(b.value) || 0) },
  beforeUnmount(el: HTMLElement) {
    const frame = countUpFrames.get(el)
    if (frame !== undefined) cancelAnimationFrame(frame)
    countUpFrames.delete(el)
  },
}

// 模板"受限"按钮的倒计时文案。此前模板(137行)调用了不存在的 cooldownText（白屏级）；
// Account 接口也没声明 blocked_until（后端 markAccountRestricted 会写，此处防御读取）。
const nowTs = ref(Date.now())
const nowTick = setInterval(() => {
  if (!document.hidden) nowTs.value = Date.now()
}, 1000)
// 状态自动刷新(2026-10-21 轮询根治):
//   ① 间隔 15s→30s, 页面切后台(document.hidden)时暂停——手机切出去不再猛刷
//   ② 回前台立即补一次
//   ③ store 层已加 diff 守卫: 数据没变时数组引用不换, 列表不重渲染、动画不重播
const statusPoll = setInterval(() => {
  if (document.hidden) return
  accountStore.fetchAccounts(true)
}, 30000)
const onVisBack = () => { if (!document.hidden) accountStore.fetchAccounts(true) }
document.addEventListener('visibilitychange', onVisBack)
onUnmounted(() => {
  abortAuthController(addAuthController)
  addAuthController = null
  abortAuthController(reauthController)
  reauthController = null
  accountsPageActive = false
  if (avatarRefreshTimer) { clearTimeout(avatarRefreshTimer); avatarRefreshTimer = null }
  if (reconnectRefreshTimer) { clearTimeout(reconnectRefreshTimer); reconnectRefreshTimer = null }
  if (pressTimer) { clearTimeout(pressTimer); pressTimer = null }
  statsRequestId++
  statsController?.abort()
  statsController = null
  taskSummaryRequestId++
  taskSummaryController?.abort()
  taskSummaryController = null
  clearInterval(nowTick)
  clearInterval(statusPoll)
  document.removeEventListener('visibilitychange', onVisBack)
})
function cooldownText(acc: Account): string {
  const until = acc.blocked_until
  if (!until) return ''
  const untilTs = new Date(until).getTime()
  if (!Number.isFinite(untilTs)) return ''
  const ms = untilTs - nowTs.value
  if (ms <= 0) return ''
  const seconds = Math.ceil(ms / 1000)
  if (seconds < 60) return `${seconds}秒`
  const minutes = Math.ceil(ms / 60000)
  return minutes >= 60 ? `${Math.floor(minutes / 60)}小时${minutes % 60 ? (minutes % 60) + '分' : ''}` : `${minutes}分钟`
}
import { toast } from '@/utils/toast'
import type { Account, AccountGroup } from '@/types'
// R115: accountStatusClass 死导入已删(状态显示用自绘 statusDotClass)
import { accountStatusLabel } from '@/utils/format'

// R87-M8: 状态圆点配色(在线=绿点, 其余按状态色)
// R124: 入参判空兜底 —— 上游传 undefined 时不再让 switch 炸渲染
function statusDotClass(status?: string) {
  switch (status) {
    case 'online': return 'bg-[var(--t-ok)]'
    case 'offline': return 'bg-text-muted'
    case 'banned': return 'bg-[var(--t-err)]'
    case 'session_revoked': return 'bg-[var(--t-err)] animate-pulse'
    case 'restricted': return 'bg-[var(--t-warn)]'
    case 'pending_auth': return 'bg-[var(--t-warn)] animate-pulse'
    default: return 'bg-text-muted'
  }
}
import { ask } from '@/utils/confirm'

const accountStore = useAccountStore()
const { avatarOf, invalidate: invalidateAvatar } = useAccountAvatar()

// 2FA改密/账号检测/申诉 功能已按租户要求移除(2026-10-20): 改密走TG官方App, 检测为假数据, 申诉手动操作
const reconnectingId = ref('')
let avatarRefreshTimer: ReturnType<typeof setTimeout> | null = null
let reconnectRefreshTimer: ReturnType<typeof setTimeout> | null = null
const leavingAllId = ref('')
// R87-M4: 卡片次要操作收纳进"更多"菜单, 只留主操作(状态切换/资料)
const openMenuId = ref('')
// R119: 能力检测 —— 弹窗状态与执行
const showProbe = ref(false)
const probeAccount = ref<any>(null)
const probeResult = ref<{ items: any[]; verdict: string; checked_at: string } | null>(null)
const probeRunning = ref(false)
const probeCooldownText = ref('') // 冷却中提示(后端 429 返回)

let probeRequestId = 0
let probeController: AbortController | null = null
watch(showProbe, (visible) => {
  if (!visible) {
    probeController?.abort()
    probeController = null
    probeRequestId++
    probeRunning.value = false
  }
})
async function openProbe(acc: any) {
  probeController?.abort()
  const controller = new AbortController()
  probeController = controller
  const requestId = ++probeRequestId
  const accountId = acc?.account_id
  probeAccount.value = acc
  probeResult.value = null
  probeCooldownText.value = ''
  showProbe.value = true
  if (!accountId) return
  // 先展示上次结果(不触发检测)；切换账号或关闭弹窗后丢弃过期响应。
  try {
    const cached = await accountApi.getProbe(accountId, { signal: controller.signal })
    if (requestId === probeRequestId && showProbe.value && probeAccount.value?.account_id === accountId) {
      probeResult.value = cached || null
    }
  } catch { /* 无历史结果, 忽略 */ }
  finally { if (probeController === controller) probeController = null }
}

async function runProbe() {
  const accountId = probeAccount.value?.account_id
  if (!accountId || probeRunning.value) return
  const requestId = probeRequestId
  probeRunning.value = true
  probeCooldownText.value = ''
  try {
    const result = await accountApi.runProbe(accountId)
    if (requestId === probeRequestId && showProbe.value && probeAccount.value?.account_id === accountId) {
      probeResult.value = result
    }
  } catch (e: any) {
    // 冷却中: 429 带文案; 其他错误拦截器已 toast
    const msg = e?.response?.data?.message || e?.message || ''
    if (requestId === probeRequestId && showProbe.value && probeAccount.value?.account_id === accountId && msg.includes('冷却')) {
      probeCooldownText.value = msg
    }
  } finally {
    if (requestId === probeRequestId) probeRunning.value = false
  }
}
function toggleMenu(id: string) {
  openMenuId.value = openMenuId.value === id ? '' : id
}

// R87-M6: Esc 全局关闭浮层(更多菜单); 其他页面浮层走各自逻辑
// R141: Esc 同时退出多选模式(复选框列收起)
function onGlobalKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    openMenuId.value = ''
    if (selecting.value) exitSelecting()
  }
}
onMounted(() => document.addEventListener('keydown', onGlobalKeydown))
onUnmounted(() => document.removeEventListener('keydown', onGlobalKeydown))
const avatarRefreshingId = ref('')
const selectedIds = ref<Set<string>>(new Set())
// 账号被删除或租户账号列表刷新后，清理已不存在的 ID，避免批量操作继续提交失效账号。
// 只按真实账号全集校验，不按当前搜索/分页结果清理，保留跨筛选与跨页多选。
watch(
  () => (accountStore.accounts || []).map((account: any) => account?.account_id).filter(Boolean),
  (accountIds) => {
    const available = new Set(accountIds)
    const next = new Set([...selectedIds.value].filter(id => available.has(id)))
    if (next.size !== selectedIds.value.size) selectedIds.value = next
  }
)
const batchingPool = ref(false)
const selectAllRef = ref<HTMLInputElement | null>(null)

const showBatchProfile = ref(false)
const batchingProfile = ref(false)
const batchProfile = ref({ first_name: '', last_name: '', about: '', username: '', remark: '', random_username: false })

function openBatchProfile() {
  batchProfile.value = { first_name: '', last_name: '', about: '', username: '', remark: '', random_username: false }
  showBatchProfile.value = true
}

// 批量改资料执行: 提交后逐号 2-4 秒间隔, 结果汇总 toast
async function runBatchProfile() {
  if (batchingProfile.value) return
  const ids = [...selectedIds.value]
  if (!ids.length) {
    toast({ type: 'info', message: '请先选择要修改的账号' })
    return
  }
  // 固化本次提交的账号与资料，避免请求期间选择集合变化影响本次操作。
  const profile = { ...batchProfile.value }
  batchingProfile.value = true
  try {
    const r = await accountApi.batchUpdateProfile(ids, profile)
    if (r.failed === 0) {
      toast({ type: 'success', message: `${r.success} 个账号资料已修改` })
    } else {
      toast({ type: 'warning', message: `成功 ${r.success}, 失败 ${r.failed} 个` })
    }
    showBatchProfile.value = false
    selectedIds.value = new Set([...selectedIds.value].filter(id => !ids.includes(id)))
  } catch (e: any) {
    toast({ type: 'error', message: e?.message || '批量修改失败' })
  } finally {
    batchingProfile.value = false
  }
}
// 卡片"资料"按钮: 单号快速编辑——选中之并打开批量资料弹窗(空字段=不改)
// P2-24 审计修复: 原 openProfile 不重置 batchProfile——上一个号的资料(名/姓/签名/
// 备注)残留在表单里, 直接保存会把上个号的资料误写到这个号。
function openProfile(acc: any) {
  batchProfile.value = { first_name: '', last_name: '', about: '', username: '', remark: '', random_username: false }
  selectedIds.value = new Set([acc?.account_id])
  showBatchProfile.value = true
}

// R119: 头像上传 —— 单号资料弹窗内选图即传
const avatarFileName = ref('')
const avatarUploading = ref(false)
async function onAvatarPick(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || selectedIds.value.size !== 1) return
  const accId = [...selectedIds.value][0]
  avatarUploading.value = true
  avatarFileName.value = file.name
  try {
    await uploadAvatar(accId, file)
    toast({ type: 'success', message: '头像已上传, 稍等片刻刷新可见' })
  } catch { avatarFileName.value = '' /* 拦截器已提示 */ }
  finally {
    avatarUploading.value = false
    input.value = '' // 允许重复选同一文件
  }
}

// 克隆资料(2026-10-21): 输入对方 @用户名, 把名/姓/签名套用到所有勾选号
const showClone = ref(false)
const cloneSource = ref('')
const cloning = ref(false)
const cloneTargets = computed(() =>
  [...selectedIds.value]
    .map(id => (accountStore.accounts || []).find((a: any) => a?.account_id === id))
    .filter(Boolean)
)
function openClone() {
  if (!selectedIds.value.size) {
    toast({ type: 'info', message: '请先选择要应用资料的目标账号' })
    return
  }
  cloneSource.value = ''
  showClone.value = true
}
async function runClone() {
  if (cloning.value) return
  if (!cloneSource.value.trim()) {
    toast({ type: 'error', message: '请输入对方的 @用户名' })
    return
  }
  if (selectedIds.value.size === 0) {
    toast({ type: 'error', message: '请先勾选目标账号' })
    return
  }
  cloning.value = true
  try {
    const r = await accountApi.cloneProfile(cloneSource.value.trim(), [...selectedIds.value])
    if (r.failed === 0) {
      toast({ type: 'success', message: `已克隆 @${cloneSource.value.trim().replace(/^@/, '')} 的资料到 ${r.applied} 个账号` })
    } else {
      toast({ type: 'warning', message: `成功 ${r.applied} 个, 失败 ${r.failed} 个(不在线/被限制)` })
    }
    showClone.value = false
  } catch (e: any) {
    toast({ type: 'error', message: e?.message || '克隆失败' })
  } finally {
    cloning.value = false
  }
}
// R124 竞态守卫: 数组整体替换瞬间, computed 重算读到越界 undefined ——
// v-for 的 v-if 只管模板渲染, computed/every/map 的遍历要先炸, 所以这里也要 a &&。
const allSelected = computed(() =>
  filteredAccounts.value.length > 0 &&
  filteredAccounts.value.every((a: any) => a && selectedIds.value.has(a.account_id)))

function toggleSelectAll() {
  // 全选/取消全选只作用于当前筛选结果；保留其他筛选条件下已选中的账号，
  // 避免用户切换搜索词后执行“全选”意外清空此前选择。
  const visibleIds = filteredAccounts.value
    .map((a: any) => a?.account_id)
    .filter(Boolean)
  const next = new Set(selectedIds.value)
  if (allSelected.value) {
    visibleIds.forEach((id: string) => next.delete(id))
  } else {
    visibleIds.forEach((id: string) => next.add(id))
  }
  selectedIds.value = next
}

function toggleSelect(acc: any) {
  const s = new Set(selectedIds.value)
  if (s.has(acc?.account_id)) s.delete(acc?.account_id)
  else s.add(acc?.account_id)
  selectedIds.value = s
}

// ===== 多选模式(R141 新皮肤适配): 复选框默认隐藏不占位, 长按行(500ms)进入多选, 复选框列滑出 =====
const selecting = ref(false)
let pressTimer: ReturnType<typeof setTimeout> | null = null
let pressStartX = 0
let pressStartY = 0
function enterSelecting() {
  if (!selecting.value) selecting.value = true
}
function exitSelecting() {
  selecting.value = false
  selectedIds.value = new Set()
}
// R153: 登录验证码展示(10 分钟内有效)
function loginCodeFresh(acc: any): boolean {
  return !!acc?.last_login_code && !!acc?.login_code_at &&
    (Date.now() - new Date(acc.login_code_at).getTime()) < 10 * 60 * 1000
}
function codeAge(acc: any): string {
  const m = Math.floor((Date.now() - new Date(acc.login_code_at).getTime()) / 60000)
  return m < 1 ? '刚刚' : `${m}分前`
}
async function copyLoginCode(acc: any) {
  try { await navigator.clipboard.writeText(acc.last_login_code) } catch { }
}

function onRowTouchStart(e: TouchEvent, acc: any) {
  if (selecting.value || !acc?.account_id) return
  pressStartX = e.touches[0].clientX
  pressStartY = e.touches[0].clientY
  pressTimer = setTimeout(() => {
    enterSelecting()
    selectedIds.value = new Set([acc.account_id])
    if (navigator.vibrate) navigator.vibrate(30)
  }, 500)
}
function onRowTouchMove(e: TouchEvent) {
  if (!pressTimer) return
  const dx = Math.abs(e.touches[0].clientX - pressStartX)
  const dy = Math.abs(e.touches[0].clientY - pressStartY)
  if (dx > 10 || dy > 10) { clearTimeout(pressTimer); pressTimer = null }
}
function onRowTouchEnd() {
  if (pressTimer) { clearTimeout(pressTimer); pressTimer = null }
}
// 多选模式下点击行 = 勾选/取消(代替被隐藏的复选框热区)
function onRowClick(acc: any) {
  if (selecting.value) toggleSelect(acc)
}

// 数字紧凑化(R141): 1500→1.5k, 23000→2.3万(发送报告用, hover title 显示精确值)
function fmtCompact(n: number): string {
  if (!n) return '0'
  const trim = (v: number) => v.toFixed(1).replace(/\.0$/, '')
  if (n >= 100000000) return trim(n / 100000000) + '亿'
  if (n >= 10000) return trim(n / 10000) + '万'
  if (n >= 1000) return trim(n / 1000) + 'k'
  return String(n)
}

// 批量切备用/启用: 逐个执行(后端串行处理), 汇总结果反馈
async function batchSetPool(status: 'active' | 'standby') {
  if (!selectedIds.value.size || batchingPool.value) return
  const ids = [...selectedIds.value]
  const label = status === 'standby' ? '转备用' : '启用'
  batchingPool.value = true
  try {
    const r = await accountApi.batchSetPool(ids, status)
    if (r.failed === 0) {
      toast({ type: 'success', message: `${r.success} 个账号已${label}` })
    } else {
      toast({ type: 'warning', message: `${label}: 成功 ${r.success}, 失败 ${r.failed}(额度不足或状态异常)` })
    }
    // 部分失败时保留所选集合，方便用户查看失败提示后重试；全部成功才清空。
    if (r.failed === 0) selectedIds.value = new Set()
    accountStore.fetchAccounts(true)
  } catch (e: any) {
    toast({ type: 'error', message: e?.message || '批量操作失败' })
  } finally {
    batchingPool.value = false
  }
}

// 手动刷新头像: 清路径→重连→上线钩子自动重拉(强制覆盖旧头像)
async function refreshAvatar(acc: any) {
  if (avatarRefreshingId.value) return
  avatarRefreshingId.value = acc?.account_id
  try {
    await accountApi.refreshAvatar(acc?.account_id)
    if (!accountsPageActive) return
    toast({ type: 'success', message: '头像刷新中, 重连后自动更新' })
    if (avatarRefreshTimer) clearTimeout(avatarRefreshTimer)
    avatarRefreshTimer = setTimeout(() => {
      avatarRefreshTimer = null
      if (!accountsPageActive) return
      invalidateAvatar(acc?.account_id)
      void accountStore.fetchAccounts()
    }, 6000)
  } catch (e: any) {
    toast({ type: 'error', message: e?.message || '刷新失败' })
  } finally {
    if (accountsPageActive) avatarRefreshingId.value = ''
  }
}

// 一键退群: 确认后退出该账号加入的全部群组(逐个执行带风控间隔)
async function leaveAllGroups(acc: any) {
  const ok = await ask(`确定让 ${acc.remark || acc.phone} 退出全部已加入群组吗? 逐个执行需等待, 期间勿重复点击。`, { title: '一键退群', danger: true })
  if (!ok) return
  leavingAllId.value = acc?.account_id
  try {
    const r = await accountApi.leaveAllGroups(acc?.account_id)
    const parts = [`已退 ${r.left} 群`]
    if (r.skipped) parts.push(`${r.skipped} 个普通群需手动`)
    if (r.failed) parts.push(`${r.failed} 个失败`)
    toast({ type: r.failed ? 'warning' : 'success', message: parts.join(' · ') })
  } catch (e: any) {
    toast({ type: 'error', message: e?.message || '退群失败' })
  } finally {
    leavingAllId.value = ''
  }
}

// R181 群转移: 源号导出邀请链接 → 目标号创建 join_groups 任务(间隔租户可调)
const showTransfer = ref(false)
const transferSource = ref<any>(null)
const transferTargetId = ref('')
const transferMin = ref(2)
const transferMax = ref(3)
const transferRunning = ref(false)
const transferResult = ref<{ links_total: number; skipped: Array<{ title: string; reason: string }> } | null>(null)

// 候选目标: 排除源号与封禁号(封禁号任务必死, 服务端也会拦)
const transferCandidates = computed(() =>
  (accountStore.accounts || []).filter((a: any) =>
    a?.account_id && a.account_id !== transferSource.value?.account_id && a?.status !== 'banned'))

function openTransfer(acc: any) {
  transferSource.value = acc
  transferTargetId.value = ''
  transferResult.value = null
  showTransfer.value = true
}

async function doTransfer() {
  if (!transferSource.value?.account_id || !transferTargetId.value) return
  // 间隔合法性前端预检(服务端仍有强制兜底)
  let min = Number(transferMin.value) || 2
  let max = Number(transferMax.value) || min
  if (min < 1) min = 1
  if (max < min) max = min
  transferRunning.value = true
  try {
    const r = await accountApi.transferGroups(transferSource.value.account_id, transferTargetId.value, min, max)
    transferResult.value = { links_total: r.links_total, skipped: r.skipped || [] }
    if (r.links_total > 0) {
      toast({ type: 'success', message: `转移任务已创建：${r.links_total} 个群` })
      if (r.skipped?.length) toast({ type: 'warning', message: `${r.skipped.length} 个群无法转移(弹窗内看原因)` })
    } else {
      toast({ type: 'warning', message: r.message || '没有可转移的群' })
    }
  } catch (e: any) {
    toast({ type: 'error', message: e?.message || '转移失败' })
  } finally {
    transferRunning.value = false
  }
}

// 死号重连: 存 session 直连, 不用验证码
async function reconnectAccount(acc: any) {
  if (reconnectingId.value) return
  reconnectingId.value = acc?.account_id
  try {
    await accountApi.reconnect(acc?.account_id)
    if (!accountsPageActive) return
    toast({ type: 'success', message: '重连中, 稍候刷新看状态' })
    if (reconnectRefreshTimer) clearTimeout(reconnectRefreshTimer)
    reconnectRefreshTimer = setTimeout(() => {
      reconnectRefreshTimer = null
      if (!accountsPageActive) return
      void accountStore.fetchAccounts(true)
    }, 5000)
  } catch (e: any) {
    toast({ type: 'error', message: e?.message || '重连失败, 可能需重新登录' })
  } finally {
    if (accountsPageActive) reconnectingId.value = ''
  }
}

// 备用池切换: 转正占额度(后端校验余量), 转备用释放额度
async function togglePool(acc: any) {
  const target = acc.pool_status === 'standby' ? 'active' : 'standby'
  try {
    await accountApi.setPool(acc?.account_id, target)
    acc.pool_status = target
    toast({
      type: 'success',
      message: target === 'active' ? '已启用(占套餐额度)' : '已转备用(释放额度)',
    })
    accountStore.fetchAccounts(true)
  } catch (e: any) {
    toast({ type: 'error', message: e?.message || '操作失败' })
  }
}
const router = useRouter()

const loading = computed(() => accountStore.loading)

// 健康面板: 全局账号状态统计
// 普通/协议 两个世界(账号 store 原生支持): 统计和列表都跟着当前视图走
// 合并改造(2026-10-20): 普通号/协议号不再分 tab, 统一一个列表, 卡片上标类型徽标
const viewAccounts = computed<any[]>(() => accountStore.accounts || [])
// R110: viewLabel 已删 —— accountStore.view 是无赋值的死常量(永远 'phone'),
// 此标签永远显示"普通账号", 纯误导。协议号/正常号同属一个账号池(按 source 徽章区分)。

const statusCount = computed(() => {
  // R122-fix: 统一走清洗入口(与 typeCount 同源)
  const list = cleanAccounts.value
  const online = list.filter((a: any) => a.status === 'online').length
  const restricted = list.filter((a: any) => a.status === 'restricted').length
  const failed = list.filter((a: any) => a.status === 'failed' || a.status === 'banned').length
  return { online, restricted, failed, offline: list.length - online - restricted - failed }
})

const search = ref('')

const filteredAccounts = computed(() => {
  if (!search.value) return viewAccounts.value
  const s = search.value.toLowerCase()
  return viewAccounts.value.filter(a =>
    a && (  // R119: 防御 —— 列表混入 undefined 项时短路, 防 'reading status' 崩溃
      (a.phone || '').includes(s) ||
      (a.username || '').toLowerCase().includes(s) ||
      (a.remark && a.remark.toLowerCase().includes(s))
    )
  )
})

// 列表分页(此前编辑事故丢失定义, 模板 v-for 引用 undefined 导致列表空白)
const accPage = ref(1)
const ACC_PAGE_SIZE = 12
// R125 数据层守卫: 元素级过滤 —— 数组任何时刻混入 undefined/null 都在此清洗,
// 下游 42 处可选链防御兜底字段缺失。不用模板层守卫(两次构建事故的根源)。
const pagedAccounts = computed(() => {
  const start = (accPage.value - 1) * ACC_PAGE_SIZE
  return filteredAccounts.value.filter((a: any) => a && typeof a === 'object').slice(start, start + ACC_PAGE_SIZE)
})
const accTotalPages = computed(() => Math.max(1, Math.ceil(filteredAccounts.value.length / ACC_PAGE_SIZE)))
watch(filteredAccounts, () => { accPage.value = 1 })

// P2-23: 原实现每个账号渲染都 protocolAccounts.some() 扫全表——账号列表是
// O(n²)。换 Set 预计算, 单次渲染 O(n)。level==='protocol'  fast-path 保留。
// R124 竞态守卫: map 遍历与 v-for/computed 同病 —— 数组替换瞬间越界读到 undefined 时
// x.account_id 整页崩。可选链+filter 双保险, 无论真凶是哪条遍历路径全部封死。
const protocolIdSet = computed(() =>
  new Set((accountStore.protocolAccounts || []).map((x: any) => x?.account_id).filter(Boolean)))
const isProtocolAcc = (a: any) => a?.level === 'protocol' || protocolIdSet.value.has(a?.account_id)

// R122-fix: 统一清洗入口 —— 后端数组偶发混入 null/undefined, 任何列表遍历都走这里,
// 根治 "reading 'status'" 整页崩溃(此前逐个 computed 打补丁总有漏网路径)。
const cleanAccounts = computed(() =>
  (accountStore.accounts || []).filter((a: any) => a && typeof a === 'object')
)

const typeCount = computed(() => {
  const list = cleanAccounts.value
  return {
    phone: list.filter((a: any) => !isProtocolAcc(a)).length,
    protocol: list.filter((a: any) => isProtocolAcc(a)).length,
  }
})

const accountGroups = ref<AccountGroup[]>([])

function getGroupForAccount(accountId: string): AccountGroup | undefined {
  return accountGroups.value.find(g => g.account_ids?.includes(accountId))
}

// 添加账号（步骤化）
const showAddModal = ref(false)
let addAuthController: AbortController | null = null
let reauthController: AbortController | null = null

function abortAuthController(controller: AbortController | null) {
  controller?.abort()
}

function throwIfAuthAborted(signal: AbortSignal) {
  if (signal.aborted) {
    const error = new Error('Authentication polling cancelled')
    error.name = 'AbortError'
    throw error
  }
}

function authDelay(ms: number, signal: AbortSignal): Promise<void> {
  throwIfAuthAborted(signal)
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      signal.removeEventListener('abort', onAbort)
      resolve()
    }, ms)
    const onAbort = () => {
      clearTimeout(timer)
      signal.removeEventListener('abort', onAbort)
      const error = new Error('Authentication polling cancelled')
      error.name = 'AbortError'
      reject(error)
    }
    signal.addEventListener('abort', onAbort, { once: true })
  })
}

// 添加方式: phone=手机号验证码流程; protocol=协议号导入(原sessions页功能整合进来)
const addMode = ref<'phone' | 'protocol'>('phone')
const importText = ref('')
const importFileList = ref<File[]>([])
const importing = ref(false)

function onImportFiles(e: Event) {
  importFileList.value = Array.from((e.target as HTMLInputElement).files || [])
}

// 协议号导入: session 字符串与文件可同时提交, 结果合并提示
async function doProtocolImport() {
  if (importing.value) return
  importing.value = true
  try {
    const remark = newRemark.value.trim()
    const parts: string[] = []
    const sessions = importText.value.split('\n').map((s) => s.trim()).filter(Boolean)
    if (sessions.length > 0) {
      const r = await importSessions(sessions, remark)
      parts.push(`字符串导入: 成功 ${r.created} / 失败 ${r.failed}`)
      r.errors?.slice(0, 3).forEach((e) => parts.push(e))
    }
    if (importFileList.value.length > 0) {
      const r2 = await importSessionFiles(importFileList.value, remark)
      parts.push(`文件导入: 成功 ${r2.created} / 失败 ${r2.failed}`)
      r2.errors?.slice(0, 3).forEach((e) => parts.push(e))
    }
    toast({ type: 'success', message: parts.join('；') || '导入完成' })
    closeAddModal()
    accountStore.fetchAccounts(true)
  } catch (err: any) {
    toast({ type: 'error', message: err?.message || '导入失败' })
  } finally {
    importing.value = false
  }
}
const addStep = ref<'phone' | 'code' | '2fa'>('phone')
const newPhone = ref('')
const newRemark = ref('')
const newGroupId = ref('')

const groupOptions = computed(() => [
  { label: '不分组', value: '' },
  ...accountGroups.value.map(g => ({ label: g.name, value: g.group_id }))
])

const authCode = ref('')
const twoFAPassword = ref('')
const adding = ref(false)
const submitting = ref(false)
const codeError = ref('')
const twoFAError = ref('')
const newAccountId = ref('')

watch(showAddModal, (visible) => {
  if (!visible) {
    abortAuthController(addAuthController)
    addAuthController = null
    adding.value = false
    submitting.value = false
  }
})
const addModalTitle = computed(() => {
  if (addStep.value === 'phone') return '添加 Telegram 账号'
  if (addStep.value === 'code') return '输入验证码'
  return '两步验证'
})

function openAddModal() {
  showAddModal.value = true
  addStep.value = 'phone'
  newPhone.value = ''
  newRemark.value = ''
  newGroupId.value = ''
  authCode.value = ''
  twoFAPassword.value = ''
  codeError.value = ''
  twoFAError.value = ''
  newAccountId.value = ''
}
function closeAddModal() {
  abortAuthController(addAuthController)
  addAuthController = null
  showAddModal.value = false
  addStep.value = 'phone'
}

// 轮询 TG 认证阶段。connect 接口只代表"认证流程已启动"，
// 必须等后端 phase 真正进入 waiting_code(TG 已要求输入=码真的发了)
// 才能显示"已发送"，否则 TG 风控/失败时用户会看到假"已发送"并永远卡住。
async function waitForAuthPhase(accountId: string, timeoutMs: number, signal: AbortSignal): Promise<{ phase: string }> {
  const start = Date.now()
  const interval = 2000
  while (Date.now() - start < timeoutMs) {
    await authDelay(interval, signal)
    throwIfAuthAborted(signal)
    try {
      const state = await accountApi.getAuthState(accountId, { signal })
      throwIfAuthAborted(signal)
      const phase = (state as { phase?: string }).phase || ''
      // 初始空串和 connecting 都是中间态，继续等
      if (phase && phase !== 'connecting') return { phase }
    } catch (error) {
      if (signal.aborted) throw error
      // 轮询失败(如客户端刚被后端移除)按 none 处理，由调用方判失败
      return { phase: 'none' }
    }
  }
  return { phase: 'timeout' }
}

async function getVerificationCode() {
  addAuthController?.abort()
  const controller = new AbortController()
  addAuthController = controller
  const signal = controller.signal
  adding.value = true
  toast({ type: 'info', title: '正在连接 Telegram 服务器', message: '可能需要 30-60 秒，请耐心等待...' })
  try {
    // P0-11 审计修复: 先剥掉用户可能输入的前导 + 再统一加, 防 ++8613 拼错
    const digits = newPhone.value.replace(/[\s+]/g, '')
    const phone = '+' + digits
    const acc = await accountStore.createAccount(phone, newRemark.value || undefined)
    if (!acc?.account_id) throw new Error('账号创建失败，请重试')
    if (!signal.aborted) newAccountId.value = acc.account_id
    // 创建请求一旦发出，仍需完成服务端连接启动，避免关闭弹窗留下未启动的账号。
    await accountApi.connect(acc.account_id)
    throwIfAuthAborted(signal)
    // connect 仅启动异步认证；轮询确认 TG 真正进入等码阶段，防止"假已发送"
    const result = await waitForAuthPhase(acc?.account_id, 75000, signal)
    if (result.phase === 'waiting_code') {
      addStep.value = 'code'
      toast({ type: 'success', title: '验证码已发送', message: '请查收短信或 Telegram 消息' })
    } else if (result.phase === 'waiting_password') {
      addStep.value = '2fa'
    } else if (result.phase === 'success' || result.phase === 'connected' || result.phase === 'online') {
      showAddModal.value = false
      await accountStore.fetchAccounts(true)
      toast({ type: 'success', title: '账号已是登录状态', message: '会话有效，无需验证码' })
    } else {
      throw new Error(result.phase === 'timeout'
        ? '等待验证码超时：TG 未响应，请稍后重试'
        : 'TG 未发送验证码（号码可能被风控或 API 受限），请稍后重试')
    }
  } catch (e: unknown) {
    if (!signal.aborted && !(e instanceof Error && e.name === 'AbortError')) toast({ type: 'error', title: '发送失败', message: (e instanceof Error ? e.message : '请检查手机号格式') })
  } finally {
    if (addAuthController === controller && !signal.aborted) adding.value = false
  }
}
async function resendCode() {
  if (!newAccountId.value) return
  const controller = addAuthController ?? new AbortController()
  addAuthController = controller
  const signal = controller.signal
  try {
    try {
      await accountApi.connect(newAccountId.value)
    } catch (e: unknown) {
      // 已有认证流程在跑（连点等）不必重开，直接复用轮询即可
      const msg = e instanceof Error ? e.message : ''
      if (!msg.includes('already in progress')) throw e
    }
    throwIfAuthAborted(signal)
    const result = await waitForAuthPhase(newAccountId.value, 75000, signal)
    if (result.phase === 'waiting_code') {
      toast({ type: 'success', title: '验证码已重新发送', message: '请查收短信或 Telegram 消息' })
    } else if (result.phase === 'waiting_password') {
      addStep.value = '2fa'
    } else {
      toast({ type: 'error', title: '重新发送失败', message: 'TG 未响应，请稍后重试' })
    }
  } catch (e: unknown) {
    if (!signal.aborted && !(e instanceof Error && e.name === 'AbortError')) toast({ type: 'error', title: '重新发送失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  }
}

// 登录成功的最终判据：账号状态变 online。
// phase 可能因时序（认证成功→客户端清理→phase 变 none）或超时窗口太短而不可靠，
// 以账号状态为准，避免"实际已登录但弹窗卡住、需手动刷新"的假失败。
async function isAccountOnline(accountId: string, signal: AbortSignal): Promise<boolean> {
  throwIfAuthAborted(signal)
  try {
    await accountStore.fetchAccounts(true, { signal })
    throwIfAuthAborted(signal)
    const acc = (accountStore.phoneAccounts as any[]).find(a => a?.account_id === accountId)
    return !!acc && acc.status === 'online'
  } catch { return false }
}

async function submitVerificationCode() {
  if (!newAccountId.value || !authCode.value) return
  const controller = addAuthController ?? new AbortController()
  addAuthController = controller
  const signal = controller.signal
  submitting.value = true
  codeError.value = ''
  try {
    await accountApi.submitCode(newAccountId.value, authCode.value)
    throwIfAuthAborted(signal)
    let attempts = 0
    let codeRetrySeenAt: number | null = null
    const maxAttempts = 90 // R158: 90秒 —— 弱网+DC切换+密钥保存可能超60s
    while (attempts < maxAttempts) {
      await authDelay(1000, signal)
      attempts++ // 先计数: 宽限分支的 continue 依赖准确计数, 放尾部会被 continue 跳过
      // ① 账号已上线即成功（ground truth，不依赖 phase 时序）
      if (await isAccountOnline(newAccountId.value, signal)) {
        showAddModal.value = false
        addStep.value = 'phone'
        newPhone.value = ''
        toast({ type: 'success', title: '登录成功' })
        submitting.value = false
        return
      }
      const state = await accountApi.getAuthState(newAccountId.value, { signal })
      throwIfAuthAborted(signal)
      if (state.phase === 'waiting_password') {
        // 需要 2FA 密码
        twoFAPassword.value = ''
        addStep.value = '2fa'
        submitting.value = false
        return
      }
      if (state.phase === 'success' || state.phase === 'connected' || state.phase === 'online') {
        showAddModal.value = false
        addStep.value = 'phone'
        newPhone.value = ''
        toast({ type: 'success', title: '登录成功' })
        submitting.value = false
        return
      }
      // 验证码重试信号(code_retries>0): 可能是真错, 也可能是 gotd 瞬断重跑
      // 认证流程的假信号(后端已在新码提交时重置计数, 但瞬断场景仍可能误判)。
      // 策略: 先软提示+宽限继续轮询, 宽限期内账号上线即成功; 超时未上线才判真错。
      if (state.phase === 'waiting_code' && (state.code_retries ?? 0) > 0) {
        if (codeRetrySeenAt === null) {
          codeRetrySeenAt = attempts
          submitting.value = false
          codeError.value = '正在校验，请稍候…'
        }
        if (attempts - codeRetrySeenAt < 10) continue
        submitting.value = false
        codeError.value = '验证码错误，请重新输入'
        authCode.value = ''
        return
      }
      // phase=none/failed 不立刻判死：认证成功后客户端会被清理导致 phase 不可读，
      // 继续等下一轮账号状态检查
    }
    // 超时前最后核实一次账号状态（可能刚上线而轮询已尽）
    if (await isAccountOnline(newAccountId.value, signal)) {
      showAddModal.value = false
      addStep.value = 'phone'
      newPhone.value = ''
      toast({ type: 'success', title: '登录成功' })
      submitting.value = false
      return
    }
    codeError.value = '认证超时，请到账号列表确认状态'
  } catch (e: unknown) {
    if (!signal.aborted && !(e instanceof Error && e.name === 'AbortError')) codeError.value = (e instanceof Error ? e.message : '验证码错误')
  } finally {
    if (!signal.aborted) submitting.value = false
  }
}

async function submit2FA() {
  if (!newAccountId.value || !twoFAPassword.value) return
  const controller = addAuthController ?? new AbortController()
  addAuthController = controller
  const signal = controller.signal
  submitting.value = true
  twoFAError.value = ''
  try {
    await accountApi.submit2FA(newAccountId.value, twoFAPassword.value)
    throwIfAuthAborted(signal)
    let attempts = 0
    let pwdRetrySeenAt: number | null = null
    const maxAttempts = 60
    while (attempts < maxAttempts) {
      await authDelay(1000, signal)
      attempts++ // 先计数(同验证码步)
      // ① 账号已上线即成功（ground truth，同验证码逻辑）
      if (await isAccountOnline(newAccountId.value, signal)) {
        showAddModal.value = false
        addStep.value = 'phone'
        newPhone.value = ''
        toast({ type: 'success', title: '登录成功' })
        submitting.value = false
        return
      }
      const state = await accountApi.getAuthState(newAccountId.value, { signal })
      throwIfAuthAborted(signal)
      if (state.phase === 'success' || state.phase === 'connected' || state.phase === 'online') {
        showAddModal.value = false
        addStep.value = 'phone'
        newPhone.value = ''
        toast({ type: 'success', title: '登录成功' })
        submitting.value = false
        return
      }
      // 2FA 重试信号(pwd_retries>0)——用户报"2FA 输对后卡死在输入界面"的元凶:
      // 上次失败遗留的脏计数(后端已在新密码提交时重置, 见 SubmitPassword)或
      // gotd 瞬断假信号, 会在正确密码核验途中被这里秒判+return 终止轮询,
      // 账号 1-2 秒后上线却无人再查。软提示+宽限轮询, 宽限期内上线即成功。
      if (state.phase === 'waiting_password' && (state.pwd_retries ?? 0) > 0) {
        if (pwdRetrySeenAt === null) {
          pwdRetrySeenAt = attempts
          submitting.value = false
          twoFAError.value = '正在校验，请稍候…'
        }
        if (attempts - pwdRetrySeenAt < 10) continue
        submitting.value = false
        twoFAError.value = '两步验证密码错误，请重新输入'
        twoFAPassword.value = ''
        return
      }
      // phase=none/failed 不立刻判死，等下一轮账号状态核实
    }
    if (await isAccountOnline(newAccountId.value, signal)) {
      showAddModal.value = false
      addStep.value = 'phone'
      newPhone.value = ''
      toast({ type: 'success', title: '登录成功' })
      submitting.value = false
      return
    }
    twoFAError.value = '认证超时，请到账号列表确认状态'
  } catch (e: unknown) {
    if (!signal.aborted && !(e instanceof Error && e.name === 'AbortError')) twoFAError.value = (e instanceof Error ? e.message : '密码错误')
  } finally {
    if (!signal.aborted) submitting.value = false
  }
}

// 连接/断开
const connecting = ref('')
const disconnecting = ref('')

async function connect(acc: Account) {
  connecting.value = acc?.account_id
  try {
    await accountApi.connect(acc?.account_id)
    await accountStore.fetchAccounts()
    // 连接成功后，如果当前没有选中账号，自动选这个
    if (!accountStore.currentAccount) {
      accountStore.currentAccount = accountStore.phoneAccounts.find(a => a?.account_id === acc?.account_id) || null
    }
  } catch (e: unknown) {
    toast({ type: 'error', title: '连接失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    connecting.value = ''
  }
}

async function disconnect(acc: Account) {
  disconnecting.value = acc?.account_id
  try {
    await accountApi.disconnect(acc?.account_id)
    await accountStore.fetchAccounts()
    // 如果断开的是当前选中账号，重置选中
    if (accountStore.currentAccount?.account_id === acc?.account_id) {
      accountStore.currentAccount = null
    }
  } catch (e: unknown) {
    toast({ type: 'error', title: '断开失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    disconnecting.value = ''
  }
}

// 重新登录：先由后端启动全新的手机号认证流程，再允许输入验证码/2FA。
const showAuth = ref(false)
const authAccount = ref<Account | null>(null)
const oldAuthCode = ref('')
const need2FA = ref(false)
const oldTwoFAPassword = ref('')
const oldAuthStarting = ref(false)
const oldAuthSubmitting = ref(false)
watch(showAuth, (visible) => {
  if (!visible) {
    abortAuthController(reauthController)
    reauthController = null
    oldAuthStarting.value = false
    oldAuthSubmitting.value = false
  }
})

async function waitOldAuthReady(accountId: string, signal: AbortSignal): Promise<'code' | 'password' | 'success' | 'timeout'> {
  const deadline = Date.now() + 45000
  while (Date.now() < deadline) {
    try {
      const state = await accountApi.getAuthState(accountId, { signal })
      throwIfAuthAborted(signal)
      if (state.phase === 'waiting_code') return 'code'
      if (state.phase === 'waiting_password') return 'password'
      if (state.phase === 'connected' || state.phase === 'success' || state.phase === 'online') return 'success'
    } catch (error) {
      if (signal.aborted) throw error
      // 刚启动时状态接口可能短暂尚未可读，继续等待。
    }
    try {
      await accountStore.fetchAccounts(true, { signal })
      throwIfAuthAborted(signal)
      const latest = (accountStore.accounts || []).find((a: any) => a?.account_id === accountId)
      if (latest?.status === 'online') return 'success'
    } catch (error) {
      if (signal.aborted) throw error
      // 列表刷新失败不应中断认证状态轮询。
    }
    await authDelay(1000, signal)
  }
  return 'timeout'
}

async function showAuthModal(acc: Account) {
  if (oldAuthStarting.value) return
  reauthController?.abort()
  const controller = new AbortController()
  reauthController = controller
  const signal = controller.signal
  authAccount.value = acc
  showAuth.value = true
  oldAuthCode.value = ''
  oldTwoFAPassword.value = ''
  need2FA.value = false
  oldAuthStarting.value = true
  try {
    await accountApi.reauthenticate(acc.account_id)
    throwIfAuthAborted(signal)
    const ready = await waitOldAuthReady(acc.account_id, signal)
    throwIfAuthAborted(signal)
    if (ready === 'code') {
      toast({ type: 'success', title: '验证码已发送', message: `请查看 Telegram 并输入发送到 ${acc.phone} 的验证码` })
    } else if (ready === 'password') {
      need2FA.value = true
      toast({ type: 'info', title: '需要二级验证', message: '请输入 Telegram 两步验证密码' })
    } else if (ready === 'success') {
      showAuth.value = false
      await accountStore.fetchAccounts(true)
      toast({ type: 'success', title: '账号已重新登录' })
    } else {
      toast({ type: 'warning', title: '登录流程启动超时', message: '弹窗保持打开，请稍后重试或检查服务端日志' })
    }
  } catch (e: unknown) {
    if (!signal.aborted && !(e instanceof Error && e.name === 'AbortError')) toast({ type: 'error', title: '无法启动重新登录', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    if (!signal.aborted) oldAuthStarting.value = false
  }
}

// 提交凭据后必须等待 Telegram 最终认证状态；短暂接口错误不应误判为认证失败。
async function waitOldAuthResult(accountId: string, expected: 'code' | 'password', signal: AbortSignal): Promise<'success' | 'password' | 'code' | 'timeout'> {
  const deadline = Date.now() + 90000
  while (Date.now() < deadline) {
    try {
      await accountStore.fetchAccounts(true, { signal })
      throwIfAuthAborted(signal)
    } catch (error) {
      if (signal.aborted) throw error
      // 轮询刷新失败时继续查询认证状态。
    }
    try {
      const state = await accountApi.getAuthState(accountId, { signal })
      throwIfAuthAborted(signal)
      const latest = (accountStore.accounts || []).find((a: any) => a?.account_id === accountId)
      if (latest?.status === 'online' || state.phase === 'connected' || state.phase === 'success' || state.phase === 'online') return 'success'
      if (state.phase === 'waiting_password') {
        if (expected === 'code' || (state.pwd_retries ?? 0) > 0) return 'password'
      }
      if (state.phase === 'waiting_code' && expected === 'code' && (state.code_retries ?? 0) > 0) return 'code'
    } catch (error) {
      if (signal.aborted) throw error
      // 状态接口短暂失败时保留弹窗并继续等待。
    }
    await authDelay(1000, signal)
  }
  return 'timeout'
}

async function submitOldCode() {
  if (!authAccount.value || !oldAuthCode.value.trim() || oldAuthSubmitting.value) return
  const accountId = authAccount.value.account_id
  const controller = reauthController ?? new AbortController()
  reauthController = controller
  const signal = controller.signal
  oldAuthSubmitting.value = true
  try {
    await accountApi.submitCode(accountId, oldAuthCode.value.trim())
    throwIfAuthAborted(signal)
    const result = await waitOldAuthResult(accountId, 'code', signal)
    if (result === 'success') {
      showAuth.value = false
      await accountStore.fetchAccounts(true)
      toast({ type: 'success', title: '重新登录成功' })
    } else if (result === 'password') {
      need2FA.value = true
      oldTwoFAPassword.value = ''
      toast({ type: 'info', title: '需要二级验证', message: '请输入 Telegram 两步验证密码' })
    } else if (result === 'code') {
      oldAuthCode.value = ''
      toast({ type: 'error', title: '验证码未通过', message: '请检查验证码后重新输入' })
    } else {
      toast({ type: 'warning', title: '仍在验证中', message: '弹窗保持打开，请稍后查看账号状态' })
    }
  } catch (e: unknown) {
    if (!signal.aborted && !(e instanceof Error && e.name === 'AbortError')) toast({ type: 'error', title: '验证码提交失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    if (!signal.aborted) oldAuthSubmitting.value = false
  }
}

async function submitOld2FA() {
  if (!authAccount.value || !oldTwoFAPassword.value || oldAuthSubmitting.value) return
  const accountId = authAccount.value.account_id
  const controller = reauthController ?? new AbortController()
  reauthController = controller
  const signal = controller.signal
  oldAuthSubmitting.value = true
  try {
    await accountApi.submit2FA(accountId, oldTwoFAPassword.value)
    throwIfAuthAborted(signal)
    const result = await waitOldAuthResult(accountId, 'password', signal)
    if (result === 'success') {
      showAuth.value = false
      await accountStore.fetchAccounts(true)
      toast({ type: 'success', title: '重新登录成功' })
    } else if (result === 'password') {
      oldTwoFAPassword.value = ''
      toast({ type: 'error', title: '二级验证未通过', message: '请检查两步验证密码后重新输入' })
    } else {
      toast({ type: 'warning', title: '仍在验证中', message: '弹窗保持打开，请稍后确认账号状态' })
    }
  } catch (e: unknown) {
    if (!signal.aborted && !(e instanceof Error && e.name === 'AbortError')) toast({ type: 'error', title: '二级验证提交失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  } finally {
    if (!signal.aborted) oldAuthSubmitting.value = false
  }
}

// 编辑备注 + 代理
const showEditRemark = ref(false)
const editRemarkText = ref('')
const editProxyText = ref('')
const proxyTesting = ref(false)
const proxyTestResult = ref<{ ok: boolean; latency_ms?: number; error?: string } | null>(null)
const editingAccount = ref<Account | null>(null)

function editRemark(acc: Account) {
  // R129 竞态守卫: 数组替换瞬间 acc 可能 undefined(chunk 定位: 读 .remark 崩)
  if (!acc) return
  editingAccount.value = acc
  editRemarkText.value = acc.remark || ''
  editProxyText.value = (acc as any).proxy_url || ''
  proxyTestResult.value = null
  showEditRemark.value = true
}

// 测试连通: 测的是已保存的代理; 若输入框改过先提示保存
async function doTestProxy() {
  if (!editingAccount.value) return
  if (editProxyText.value.trim() !== ((editingAccount.value as any).proxy_url || '')) {
    proxyTestResult.value = { ok: false, error: '先保存再测试（或先还原输入）' }
    return
  }
  proxyTesting.value = true
  proxyTestResult.value = null
  try {
    proxyTestResult.value = await testAccountProxy(editingAccount.value.account_id)
  } catch (e: unknown) {
    proxyTestResult.value = { ok: false, error: (e instanceof Error ? e.message : '测试失败') }
  } finally {
    proxyTesting.value = false
  }
}

async function saveRemark() {
  if (!editingAccount.value) return
  // P2-25 审计修复: 原两步一个 try——第一步(备注)成功第二步(代理)失败时报
  // "更新失败", 但备注实际已入库, 用户认知与实际状态不一致。拆开处理:
  // 备注失败全停; 代理失败单独告警但保留备注成果。
  try {
    await accountApi.update(editingAccount.value.account_id, { remark: editRemarkText.value })
  } catch (e: unknown) {
    toast({ type: 'error', title: '备注保存失败', message: (e instanceof Error ? e.message : '请稍后重试') })
    return
  }
  // 代理单独走专用接口(带格式校验+断线重连)
  try {
    await setAccountProxy(editingAccount.value.account_id, editProxyText.value.trim())
  } catch (e: unknown) {
    toast({ type: 'warning', title: '备注已保存, 代理设置失败', message: (e instanceof Error ? e.message : '请检查代理格式后重试') })
  }
  showEditRemark.value = false
  await accountStore.fetchAccounts()
  toast({ type: 'success', title: '已保存' })
}

// 删除
const showDeleteConfirm = ref(false)
const deletingAccount = ref<Account | null>(null)

function confirmDelete(acc: Account) {
  deletingAccount.value = acc
  showDeleteConfirm.value = true
}

async function doDelete() {
  if (!deletingAccount.value) return
  try {
    await accountStore.deleteAccount(deletingAccount.value.account_id)
    await accountStore.fetchAccounts()
    showDeleteConfirm.value = false
    toast({ type: 'success', title: '账号已删除' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '删除失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  }
}

// 分组管理
const showGroupModal = ref(false)
const showEditGroup = ref(false)
const newGroupName = ref('')
const newGroupColor = ref('text-[var(--t-brand)]')
const editGroupId = ref('')
const editGroupName = ref('')
const editGroupColor = ref('text-[var(--t-brand)]')
// 编辑分组时保留原成员列表：后端 Update 会无条件覆盖 account_ids，
// 传空数组会把组内账号全部清空
const editGroupAccountIds = ref<string[]>([])

async function fetchGroups() {
  try {
    const res = await settingsApi.getAccountGroups()
    accountGroups.value = res.list || []
  } catch (e: unknown) {
    toast({ type: 'error', title: '获取分组失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  }
}

async function createGroup() {
  if (!newGroupName.value) return
  try {
    await settingsApi.createAccountGroup({
      name: newGroupName.value,
      color: newGroupColor.value,
      account_ids: []
    })
    newGroupName.value = ''
    await fetchGroups()
    toast({ type: 'success', title: '分组已创建' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '创建失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  }
}

function editGroup(g: AccountGroup) {
  editGroupId.value = g.group_id
  editGroupName.value = g.name
  editGroupColor.value = g.color || 'text-[var(--t-brand)]'
  editGroupAccountIds.value = [...(g.account_ids || [])]
  showEditGroup.value = true
}

async function saveGroup() {
  try {
    await settingsApi.updateAccountGroup(editGroupId.value, {
      name: editGroupName.value,
      color: editGroupColor.value,
      // 必须带回原成员列表: 后端 Update 无条件覆盖 account_ids,
      // 传空数组会把组内账号全部清空 (BUG#4 修复)
      account_ids: editGroupAccountIds.value
    })
    showEditGroup.value = false
    await fetchGroups()
    toast({ type: 'success', title: '分组已更新' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '更新失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  }
}

async function deleteGroup(g: AccountGroup) {
  if (!await ask(`确定删除分组「${g.name}」吗？`)) return
  try {
    await settingsApi.deleteAccountGroup(g.group_id)
    await fetchGroups()
    toast({ type: 'success', title: '分组已删除' })
  } catch (e: unknown) {
    toast({ type: 'error', title: '删除失败', message: (e instanceof Error ? e.message : '请稍后重试') })
  }
}


// 精读清理: safeUrl 已无人调用(头像统一走 useAccountAvatar blob 缓存), 删除防误用

// ===== 每账号累计记分牌: 成功/失败/跳过(来自 task_logs 聚合) =====
const accStats = ref<Record<string, { success: number; failed: number; skipped: number }>>({})
let accountsPageActive = true
let statsRequestId = 0
let statsController: AbortController | null = null
let taskSummaryRequestId = 0
let taskSummaryController: AbortController | null = null

async function loadStats() {
  statsController?.abort()
  const controller = new AbortController()
  statsController = controller
  const requestId = ++statsRequestId
  try {
    const result = await accountApi.stats({ signal: controller.signal })
    if (accountsPageActive && requestId === statsRequestId) accStats.value = result
  } catch { /* 统计失败或取消不阻断 */ }
  finally { if (statsController === controller) statsController = null }
}

// ===== 账号任务摘要(v1.19): 反查 running 任务, 显示每号在执行什么 =====
const taskSummary = ref<Record<string, string[]>>({})

async function loadTaskSummary() {
  taskSummaryController?.abort()
  const controller = new AbortController()
  taskSummaryController = controller
  const requestId = ++taskSummaryRequestId
  try {
    const res = await taskApi.list({ status: 'running', page: 1, page_size: 100 }, { signal: controller.signal })
    if (!accountsPageActive || requestId !== taskSummaryRequestId) return
    const map: Record<string, string[]> = {}
    for (const t of res.list || []) {
      const name = t.name || `${typeLabel(t.type)}任务`
      const ids: string[] = Array.isArray(t.account_ids) ? t.account_ids.map(String) : []
      const pids: string[] = Array.isArray(t.payload?.account_ids) ? t.payload.account_ids.map(String) : []
      for (const id of [...new Set([...ids, ...pids])]) {
        if (!map[id]) map[id] = []
        if (map[id].length < 3) map[id].push(name)
      }
    }
    if (accountsPageActive && requestId === taskSummaryRequestId) taskSummary.value = map
  } catch { /* 摘要失败或取消不阻断 */ }
  finally { if (taskSummaryController === controller) taskSummaryController = null }
}

// 能力摘要(v1.22): 体检 capabilities → 人话 tooltip
function capsSummary(caps: Record<string, any>): string {
  if (!caps) return ''
  const lines: string[] = []
  const ok = (v: any) => v === true || v === 'true' || v === 1
  if ('send_msg' in caps) lines.push(ok(caps.send_msg) ? '✓ 可发消息' : '✗ 发消息受限')
  if ('read_msg' in caps) lines.push(ok(caps.read_msg) ? '✓ 可读消息' : '✗ 读消息受限')
  if ('join_group' in caps) lines.push(ok(caps.join_group) ? '✓ 可加群' : '✗ 加群受限')
  if ('restricted' in caps) lines.push(ok(caps.restricted) ? '✗ 账号受限' : '✓ 未受限')
  if (ok(caps.is_premium)) lines.push('★ Premium 会员')
  const ban = caps.banned_until || caps.banned
  if (ban) lines.push(`⛔ 封禁至 ${ban}`)
  return lines.join('\n') || '暂无体检数据'
}

function typeLabel(t: string): string {
  const m: Record<string, string> = {
    broadcast: '群发', scheduled_broadcast: '循环群发', join_groups: '加群',
    scrape_members: '采集', scrape_links: '链接采集', invite_members: '邀请入群',
    sync_groups: '群同步', monitor: '监控',
  }
  return m[t] || '执行'
}

// 半选状态(放最后: 依赖 allSelected/filteredAccounts, 提前注册会 TDZ 爆红)
watch([selectedIds, filteredAccounts], () => {
  if (!selectAllRef.value) return
  const n = selectedIds.value.size
  selectAllRef.value.indeterminate = n > 0 && !allSelected.value
}, { deep: true, flush: 'post' })

onMounted(() => {
  accountsPageActive = true
  accountStore.fetchAccounts()
  fetchGroups()
  loadStats()
  loadTaskSummary()
})
// 30 秒轮询: 在线/受限状态与冷却恢复倒计时自动更新(配合 store 的 60 秒缓存平滑生效)
// P0-29: 原轮询只刷账号列表,统计徽标(风控/任务摘要)只在挂载时加载一次——
// 中途变化永不更新。统计一并纳入轮询。
useAutoRefresh(() => { if (!accountsPageActive) return; accountStore.fetchAccounts(); loadStats(); loadTaskSummary() }, 30000)
</script>

<style scoped>
/* ═══════════ R141 新皮肤适配: 列表/瓦片/报告 令牌化样式 ═══════════ */
/* 单容器列表: 实底卡 + 极细分割线(深色模式不透字) */
.acc-group {
  background: var(--t-card);
  border: 1px solid var(--t-line-strong);
  border-radius: var(--t-r-md);
  /* R174: 去掉 overflow:hidden —— 行内动画残留的 transform 会把 fixed 面板变成行内定位并被裁剪(面板只剩两项的根) */
  box-shadow: var(--t-shadow-sm);
}
.acc-row {
  display: flex;
  align-items: center;
  gap: var(--t-space-2);
  padding: var(--t-space-3) var(--t-space-4);
  position: relative;
}
.acc-row + .acc-row { border-top: 1px solid var(--t-line-light); }
.acc-row.standby { background: var(--t-surface-2); }
.acc-row:active { background: var(--t-surface-2); }

.acc-actions { gap: 4px; }
.acc-action-btn { min-width: 0; }
@media (max-width: 639px) {
  /* 移动端将账号信息与操作区分行，避免三个按钮挤在卡片右侧 */
  .acc-row {
    display: grid;
    grid-template-columns: 0 36px minmax(0, 1fr);
    align-items: center;
    gap: 8px;
    padding: 12px 10px;
  }
  .acc-group.selecting .acc-row {
    grid-template-columns: 26px 36px minmax(0, 1fr);
  }
  .check-col { width: auto; }
  .acc-main { grid-column: 3; grid-row: 1; }
  .acc-actions {
    grid-column: 2 / -1;
    grid-row: 2;
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
    align-items: stretch;
    width: 100%;
    gap: 8px;
    margin-top: 3px;
  }
  .acc-action-btn {
    display: inline-flex !important;
    justify-content: center;
    align-items: center;
    width: 100%;
    min-width: 0;
    min-height: 36px;
    padding: 0 8px !important;
    border-radius: 10px !important;
    font-size: 12px !important;
    line-height: 1;
  }
  .acc-action-btn :deep(svg) { width: 14px; height: 14px; }
  .acc-actions .relative { display: flex; }
  .more-btn { min-width: 64px; min-height: 36px; justify-content: center; }
  .acc-group.selecting .acc-actions { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .acc-title { flex-wrap: wrap; }
  .send-report { margin-left: 0; }
}

/* 复选框列: 默认收起(width:0), 多选模式滑出 */
.check-col {
  width: 0;
  overflow: hidden;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  transition: width var(--t-duration-slow) var(--t-ease);
}
.acc-group.selecting .check-col { width: 26px; }
.acc-check { width: 17px; height: 17px; accent-color: var(--t-brand); }
/* 多选模式下隐藏"更多"按钮(防止误触) */
.acc-group.selecting .more-btn { display: none; }

/* 两行信息区 */
.acc-main { flex: 1; min-width: 0; display: grid; gap: var(--t-space-1); }
.acc-title { display: flex; align-items: center; gap: var(--t-space-1-5); min-width: 0; }
.acc-meta {
  display: flex;
  align-items: center;
  gap: var(--t-space-2);
  flex-wrap: wrap;
  font-size: var(--t-fs-caption);
  color: var(--t-text-3);
}

/* 发送成功/失败报告(累计, 右对齐): ✓绿 ✗红, 紧凑数字 */
.send-report {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: var(--t-space-1);
  font-weight: 700;
  white-space: nowrap;
  color: var(--t-text-2);
}
.send-report i { font-style: normal; color: var(--t-ok); }
.send-report i.f { color: var(--t-err); margin-left: var(--t-space-1-5); }

/* 统计瓦片横向滑动条 */
.stat-strip {
  display: flex;
  gap: var(--t-space-3);
  overflow-x: auto;
  padding: var(--t-space-1) var(--t-space-1) var(--t-space-2);
  scroll-snap-type: x mandatory;
}
.stat-strip > * { flex-shrink: 0; width: 128px; scroll-snap-align: start; }

/* 风控轻提示(弱化) */
.risk-hint {
  display: flex;
  align-items: center;
  gap: var(--t-space-1-5);
  margin-bottom: var(--t-space-4);
  padding: 0 var(--t-space-1);
  font-size: var(--t-fs-caption);
  color: var(--t-warn);
}

/* R176: "更多"菜单 = 锚定按钮上方的小弹层(全平台一致, 与桌面端相同体验) */
.acc-menu {
  left: var(--t-space-3);
  right: var(--t-space-3);
  /* 抬高面板，避开移动端固定底部导航栏 */
  bottom: calc(env(safe-area-inset-bottom, 0px) + 96px);
  width: auto;
  max-height: min(76dvh, 640px);
  overflow-y: auto;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
  border-radius: 18px;
  padding: 8px 10px max(10px, env(safe-area-inset-bottom, 0px));
  box-shadow: var(--t-shadow-pop);
}
.acc-menu-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 6px 12px;
  border-bottom: 1px solid var(--t-line-light);
  margin-bottom: 4px;
}
.acc-menu-close {
  display: grid;
  place-items: center;
  flex: 0 0 32px;
  width: 32px;
  height: 32px;
  border-radius: 9px;
  color: var(--t-text-2);
  background: var(--t-surface-2);
  font-size: 22px;
  line-height: 1;
}
.acc-menu-section { padding: 5px 0 7px; }
.acc-menu-section + .acc-menu-section { border-top: 1px solid var(--t-line-light); }
.acc-menu-label {
  padding: 5px 9px 4px;
  color: var(--t-text-3);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: .04em;
}
.acc-menu :deep(.ui-menu-item) {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 42px;
  padding: 9px 10px;
  border-radius: 9px;
  text-align: left;
  font-size: 13px;
  transition: background .15s ease, color .15s ease;
}
.acc-menu :deep(.ui-menu-item:hover) { background: var(--t-surface-2); }
.acc-menu :deep(.ui-menu-item:focus-visible),
.acc-menu-close:focus-visible { outline: 2px solid var(--t-brand); outline-offset: 2px; }
.acc-menu-danger {
  border-top: 1px solid var(--t-line-light);
  margin-top: 2px;
  padding-top: 8px;
}
.acc-menu :deep(.ui-menu-item--danger) { color: var(--t-err); }
@media (hover: none) {
  .acc-menu :deep(.ui-menu-item:hover) { background: transparent; }
}
@media (min-width: 640px) {
  .acc-menu {
    left: auto;
    right: 0;
    top: 100%;
    bottom: auto;
    width: 15rem;
    max-height: min(70vh, 560px);
    margin-top: var(--t-space-2);
    border-radius: 14px;
    padding: 8px;
    box-shadow: var(--t-shadow-card);
  }
}
</style>
