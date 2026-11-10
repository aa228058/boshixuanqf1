#!/usr/bin/env node
/**
 * Conservative audit for legacy success/error button classes.
 *
 * Reports CSS definitions separately from source references and highlights
 * likely dynamic class construction. This is an inventory aid, not a deletion
 * proof: runtime-generated strings and external integrations still need review.
 */
import { readdir, readFile } from 'node:fs/promises'
import { extname, join, relative, resolve } from 'node:path'

const root = resolve(process.cwd(), 'src')
const sourceExtensions = new Set(['.vue', '.ts', '.tsx', '.js', '.jsx'])
const styleExtensions = new Set(['.css', '.scss'])
const files = []

async function walk(dir) {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) await walk(full)
    else if (sourceExtensions.has(extname(entry.name)) || styleExtensions.has(extname(entry.name))) files.push(full)
  }
}

await walk(root)
const candidates = ['btn-success', 'btn-error']
const definitions = []
const sourceReferences = []
const dynamicCandidates = []
const helperReferences = []

for (const file of files) {
  const content = await readFile(file, 'utf8')
  const lines = content.split(/\r?\n/)
  const isStyle = styleExtensions.has(extname(file))
  for (let index = 0; index < lines.length; index++) {
    const line = lines[index]
    const path = relative(root, file)
    if (isStyle) {
      for (const name of candidates) {
        if (line.includes('.' + name) && line.split(/[^A-Za-z0-9_-]+/).includes(name)) {
          definitions.push({ path, line: index + 1, name, text: line.trim() })
        }
      }
    } else {
      for (const name of candidates) {
        if (line.split(/[^A-Za-z0-9_-]+/).includes(name)) {
          sourceReferences.push({ path, line: index + 1, name, text: line.trim() })
        }
      }
    }
    if (/useAsyncBtn/i.test(line)) helperReferences.push({ path, line: index + 1, text: line.trim() })
    if (/(:class\b|v-bind:class\b|classList\.(?:add|remove|toggle)\s*\(|className\s*=|setAttribute\s*\(\s*['"]class['"])/i.test(line) && /btn|success|error/i.test(line)) {
      dynamicCandidates.push({ path, line: index + 1, text: line.trim() })
    }
  }
}

console.log('Legacy button CSS reference audit')
console.log('==================================')
console.log(`Files scanned: ${files.length}`)
console.log(`CSS definitions: ${definitions.length}`)
for (const item of definitions) console.log(`- ${item.path}:${item.line} [${item.name}] ${item.text}`)
console.log('')
console.log(`Static source references: ${sourceReferences.length}`)
for (const item of sourceReferences) console.log(`- ${item.path}:${item.line} [${item.name}] ${item.text}`)
console.log('')
console.log(`useAsyncBtn references: ${helperReferences.length}`)
for (const item of helperReferences) console.log(`- ${item.path}:${item.line} ${item.text}`)
console.log('')
console.log(`Potential dynamic class expressions: ${dynamicCandidates.length}`)
for (const item of dynamicCandidates) console.log(`- ${item.path}:${item.line} ${item.text}`)
console.log('')
console.log('Review every dynamic candidate before deleting a selector. A zero static count alone is not proof of runtime non-use.')
