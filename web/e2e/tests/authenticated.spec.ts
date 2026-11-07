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

test('租户认证路由可渲染且没有浏览器运行时异常', async ({ page }, testInfo) => {
  // 移动端逐页截图比桌面慢；给完整路由矩阵留出合理时间，避免误报超时。
  test.setTimeout(120_000)
  const runtimeErrors: string[] = []
  page.on('pageerror', error => runtimeErrors.push(error.message))

  for (const route of await tenantRoutes(page)) {
    await page.goto(route)
    await expect(page).toHaveURL(url => url.pathname === route, {
      timeout: 15_000,
    })
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
    await page.goto(route)
    await expect(page).toHaveURL(url => url.pathname === route, {
      timeout: 20_000,
    })
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
    await page.goto(route)
    await expect(page).toHaveURL(url => url.pathname === route, { timeout: 20_000 })
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
    await page.goto(route)
    await expect(page).toHaveURL(url => url.pathname === route, { timeout: 20_000 })
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
