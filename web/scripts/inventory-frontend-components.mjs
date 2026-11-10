#!/usr/bin/env node
/**
 * Frontend component inventory.
 *
 * Reports direct component imports, possible unreferenced SFCs, duplicate component
 * basenames, and global stylesheet loading order. This is a conservative inventory:
 * dynamic imports, auto-registration, runtime string references and template aliases
 * can make a component appear unused, so the report never deletes files.
 */
import { readdir, readFile } from 'node:fs/promises'
import { dirname, extname, join, relative, resolve } from 'node:path'

const root = resolve(process.cwd(), 'src')
const sourceExt = new Set(['.vue', '.ts', '.tsx', '.js', '.jsx'])
const files = []

async function walk(dir) {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) await walk(full)
    else if (sourceExt.has(extname(entry.name))) files.push(full)
  }
}

await walk(root)
const contents = new Map()
for (const file of files) contents.set(file, await readFile(file, 'utf8'))

const componentFiles = files.filter(file => file.endsWith('.vue') && relative(root, file).startsWith('components/'))
const references = new Map(componentFiles.map(file => [file, new Set()]))
const importPattern = /(?:from\s*|import\s*)['"]([^'"]+)['"]/g

for (const [source, text] of contents) {
  for (const match of text.matchAll(importPattern)) {
    const specifier = match[1]
    if (!specifier.startsWith('.') && !specifier.startsWith('@/')) continue
    const base = specifier.startsWith('@/') ? join(root, specifier.slice(2)) : resolve(dirname(source), specifier)
    const candidates = [base, base + '.vue', join(base, 'index.vue')]
    for (const candidate of candidates) {
      const target = resolve(candidate)
      if (references.has(target)) references.get(target).add(relative(root, source))
    }
  }
}

const byName = new Map()
for (const file of componentFiles) {
  const name = file.split('/').at(-1)
  if (!byName.has(name)) byName.set(name, [])
  byName.get(name).push(relative(root, file))
}

const mainFile = files.find(file => relative(root, file) === 'main.ts')
const mainText = mainFile ? contents.get(mainFile) : ''
const globalStyles = [...mainText.matchAll(/import\s+['"]([^'"]+\.css)['"]/g)].map(match => match[1])
const buttonPatterns = [
  ['TDesign <t-button>', /<t-button\b/gi],
  ['Native <button>', /<button\b/gi],
  ['Shared .ui-t-button class', /ui-t-button/g],
  ['Shared .ui-native-button class', /ui-native-button/g],
]
const ungovernedTButtons = files
  .filter(file => file.endsWith('.vue'))
  .flatMap(file => {
    const source = contents.get(file)
    return [...source.matchAll(/<t-button\b[^>]*>/gs)]
      .filter(match => !match[0].includes('ui-t-button'))
      .map(match => ({ path: relative(root, file), line: source.slice(0, match.index).split('\n').length }))
  })

const iconButtonsMissingName = files
  .filter(file => file.endsWith('.vue'))
  .flatMap(file => {
    const source = contents.get(file)
    return [...source.matchAll(/<button\b[^>]*>/gs)]
      .filter(match => /\bui-icon-btn\b/.test(match[0]) && !/\b(?:aria-label|title)\s*=/.test(match[0]))
      .map(match => ({ path: relative(root, file), line: source.slice(0, match.index).split('\n').length }))
  })

const formControlPatterns = [
  ['TDesign <t-input>', /<t-input\b/gi],
  ['TDesign <t-textarea>', /<t-textarea\b/gi],
  ['TDesign <t-input-number>', /<t-input-number\b/gi],
  ['TDesign <t-select>', /<t-select\b/gi],
  ['TDesign <t-checkbox>', /<t-checkbox\b/gi],
  ['TDesign <t-switch>', /<t-switch\b/gi],
  ['TDesign <t-date-picker>', /<t-date-picker\b/gi],
  ['TDesign <t-table>', /<t-table\b/gi],
  ['TDesign <t-pagination>', /<t-pagination\b/gi],
  ['Native <table>', /<table\b/gi],
  ['Native <thead>', /<thead\b/gi],
  ['Native <tbody>', /<tbody\b/gi],
  ['Native <tr>', /<tr\b/gi],
  ['Native <input>', /<input\b/gi],
  ['Native <textarea>', /<textarea\b/gi],
  ['Native <select>', /<select\b/gi],
]
const formControlUsage = formControlPatterns.map(([label, pattern]) => ({
  label,
  total: [...contents.values()].reduce((sum, source) => sum + [...source.matchAll(pattern)].length, 0),
  files: [...contents.entries()]
    .map(([file, source]) => ({ path: relative(root, file), count: [...source.matchAll(pattern)].length }))
    .filter(entry => entry.count > 0)
    .sort((a, b) => b.count - a.count),
}))

const confirmationPatterns = [
  ['Native window.confirm()', /window\.confirm\s*\(/g],
  ['Native window.alert()', /window\.alert\s*\(/g],
  ['Shared ask() calls', /\bask\s*\(/g],
]
const confirmationUsage = confirmationPatterns.map(([label, pattern]) => ({
  label,
  total: [...contents.values()].reduce((sum, source) => sum + [...source.matchAll(pattern)].length, 0),
  files: [...contents.entries()]
    .map(([file, source]) => ({ path: relative(root, file), count: [...source.matchAll(pattern)].length }))
    .filter(entry => entry.count > 0)
    .sort((a, b) => b.count - a.count),
}))

const stateAndOverlayPatterns = [
  ['TDesign <t-dialog>', /<t-dialog\b/gi],
  ['TDesign <Dialog>', /<Dialog\b/g],
  ['Dynamic resolveComponent("Dialog")', /resolveComponent\s*\(\s*['"]Dialog['"]\s*\)/g],
  ['Render-function h(Dialog)', /\bh\s*\(\s*Dialog\b/g],
  ['Global Dialog registration', /\.use\(Dialog\)/g],
  ['TDesign <t-empty>', /<t-empty\b/gi],
  ['TDesign <t-loading>', /<t-loading\b/gi],
  ['Project <Modal>', /<Modal\b/g],
  ['Project <EmptyState>', /<EmptyState\b/g],
  ['Project <Loading>', /<Loading\b/g],
  ['Project <Skeleton>', /<Skeleton\b/g],
]
const stateAndOverlayUsage = stateAndOverlayPatterns.map(([label, pattern]) => ({
  label,
  total: [...contents.values()].reduce((sum, source) => sum + [...source.matchAll(pattern)].length, 0),
  files: [...contents.entries()]
    .map(([file, source]) => ({ path: relative(root, file), count: [...source.matchAll(pattern)].length }))
    .filter(entry => entry.count > 0)
    .sort((a, b) => b.count - a.count),
}))

const buttonUsage = buttonPatterns.map(([label, pattern]) => ({
  label,
  total: [...contents.values()].reduce((sum, source) => sum + [...source.matchAll(pattern)].length, 0),
  files: [...contents.entries()]
    .map(([file, source]) => ({ path: relative(root, file), count: [...source.matchAll(pattern)].length }))
    .filter(entry => entry.count > 0)
    .sort((a, b) => b.count - a.count),
}))

console.log('Frontend component inventory')
console.log('============================')
console.log(`Source files scanned: ${files.length}`)
console.log(`Vue components under src/components: ${componentFiles.length}`)
console.log('')
console.log('Component reference map (direct imports only)')
for (const file of componentFiles.sort()) {
  const users = [...references.get(file)]
  console.log(`- ${relative(root, file)}: ${users.length ? users.join(', ') : 'NO DIRECT IMPORT FOUND — verify before removal'}`)
}
console.log('')
console.log('Duplicate component filenames')
const duplicates = [...byName.entries()].filter(([, paths]) => paths.length > 1)
if (!duplicates.length) console.log('- none')
for (const [name, paths] of duplicates) console.log(`- ${name}: ${paths.join(', ')}`)
console.log('')
console.log('Global CSS import order (src/main.ts)')
for (const style of globalStyles) console.log(`- ${style}`)
console.log('')
console.log('Button implementation usage (source-level matches)')
for (const usage of buttonUsage) {
  console.log(`- ${usage.label}: ${usage.total} matches across ${usage.files.length} files`)
  for (const file of usage.files) console.log(`  - ${file.path}: ${file.count}`)
}
console.log('')
console.log('TDesign button governance gaps (review-only)')
if (!ungovernedTButtons.length) console.log('- none')
for (const button of ungovernedTButtons) console.log(`- ${button.path}:${button.line}`)
console.log('')
console.log('Icon-only button accessible-name gaps (review-only)')
if (!iconButtonsMissingName.length) console.log('- none')
for (const button of iconButtonsMissingName) console.log(`- ${button.path}:${button.line}`)
console.log('')
console.log('Form-control implementation usage (source-level matches)')
for (const usage of formControlUsage) {
  console.log(`- ${usage.label}: ${usage.total} matches across ${usage.files.length} files`)
  for (const file of usage.files) console.log(`  - ${file.path}: ${file.count}`)
}
console.log('')
console.log('')
console.log('Confirmation entrypoint usage (source-level matches)')
for (const usage of confirmationUsage) {
  console.log(`- ${usage.label}: ${usage.total} matches across ${usage.files.length} files`)
  for (const file of usage.files) console.log(`  - ${file.path}: ${file.count}`)
}
console.log('')
console.log('Modal, empty, and loading-state usage (source-level matches)')
for (const usage of stateAndOverlayUsage) {
  console.log(`- ${usage.label}: ${usage.total} matches across ${usage.files.length} files`)
  for (const file of usage.files) console.log(`  - ${file.path}: ${file.count}`)
}
console.log('')
console.log('Important: this is a review aid, not proof that a component or stylesheet is safe to delete.')
