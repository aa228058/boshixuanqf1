import { expect, test } from '@playwright/test'

const tenantUsername = process.env.E2E_TENANT_USERNAME
const tenantPassword = process.env.E2E_TENANT_PASSWORD

test.beforeEach(async ({ page }) => {
  if (!tenantUsername || !tenantPassword) return // 使用项目配置加载的 .auth/user.json
  await page.goto('/login')
  await page.locator('input[autocomplete="username"]').fill(tenantUsername)
  await page.locator('input[autocomplete="current-password"]').fill(tenantPassword)
  await page.getByRole('button', { name: /登录并进入工作台/ }).click()
  await page.waitForURL(url => url.pathname === '/dashboard', { timeout: 20_000 })
})

const routes = [
  '/dashboard',
  '/more',
  '/accounts',
  '/settings',
  '/groups/discover',
  '/verifier/logs',
  '/updates',
  '/help',
  '/raffle',
  '/sessions',
]

// 通过 Vue 应用实例调用真实 Vue Router，避免整页 reload 重复恢复会话，
// 同时不伪造 history/popstate 事件，也不绕过路由守卫。
async function navigateTenantRoute(page: import('@playwright/test').Page, route: string) {
  const finalPath = await page.evaluate(async (path) => {
    const root = document.querySelector('#app') as (HTMLElement & {
      __vue_app__?: {
        config: { globalProperties: { $router?: {
          push: (to: string) => Promise<unknown>
          currentRoute: { value: { fullPath: string } }
        } } }
      }
    }) | null
    const router = root?.__vue_app__?.config.globalProperties.$router
    if (!router) throw new Error('无法从 Vue 应用实例获取 Vue Router')
    await router.push(path)
    return router.currentRoute.value.fullPath
  }, route)
  await expect(page).toHaveURL(url => url.pathname === new URL(finalPath, url.origin).pathname, { timeout: 15_000 })
}

async function tenantRoutes(page: import('@playwright/test').Page): Promise<string[]> {
  let accountId = process.env.E2E_ACCOUNT_ID
  if (!accountId) {
    await page.waitForFunction(
      () => Boolean(localStorage.getItem('tg_current_account')),
      null,
      { timeout: 5_000 },
    ).catch(() => undefined)
    accountId = await page.evaluate(() => localStorage.getItem('tg_current_account') || '')
  }

  // The app itself routes to the explicit "none" workspace when no account is selected.
  // Auditing that fallback still exercises the tenant workspace shell and empty states.
  if (!accountId) accountId = 'none'
  return [
    ...routes,
    `/workspace/${accountId}/broadcast`,
    `/workspace/${accountId}/join`,
    `/workspace/${accountId}/scrape`,
    `/workspace/${accountId}/scrape-links`,
    `/workspace/${accountId}/monitor`,
    `/workspace/${accountId}/tasks`,
  ]
}

// 仅拦截群库只读 API 的确定性演示数据；不写数据库、不执行 Telegram 操作。
test('群组发现页演示数据视觉回归（内容态与已选态）', async ({ page }, testInfo) => {
  const demoGroups = [
    ['e2e-01', '产品交流与资源分享', 'product_hub_cn', 'supergroup', 12840, 'allowed', 'valid', 'scraped'],
    ['e2e-02', '跨境电商运营讨论', 'crossborder_ops', 'group', 5630, 'restricted', 'valid', 'search'],
    ['e2e-03', 'Telegram 自动化开发', 'tg_dev_lab', 'channel', 42100, 'unknown', 'pending', 'bio'],
    ['e2e-04', '独立站卖家互助群', 'indie_sellers', 'supergroup', 2860, 'allowed', 'valid', 'import'],
    ['e2e-05', '运营素材与案例库', 'ops_cases', 'channel', 97500, 'unknown', 'invalid', 'scraped'],
    ['e2e-06', 'SaaS 产品增长交流', 'saas_growth_cn', 'supergroup', 7340, 'allowed', 'valid', 'search'],
    ['e2e-07', '海外营销观察', 'global_marketing', 'channel', 18300, 'unknown', 'pending', 'bio'],
    ['e2e-08', '开发者闲聊群', 'dev_chat_room', 'group', 1180, 'restricted', 'valid', 'joined'],
    ['e2e-09', '电报工具测评频道', 'telegram_tools_review', 'channel', 65200, 'unknown', 'valid', 'import'],
    ['e2e-10', '本地商家交流圈', 'local_business_cn', 'supergroup', 3420, 'allowed', 'pending', 'scraped'],
  ].map(([id, title, username, gtype, member_count, free_speech_status, status, source], i) => ({
    sg_id: id as string, title: title as string, username: username as string,
    tme_link: 'https://t.me/' + username, gtype: gtype as string,
    member_count: member_count as number, free_speech_status: free_speech_status as string,
    status: status as string, source: source as string, ref_count: i + 1,
    entry_count: i === 0 ? 2 : 1,
    entries: i === 0 ? [
      { sg_id: id as string, tme_link: 'https://t.me/' + username, username: username as string, source: source as string, status: status as string },
      { sg_id: 'e2e-01b', tme_link: 'https://t.me/+DemoInvite001', source: 'import', status: 'valid' },
    ] : undefined,
    last_seen_at: new Date(Date.UTC(2026, 10, 6 - i, 9, 30)).toISOString(),
  }))
  await page.route(/\/api\/v1\/group-library(?:\\?.*)?$/, async route => {
    if (route.request().method() !== 'GET') return route.continue()
    const url = new URL(route.request().url())
    const keyword = (url.searchParams.get('keyword') || '').toLowerCase()
    const gtype = url.searchParams.get('gtype') || ''
    const status = url.searchParams.get('status') || ''
    const source = url.searchParams.get('source') || ''
    const rows = demoGroups.filter(group =>
      (!keyword || group.title.toLowerCase().includes(keyword) || group.username.includes(keyword)) &&
      (!gtype || group.gtype === gtype) &&
      (!status || (status === 'active' ? group.status !== 'invalid' : group.status === status)) &&
      (!source || group.source === source),
    )
    await route.fulfill({
      status: 200, contentType: 'application/json',
      body: JSON.stringify({ code: 0, data: rows, total: rows.length, page: 1, page_size: 20 }),
    })
  })

  await page.goto('/groups/discover')
  await expect(page.getByRole('heading', { name: '发现群组' })).toBeVisible()
  const visibleGroupTitle = (title: string) =>
    page.locator('span:visible').filter({ hasText: title }).first()
  await expect(visibleGroupTitle('产品交流与资源分享')).toBeVisible()
  await expect(visibleGroupTitle('跨境电商运营讨论')).toBeVisible()
  await expect(visibleGroupTitle('运营素材与案例库')).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('groups-discover-populated.png'), fullPage: true })

  const joinButton = page.getByRole('button', { name: /加入所选/ })
  await expect(joinButton).toBeDisabled()
  if (testInfo.project.name.includes('mobile')) {
    const groupCard = page.locator('.md\\:hidden .ui-card').filter({ hasText: '产品交流与资源分享' })
    await groupCard.locator('input[type="checkbox"]').check()
  } else {
    const groupRow = page.getByRole('row').filter({ hasText: '产品交流与资源分享' })
    await groupRow.getByRole('checkbox').check()
  }
  await expect(joinButton).toBeEnabled()
  await expect(joinButton).toContainText('1')
  await page.screenshot({ path: testInfo.outputPath('groups-discover-selected.png'), fullPage: true })
})

test('租户认证路由可渲染且没有浏览器运行时异常', async ({ page }, testInfo) => {
  // 移动端逐页截图比桌面慢；给完整路由矩阵留出合理时间，避免误报超时。
  test.setTimeout(120_000)
  const runtimeErrors: string[] = []
  page.on('pageerror', error => runtimeErrors.push(error.message))

  for (const route of await tenantRoutes(page)) {
    await navigateTenantRoute(page, route)
    await expect(page.locator('body'), route).toBeVisible()
    await expect(page.locator('body'), route).not.toBeEmpty()
    await page.waitForTimeout(800)
    await page.screenshot({
      path: testInfo.outputPath(`${route.replace(/[^a-z0-9]+/gi, '_')}.png`),
      fullPage: true,
    })
  }

  expect(runtimeErrors, '页面不应出现未处理的前端运行时异常').toEqual([])
})


test('会话在页面切换和硬刷新后仍然有效', async ({ page }) => {
  const checkedRoutes = ['/dashboard', '/accounts', '/settings', '/groups/discover', '/help']

  for (const route of checkedRoutes) {
    await navigateTenantRoute(page, route)
    await expect(page.locator('body')).not.toBeEmpty()
  }

  await page.reload()
  await expect(page).toHaveURL(url => url.pathname === '/help', {
    timeout: 20_000,
  })
  await expect(page.locator('body')).not.toBeEmpty()

  await page.goto('/dashboard')
  await expect(page).toHaveURL(url => url.pathname === '/dashboard', {
    timeout: 20_000,
  })
})

test('租户页面没有页面级横向溢出', async ({ page }) => {
  for (const route of await tenantRoutes(page)) {
    await navigateTenantRoute(page, route)
    await page.waitForTimeout(500)
    const dimensions = await page.evaluate(() => ({
      viewport: document.documentElement.clientWidth,
      document: document.documentElement.scrollWidth,
      body: document.body.scrollWidth,
      main: document.querySelector('main')?.scrollWidth ?? 0,
    }))
    expect(dimensions.document, `${route} 桌面页面级横向溢出: ${JSON.stringify(dimensions)}`).toBeLessThanOrEqual(dimensions.viewport + 1)
    expect(dimensions.body, `${route} 桌面 body 横向溢出: ${JSON.stringify(dimensions)}`).toBeLessThanOrEqual(dimensions.viewport + 1)
  }
})


test('移动端租户页面没有页面级横向溢出', async ({ page }) => {
  for (const route of await tenantRoutes(page)) {
    await navigateTenantRoute(page, route)
    await page.waitForTimeout(500)
    const dimensions = await page.evaluate(() => ({
      viewport: document.documentElement.clientWidth,
      document: document.documentElement.scrollWidth,
      body: document.body.scrollWidth,
    }))
    expect(dimensions.document, `${route} 页面级横向溢出: ${JSON.stringify(dimensions)}`).toBeLessThanOrEqual(dimensions.viewport + 1)
    expect(dimensions.body, `${route} body 横向溢出: ${JSON.stringify(dimensions)}`).toBeLessThanOrEqual(dimensions.viewport + 1)
  }
})

test('添加账号弹窗在桌面端居中、移动端贴合底部', async ({ page }, testInfo) => {
  await page.goto('/accounts')
  const addButton = page.getByRole('button', { name: '添加账号' }).first()
  await expect(addButton).toBeVisible()
  await addButton.click()

  const visibleModals = page.locator('.modal-card:visible')
  await expect(visibleModals).toHaveCount(1)
  const modal = visibleModals.first()
  await expect(modal).toBeVisible()
  // 等待 Vue Final Modal 的进入动画完成后再测量最终布局，避免把 translateY(100%) 过渡帧误判为溢出。
  await modal.evaluate(async (element) => {
    const ancestors: Element[] = []
    let current: Element | null = element
    for (let depth = 0; current && depth < 4; depth += 1, current = current.parentElement) {
      ancestors.push(current)
    }
    await Promise.all(ancestors.flatMap(node =>
      node.getAnimations().map(animation => animation.finished.catch(() => undefined)),
    ))
  })
  const layoutDiagnostics = await modal.evaluate((element) => {
    const rect = (node: Element) => {
      const r = node.getBoundingClientRect()
      const s = getComputedStyle(node)
      return {
        tag: node.tagName,
        className: typeof node.className === 'string' ? node.className : '',
        rect: { x: r.x, y: r.y, top: r.top, right: r.right, bottom: r.bottom, width: r.width, height: r.height },
        style: { position: s.position, display: s.display, alignItems: s.alignItems, justifyContent: s.justifyContent, inset: s.inset, top: s.top, right: s.right, bottom: s.bottom, left: s.left, height: s.height, margin: s.margin, padding: s.padding, transform: s.transform, overflow: s.overflow, boxSizing: s.boxSizing },
      }
    }
    const ancestors: Element[] = []
    let node: Element | null = element
    while (node && ancestors.length < 6) { ancestors.push(node); node = node.parentElement }
    return {
      viewport: { innerWidth: window.innerWidth, innerHeight: window.innerHeight, visual: window.visualViewport ? { width: window.visualViewport.width, height: window.visualViewport.height, offsetTop: window.visualViewport.offsetTop, pageTop: window.visualViewport.pageTop } : null },
      keyboardOffset: getComputedStyle(document.documentElement).getPropertyValue('--modal-kb'),
      ancestors: ancestors.map(rect),
    }
  })
  console.log('[modal-layout-diagnostics]', JSON.stringify(layoutDiagnostics))
  const bounds = await modal.boundingBox()
  const viewport = page.viewportSize()
  expect(bounds, '弹窗应有可见布局边界').not.toBeNull()
  expect(viewport).not.toBeNull()
  if (!bounds || !viewport) return

  if (testInfo.project.name.includes('mobile')) {
    expect(bounds.y + bounds.height, '移动端弹窗应贴合视口底部').toBeGreaterThanOrEqual(viewport.height - 4)
    expect(bounds.y + bounds.height, '移动端弹窗不能溢出视口').toBeLessThanOrEqual(viewport.height + 1)
  } else {
    expect(Math.abs(bounds.x + bounds.width / 2 - viewport.width / 2), '桌面端弹窗应水平居中').toBeLessThanOrEqual(5)
    expect(Math.abs(bounds.y + bounds.height / 2 - viewport.height / 2), '桌面端弹窗应垂直居中').toBeLessThanOrEqual(5)
  }
})
