import { expect, test } from '@playwright/test'
import { mkdir } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'

const username = process.env.E2E_TENANT_USERNAME
const password = process.env.E2E_TENANT_PASSWORD
const statePath = resolve(process.cwd(), '.auth/e2e-user.json')

test('sign in once and save tenant browser state', async ({ page }) => {
  test.skip(!username || !password, 'Tenant E2E credentials are not configured')
  await page.goto('/login')
  await page.locator('input[autocomplete="username"]').fill(username as string)
  await page.locator('input[autocomplete="current-password"]').fill(password as string)
  await page.getByRole('button', { name: /登录并进入工作台/ }).click()
  await expect(page).toHaveURL(url => url.pathname === '/dashboard', { timeout: 20_000 })
  await mkdir(dirname(statePath), { recursive: true })
  await page.context().storageState({ path: statePath })
})
