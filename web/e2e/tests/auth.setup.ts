import { expect, test } from '@playwright/test'
import { mkdir } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'

const authFile = resolve(process.cwd(), '.auth/user.json')

test('authenticate tenant once for the browser audit', async ({ page }) => {
  const username = process.env.E2E_TENANT_USERNAME
  const password = process.env.E2E_TENANT_PASSWORD
  test.skip(!username || !password, 'tenant credentials are not configured')

  await page.goto('/login')
  await page.locator('input[autocomplete="username"]').fill(username!)
  await page.locator('input[autocomplete="current-password"]').fill(password!)
  await page.getByRole('button', { name: /登录并进入工作台/ }).click()
  await expect(page).toHaveURL(url => url.pathname === '/dashboard', { timeout: 30_000 })

  await mkdir(dirname(authFile), { recursive: true })
  await page.context().storageState({ path: authFile })
})
