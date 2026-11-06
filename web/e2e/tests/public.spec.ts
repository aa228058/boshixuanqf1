import { expect, test } from '@playwright/test'

test('登录页可访问且核心表单可见', async ({ page }) => {
  await page.goto('/login')
  await expect(page.locator('body')).toBeVisible()
  await expect(page.locator('input').first()).toBeVisible()
  await expect(page.locator('input[type="password"]').first()).toBeVisible()
  await expect(page.getByRole('button', { name: /登录/ }).first()).toBeVisible()
  await page.screenshot({ path: 'screenshots/login-desktop.png', fullPage: true })
})

test('管理员登录页可访问', async ({ page }) => {\n  await page.goto('/admin/login')\n  await expect(page.locator('body')).toBeVisible()\n  await expect(page.locator('input[type="password"]').first()).toBeVisible()\n})\n\ntest('注册页可访问', async ({ page }) => {
  await page.goto('/register')
  await expect(page.locator('body')).toBeVisible()
  await page.screenshot({ path: 'screenshots/register-desktop.png', fullPage: true })
})

test('未知路由显示 404 页面而非空白', async ({ page }) => {
  await page.goto('/__playwright_not_found_probe__')
  await expect(page.locator('body')).toBeVisible()
  await expect(page.locator('body')).not.toBeEmpty()
})
