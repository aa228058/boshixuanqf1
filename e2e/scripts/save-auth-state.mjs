import { chromium } from '@playwright/test'
import { mkdir, writeFile, chmod } from 'node:fs/promises'
import { resolve } from 'node:path'
import { stdin, stdout } from 'node:process'

function ask(label, masked = false) {
  if (!stdin.isTTY || typeof stdin.setRawMode !== 'function') {
    throw new Error('请在支持交互输入的服务器 SSH 终端中运行此命令。')
  }
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
        stdout.write('\n')
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
if (mode !== 'tenant') {
  throw new Error('目前只支持租户会话保存。管理员 Token 仅在内存中有效，管理员巡检会在测试进程内登录。')
}

const baseURL = process.env.BASE_URL ?? 'https://boshixuan.duckdns.org'
const authFile = resolve('.auth/user.json')
const username = await ask('租户用户名: ')
const password = await ask('密码（输入时以 * 隐藏）: ', true)
if (!username || !password) throw new Error('用户名和密码不能为空。')

const browser = await chromium.launch({ headless: true })
const context = await browser.newContext()
const page = await context.newPage()

try {
  await page.goto(new URL('/login', baseURL).toString(), { waitUntil: 'domcontentloaded' })
  await page.locator('input[autocomplete="username"]').fill(username)
  await page.locator('input[autocomplete="current-password"]').fill(password)
  await page.getByRole('button', { name: /登录并进入工作台/ }).click()
  await page.waitForURL(url => !['/login', '/admin/login'].includes(url.pathname), { timeout: 20_000 })
  await page.waitForLoadState('domcontentloaded')

  const storage = await context.storageState()
  await mkdir(resolve('.auth'), { recursive: true, mode: 0o700 })
  await chmod(resolve('.auth'), 0o700)
  await writeFile(authFile, JSON.stringify(storage, null, 2), { mode: 0o600 })
  await chmod(authFile, 0o600)
  console.log('租户登录状态已安全保存到 .auth/user.json（目录权限 700，文件权限 600）。')
  console.log('该文件包含可用会话凭据，只能留在服务器本机，切勿提交、上传或转发。')
} catch (error) {
  const message = await page.locator('body').innerText().catch(() => '')
  throw new Error(`租户登录或保存会话失败。页面提示：${message.slice(0, 300)}。原因：${error instanceof Error ? error.message : String(error)}`)
} finally {
  await browser.close()
}
