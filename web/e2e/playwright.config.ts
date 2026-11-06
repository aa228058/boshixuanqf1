import { defineConfig, devices } from '@playwright/test'
import { existsSync } from 'node:fs'
import { resolve } from 'node:path'

const authState = resolve(process.cwd(), '.auth/user.json')
const hasAuthState = existsSync(authState)

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI
    ? [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]]
    : [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
  outputDir: 'test-results',
  use: {
    baseURL: process.env.BASE_URL ?? 'http://127.0.0.1:4173',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 10_000,
    navigationTimeout: 30_000,
  },
  projects: [
    {
      name: 'public-chromium',
      testMatch: /public\.spec\.ts/,
      use: { ...devices['Desktop Chrome'] },
    },
    ...(hasAuthState ? [{
      name: 'authenticated-desktop',
      testMatch: /authenticated\.spec\.ts/,
      use: { ...devices['Desktop Chrome'], storageState: authState },
    }, {
      name: 'authenticated-mobile',
      testMatch: /authenticated\.spec\.ts/,
      use: { ...devices['iPhone 13'], storageState: authState },
    }] : []),
  ],
  webServer: process.env.E2E_SKIP_WEBSERVER ? undefined : {
    command: 'npm run preview -- --host 127.0.0.1 --port 4173 --strictPort',
    url: 'http://127.0.0.1:4173',
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
})
