import { expect, test } from '@playwright/test'

const username = process.env.E2E_TENANT_USERNAME
const password = process.env.E2E_TENANT_PASSWORD

const baseRoutes = [
  '/dashboard',
  '/accounts',
  '/settings',
  '/groups/discover',
  '/verifier/logs',
  '/help',
  '/raffle',
  '/more',
]

async function auditedRoutes(page: import('@playwright/test').Page): Promise<string[]> {
  let accountId = process.env.E2E_ACCOUNT_ID
  if (!accountId) {
    accountId = await page.evaluate(() => localStorage.getItem('tg_current_account') || '')
  }
  if (!accountId) accountId = 'none'

  return [
    ...baseRoutes,
    `/workspace/${accountId}/broadcast`,
    `/workspace/${accountId}/join`,
    `/workspace/${accountId}/scrape`,
    `/workspace/${accountId}/scrape-links`,
    `/workspace/${accountId}/monitor`,
    `/workspace/${accountId}/tasks`,
  ]
}

test('production routes render on this device and save a screenshot for every page', async ({ page }, testInfo) => {
  test.skip(!username || !password, 'Set E2E_TENANT_USERNAME and E2E_TENANT_PASSWORD as GitHub Actions secrets.')
  test.setTimeout(240_000)

  const runtimeErrors: string[] = []
  page.on('pageerror', error => runtimeErrors.push(error.message))

  // Log in with a dedicated test account. No forms that create, send, join, delete, or modify data are submitted.
  await page.goto('/login')
  await page.locator('input[autocomplete="username"]').fill(username!)
  await page.locator('input[autocomplete="current-password"]').fill(password!)
  await page.getByRole('button', { name: /登录并进入工作台/ }).click()
  await page.waitForURL(url => url.pathname === '/dashboard', { timeout: 30_000 })

  const routes = await auditedRoutes(page)
  const failures: string[] = []

  for (const route of routes) {
    const response = await page.goto(route, { waitUntil: 'domcontentloaded' })
    await expect(page.locator('body'), route).toBeVisible()
    await page.waitForTimeout(700)

    const actualPath = new URL(page.url()).pathname
    const bodyText = await page.locator('body').innerText().catch(() => '')
    const dimensions = await page.evaluate(() => ({
      viewport: document.documentElement.clientWidth,
      document: document.documentElement.scrollWidth,
      body: document.body.scrollWidth,
    }))

    if (!response || response.status() >= 400) failures.push(`${route}: HTTP ${response?.status() ?? 'no response'}`)
    if (actualPath !== route) failures.push(`${route}: redirected to ${actualPath}`)
    if (/页面不存在|Page not found|404 Not Found/i.test(bodyText)) failures.push(`${route}: rendered a not-found page`)
    if (dimensions.document > dimensions.viewport + 1 || dimensions.body > dimensions.viewport + 1) {
      failures.push(`${route}: horizontal overflow ${JSON.stringify(dimensions)}`)
    }

    const filename = route.replace(/[^a-z0-9]+/gi, '_').replace(/^_|_$/g, '') || 'root'
    await page.screenshot({
      path: testInfo.outputPath(`${filename}.png`),
      fullPage: true,
    })
  }

  expect(failures, 'Production route and responsive layout audit findings').toEqual([])
  expect(runtimeErrors, 'No uncaught frontend runtime errors during route traversal').toEqual([])
})
