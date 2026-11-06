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

test('租户认证路由可渲染且没有浏览器运行时异常', async ({ page }, testInfo) => {
  const runtimeErrors: string[] = []
  page.on('pageerror', error => runtimeErrors.push(error.message))

  for (const route of routes) {
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

test('常见桌面视口没有明显横向溢出', async ({ page }) => {
  for (const route of ['/dashboard', '/accounts', '/settings']) {
    await page.goto(route)
    const overflow = await page.evaluate(() =>
      document.documentElement.scrollWidth > window.innerWidth + 1
    )
    expect(overflow, `${route} 存在页面级横向溢出`).toBe(false)
  }
})


test('移动端租户页面没有页面级横向溢出', async ({ page }) => {
  for (const route of routes) {
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
