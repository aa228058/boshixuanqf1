import { expect, test } from '@playwright/test'

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

const accountId = process.env.E2E_ACCOUNT_ID
if (accountId) {
  routes.push(
    `/workspace/${encodeURIComponent(accountId)}/broadcast`,
    `/workspace/${encodeURIComponent(accountId)}/join`,
    `/workspace/${encodeURIComponent(accountId)}/scrape`,
    `/workspace/${encodeURIComponent(accountId)}/scrape-links`,
    `/workspace/${encodeURIComponent(accountId)}/monitor`,
    `/workspace/${encodeURIComponent(accountId)}/tasks`,
  )
}

test('authenticated routes render without browser runtime errors', async ({ page }, testInfo) => {
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

  expect(runtimeErrors, '页面不应出现未处理的前端运行时异常').toEqual([])
})

test('常见视口没有明显横向溢出', async ({ page }) => {
  for (const route of ['/dashboard', '/accounts', '/settings']) {
    await page.goto(route)
    const overflow = await page.evaluate(() =>
      document.documentElement.scrollWidth > window.innerWidth + 1
    )
    expect(overflow, `${route} 存在页面级横向溢出`).toBe(false)
  }
})
