import { expect, test } from '@playwright/test'

// 账号页视觉与关键交互审计: 不依赖后端与真实账号。
// 通过 route 拦截模拟登录会话与账号接口, 覆盖桌面端和移动端。
// 只在 VISUAL_AUDIT=1 时由 playwright.config.ts 启用。

const sessionKey = 'tg_sender_session'

const tenantProfile = {
  tenant_id: 'e2e-tenant',
  tenant_name: '示例租户',
  name: '示例租户',
  plan_id: 'pro',
  status: 'active',
  expires_at: '2027-01-01T00:00:00Z',
  plan_features: {},
  max_accounts: 20,
  max_concurrent_tasks: 5,
}

const demoAccounts = [
  {
    account_id: 'e2e-acc-01', tenant_id: 'e2e-tenant', phone: '+86 138****1024',
    username: 'demo_a', first_name: '示例', status: 'online', remark: '示例账号 A',
    created_at: '2026-11-01T08:00:00Z', source: 'phone',
  },
  {
    account_id: 'e2e-acc-02', tenant_id: 'e2e-tenant', phone: '+86 139****7731',
    username: 'demo_b', first_name: '示例', status: 'pending_auth', remark: '示例账号 B',
    created_at: '2026-11-02T08:00:00Z', source: 'phone',
  },
  {
    account_id: 'e2e-acc-03', tenant_id: 'e2e-tenant', phone: '+86 150****9058',
    username: 'demo_c', first_name: '示例', status: 'offline', remark: '示例账号 C',
    created_at: '2026-11-03T08:00:00Z', source: 'protocol',
  },
]

test.describe('账号页视觉审计', () => {
  test('账号页与更多菜单在桌面端和移动端正常显示', async ({ page }, testInfo) => {
    await page.addInitScript((key) => {
      try { localStorage.setItem(key, '1') } catch { /* 隐私模式可能禁用 localStorage */ }
    }, sessionKey)

    await page.route('**/api/v1/**', async route => {
      const request = route.request()
      const path = new URL(request.url()).pathname.replace('/api/v1', '')
      const ok = (data: unknown) => route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ code: 0, data }),
      })

      if (path === '/auth/refresh') return ok({ token: 'e2e-visual-token' })
      if (path === '/auth/me') return ok(tenantProfile)
      if (path === '/accounts' && request.method() === 'GET') {
        return ok({ list: demoAccounts, total: demoAccounts.length, page: 1, page_size: 50 })
      }
      if (request.method() === 'GET') return ok({ list: [], total: 0 })
      return ok({})
    })

    await page.goto('/accounts')
    await expect(page.getByRole('heading', { name: '账号管理' })).toBeVisible({ timeout: 20_000 })

    const firstActions = page.locator('.acc-actions').first()
    await expect(firstActions.getByRole('button', { name: '备用' })).toBeVisible()
    await expect(firstActions.getByRole('button', { name: '检测' })).toBeVisible()
    const moreButton = firstActions.getByRole('button', { name: '更多操作' })
    await expect(moreButton).toBeVisible()

    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
    )
    expect(overflow, '页面不应横向溢出').toBeLessThanOrEqual(1)

    await page.screenshot({
      path: testInfo.outputPath(`accounts-${testInfo.project.name}.png`),
      fullPage: true,
    })

    await moreButton.click()
    const menu = page.locator('.acc-menu')
    await expect(menu).toBeVisible()
    await expect(menu.getByText('连接与登录')).toBeVisible()
    await expect(menu.getByText('账号维护')).toBeVisible()
    await expect(menu.getByText('资料与备注')).toBeVisible()
    await expect(menu.getByRole('button', { name: '断开连接' })).toBeVisible()
    await expect(menu.getByRole('button', { name: '删除账号' })).toBeVisible()
    await expect(menu.getByRole('button', { name: '备用' })).toHaveCount(0)
    await expect(menu.getByRole('button', { name: '检测' })).toHaveCount(0)

    const menuBox = await menu.boundingBox()
    const viewport = page.viewportSize()
    expect(menuBox, '更多菜单必须有可见尺寸').not.toBeNull()
    expect(viewport).not.toBeNull()
    if (testInfo.project.name.includes('mobile')) {
      expect(menuBox!.x).toBeGreaterThanOrEqual(0)
      expect(menuBox!.x + menuBox!.width).toBeLessThanOrEqual(viewport!.width + 1)
      expect(menuBox!.y + menuBox!.height).toBeLessThanOrEqual(viewport!.height - 88)
    } else {
      expect(menuBox!.width).toBeLessThanOrEqual(260)
    }

    await page.screenshot({
      path: testInfo.outputPath(`accounts-menu-${testInfo.project.name}.png`),
      fullPage: true,
    })
    await page.getByRole('button', { name: '关闭更多菜单' }).click()
    await expect(menu).toHaveCount(0)
  })
})
