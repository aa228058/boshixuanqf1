import { chromium } from '@playwright/test'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
import { stdin, stdout } from 'node:process'

function ask(label, masked = false) {
  if (!stdin.isTTY) throw new Error('请在服务器 SSH 终端中交互运行此命令。')
  return new Promise((resolvePromise, reject) => {
    stdout.write(label)
    let value = ''
    const wasRaw = Boolean(stdin.isRaw)
    stdin.setRawMode(true)
    stdin.resume()
    const onData = chunk => {
      const char = chunk.toString('utf8')
      if (char === '\u0003') {
        cleanup()
        reject(new Error('用户取消'))
      } else if (char === '\r' || char === '\n') {
        cleanup()
        stdout.write('\n')
        resolvePromise(value)
      } else if (char === '\u007f' || char === '\b') {
        value = value.slice(0, -1)
        stdout.write('\b \b')
      } else if (char >= ' ') {
        value += char
        stdout.write(masked ? '*' : char)
      }
    }
    function cleanup() {
      stdin.off('data', onData)
      stdin.setRawMode(wasRaw)
      stdin.pause()
    }
    stdin.on('data', onData)
  })
}

const mode = process.argv[2] ?? 'tenant'
if (!['tenant', 'admin'].includes(mode)) {
  throw new Error('用法: npm run auth:setup -- tenant|admin')
}
const baseURL = process.env.BASE_URL ?? 'https://boshixuan.duckdns.org'
const loginPath = mode === 'admin' ? '/admin/login' : '/login'
const authFile = resolve(mode === 'admin' ? '.auth/admin.json' : '.auth/user.json')
const username = await ask(mode === 'admin' ? '管理员用户名: ' : '租户用户名: ')
const password = await ask('密码（输入时以 * 隐藏）: ', true)
if (!username || !password) throw new Error('用户名和密码不能为空。')

const browser = await chromium.launch({ headless: true })
const context = await browser.newContext()
const page = await context.newPage()

try {
  await page.goto(new URL(loginPath, baseURL).toString(), { waitUntil: 'domcontentloaded' })
  if (mode === 'tenant') {
    await page.locator('input[autocomplete="username"]').fill(username)
    await page.locator('input[autocomplete="current-password"]').fill(password)
  } else {
    const inputs = page.locator('input')
    await inputs.nth(0).fill(username)
    await inputs.nth(1).fill(password)
  }

  await page.getByRole('button', { name: /登录/ }).first().click()
  await page.waitForURL(url => !url.pathname.endsWith('/login'), { timeout: 20_000 })
  await page.waitForLoadState('domcontentloaded')
  await mkdir(resolve('.auth'), { recursive: true })
  await context.storageState({ path: authFile })
  console.log(`登录状态已保存：${authFile}`)
  console.log('会话文件只保存在服务器本地，请勿提交到 Git 或发送给他人。')
} catch (error) {
  const message = await page.locator('body').innerText().catch(() => '')
  throw new Error(`登录或保存会话失败。页面提示：${message.slice(0, 500)}。原因：${error instanceof Error ? error.message : String(error)}`)
} finally {
  await browser.close()
}
