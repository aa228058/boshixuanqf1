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

test('admin routes render without browser runtime errors', async ({ page }, testInfo) => {
  const runtimeErrors: string[] = []
  page.on('pageerror', error => runtimeErrors.push(error.message))

  for (const route of routes) {
    await page.goto(route)
    await expect(page.locator('body'), route).toBeVisible()
    await expect(page.locator('body'), route).not.toBeEmpty()
    await page.screenshot({
      path: `screenshots/${testInfo.project.name}-${route.replace(/[^a-z0-9]+/gi, '_')}.png`,
      fullPage: true,
    })
  }

  expect(runtimeErrors, '管理后台不应出现未处理的前端运行时异常').toEqual([])
})
