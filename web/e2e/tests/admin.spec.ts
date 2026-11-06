import { expect, test } from '@playwright/test'

const routes = [
  '/admin/dashboard',
  '/admin/plans',
  '/admin/tenants',
  '/admin/codes',
  '/admin/support',
  '/admin/probe',
  '/admin/announcements',
  '/admin/system',
  '/admin/audit',
]

test.beforeEach(async ({ page }) => {
  const username = process.env.E2E_ADMIN_USERNAME
  const password = process.env.E2E_ADMIN_PASSWORD
  test.skip(!username || !password, '设置 E2E_ADMIN_USERNAME 和 E2E_ADMIN_PASSWORD 后才运行管理员认证巡检')

  await page.goto('/admin/login')
  const inputs = page.locator('input')
  await inputs.nth(0).fill(username!)
  await inputs.nth(1).fill(password!)
  await page.getByRole('button', { name: '登录' }).click()
  await page.waitForURL(url => url.pathname.startsWith('/admin/') && url.pathname !== '/admin/login', {
    timeout: 20_000,
  })
})

test('管理员路由可渲染且没有浏览器运行时异常', async ({ page }, testInfo) => {
  const runtimeErrors: string[] = []
  page.on('pageerror', error => runtimeErrors.push(error.message))

  for (const route of routes) {
    // Admin token is memory-only: use Vue Router's history navigation, not a document reload.
    await page.evaluate(path => {
      window.history.pushState({}, '', path)
      window.dispatchEvent(new PopStateEvent('popstate'))
    }, route)
    await expect(page).toHaveURL(new RegExp(`${route.replace(/[.*+?^${}()|[\\]\\\\]/g, '\\\\$&')}$`))
    await expect(page.locator('body'), route).toBeVisible()
    await expect(page.locator('body'), route).not.toBeEmpty()
    await page.screenshot({
      path: testInfo.outputPath(`${route.replace(/[^a-z0-9]+/gi, '_')}.png`),
      fullPage: true,
    })
  }

  expect(runtimeErrors, '管理后台不应出现未处理的前端运行时异常').toEqual([])
})
