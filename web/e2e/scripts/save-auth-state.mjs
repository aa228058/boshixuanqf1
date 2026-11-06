import { chromium } from '@playwright/test'
import { mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { createInterface } from 'node:readline/promises'
import { stdin, stdout } from 'node:process'

const baseURL = process.env.BASE_URL ?? 'https://boshixuan.duckdns.org'
const authDir = resolve('.auth')
const authFile = resolve(process.env.AUTH_STATE_PATH ?? '.auth/user.json')
const browser = await chromium.launch({ headless: false })
const context = await browser.newContext()
const page = await context.newPage()

try {
  await page.goto(new URL(process.env.LOGIN_PATH ?? '/login', baseURL).toString(), { waitUntil: 'domcontentloaded' })
  console.log('\n请在打开的浏览器中手动登录 TG Sender。')
  console.log('登录成功并进入后台后，回到此终端按 Enter 保存登录状态。')
  const readline = createInterface({ input: stdin, output: stdout })
  await readline.question('已登录？按 Enter 继续...')
  readline.close()

  if (new URL(page.url()).origin !== new URL(baseURL).origin) {
    throw new Error('当前页面已离开目标站点，未保存登录状态。')
  }
  await mkdir(authDir, { recursive: true })
  await context.storageState({ path: authFile })
  console.log(`登录状态已保存到 ${authFile}（该文件包含敏感会话信息，禁止提交到 Git）。`)
} finally {
  await browser.close()
}
