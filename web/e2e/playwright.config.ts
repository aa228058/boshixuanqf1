import { defineConfig, devices } from '@playwright/test'
import { existsSync } from 'node:fs'
import { resolve } from 'node:path'

const authState = resolve(process.cwd(), '.auth/user.json')
const generatedAuthState = resolve(process.cwd(), '.auth/e2e-user.json')
const hasAuthState = existsSync(authState)
const hasAdminCredentials = Boolean(process.env.E2E_ADMIN_USERNAME && process.env.E2E_ADMIN_PASSWORD)
const hasTenantCredentials = Boolean(process.env.E2E_TENANT_USERNAME && process.env.E2E_TENANT_PASSWORD)

const tenantAuthProjects = hasTenantCredentials
  ? [
      {
        name: 'tenant-auth-setup',
        testMatch: /auth\.setup\.ts/,
        use: { ...devices['Desktop Chrome'], storageState: { cookies: [], origins: [] } },
      },
      {
        name: 'authenticated-desktop',
        testMatch: /authenticated\.spec\.ts/,
        dependencies: ['tenant-auth-setup'],
        use: { ...devices['Desktop Chrome'], storageState: generatedAuthState },
      },
      {
        name: 'authenticated-mobile',
        testMatch: /authenticated\.spec\.ts/,
        dependencies: ['tenant-auth-setup'],
        use: { ...devices['iPhone 13'], browserName: 'chromium', storageState: generatedAuthState },
      },
    ]
  : hasAuthState
    ? [
        {
          name: 'authenticated-desktop',
          testMatch: /authenticated\.spec\.ts/,
          use: { ...devices['Desktop Chrome'], storageState: authState },
        },
        {
          name: 'authenticated-mobile',
          testMatch: /authenticated\.spec\.ts/,
          use: { ...devices['iPhone 13'], browserName: 'chromium', storageState: authState },
        },
      ]
    : []

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  workers: process.env.CI ? 1 : undefined,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
  outputDir: 'test-results',
  use: {
    baseURL: process.env.BASE_URL || 'http://127.0.0.1:4173',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 10_000,
    navigationTimeout: 30_000,
  },
  projects: [
    ...(process.env.E2E_PRODUCTION_AUDIT === '1' && (hasAuthState || hasTenantCredentials) ? [
      {
        name: 'production-desktop',
        testMatch: /production\.spec\.ts/,
        use: { ...devices['Desktop Chrome'], ...(hasAuthState ? { storageState: authState } : {}) },
      },
      {
        name: 'production-mobile',
        testMatch: /production\.spec\.ts/,
        use: { ...devices['iPhone 13'], browserName: 'chromium', ...(hasAuthState ? { storageState: authState } : {}) },
      },
    ] : []),
    {
      name: 'public-chromium',
      testMatch: /public\.spec\.ts/,
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'public-mobile',
      testMatch: /public\.spec\.ts/,
      use: { ...devices['iPhone 13'], browserName: 'chromium' },
    },
    ...tenantAuthProjects,
    ...(hasAdminCredentials ? [
      {
        name: 'admin-desktop',
        testMatch: /admin\.spec\.ts/,
        use: { ...devices['Desktop Chrome'] },
      },
      {
        name: 'admin-mobile',
        testMatch: /admin\.spec\.ts/,
        use: { ...devices['iPhone 13'], browserName: 'chromium' },
      },
    ] : []),
    // 账号页视觉审计：仅在 VISUAL_AUDIT=1 时启用。
    ...(process.env.VISUAL_AUDIT === '1' ? [
      { name: 'visual-desktop', testMatch: /accounts-visual\.spec\.ts/, use: { ...devices['Desktop Chrome'] } },
      { name: 'visual-mobile', testMatch: /accounts-visual\.spec\.ts/, use: { ...devices['iPhone 13'], browserName: 'chromium' } },
    ] : []),
  ],
  webServer: process.env.E2E_SKIP_WEBSERVER ? undefined : {
    cwd: '..',
    command: 'npx serve -s dist -l 4173',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
})
