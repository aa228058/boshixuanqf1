import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const changed = process.argv.slice(2)
const files = changed.length
  ? changed
  : execFileSync('git', ['diff', '--name-only', 'HEAD^', 'HEAD', '--', 'web/src'], { encoding: 'utf8' })
      .split(/\r?\n/)
      .filter((file) => file.endsWith('.vue'))

const violations = []
for (const file of files) {
  const absolute = resolve(process.cwd(), file.replace(/^web\//, ''))
  let source
  try {
    source = readFileSync(absolute, 'utf8')
  } catch {
    continue
  }

  const styleBlocks = [...source.matchAll(/<style\b[^>]*>([\s\S]*?)<\/style>/g)]
  for (const [, rawCss] of styleBlocks) {
    const css = rawCss.replace(/\/\*[\s\S]*?\*\//g, '')
    const checks = [
      ['color literal', /#[\da-f]{3,8}\b|\b(?:rgba?|hsla?)\s*\(/i],
      ['hard-coded shadow', /box-shadow\s*:\s*(?!none\b|var\()/i],
      ['hard-coded radius', /border-radius\s*:\s*(?!0(?:\s|;|$)|var\()/i],
      ['hard-coded font size', /font-size\s*:\s*(?!0(?:\s|;|$)|var\()/i],
    ]

    for (const [label, pattern] of checks) {
      if (pattern.test(css)) violations.push(`${file}: ${label}`)
    }

    for (const match of css.matchAll(/(?:padding|margin|gap)(?:-[a-z-]+)?\s*:\s*([^;]+);/gi)) {
      const value = match[1].replace(/var\([^)]*\)|calc\([^)]*\)|env\([^)]*\)/g, '')
      if (/(?:[1-9]\d*(?:\.\d+)?|0?\.\d+)(?:px|rem|em|vw|vh|%)/i.test(value)) {
        violations.push(`${file}: hard-coded spacing (${match[0].trim()})`)
      }
    }
  }
}

if (violations.length) {
  console.error('Design token check failed:')
  for (const violation of [...new Set(violations)]) console.error(` - ${violation}`)
  process.exit(1)
}

console.log(`Design token check passed (${files.length} changed Vue file(s)).`)
