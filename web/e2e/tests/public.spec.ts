import { expect, test } from '@playwright/test'

test('租户登录页可访问且核心表单可见', async ({ page }, testInfo) => {
  await page.goto('/login')
  await expect(page.locator('body')).toBeVisible()
  await expect(page.locator('input[autocomplete="username"]')).toBeVisible()
  await expect(page.locator('input[autocomplete="current-password"]')).toBeVisible()
  await expect(page.getByRole('button', { name: /登录并进入工作台/ })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('login-desktop.png'), fullPage: true })
})

test('管理员登录页可访问', async ({ page }, testInfo) => {
  await page.goto('/admin/login')
  await expect(page.locator('body')).toBeVisible()
  await expect(page.locator('input[type="password"]').first()).toBeVisible()
  await expect(page.getByRole('button', { name: '登录' })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('admin-login-desktop.png'), fullPage: true })
})

test('注册页可访问', async ({ page }, testInfo) => {
  await page.goto('/register')
  await expect(page.locator('body')).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('register-desktop.png'), fullPage: true })
})

test('未知路由显示 404 页面而非空白', async ({ page }) => {
  await page.goto('/__playwright_not_found_probe__')
  await expect(page.locator('body')).toBeVisible()
  await expect(page.locator('body')).not.toBeEmpty()
})
