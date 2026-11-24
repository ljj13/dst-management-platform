#!/usr/bin/env node
// 跨平台目录同步：清空目标目录后完整复制源目录。
// 用法：node scripts/copy-dir.mjs <src> <dst>
import { cpSync, existsSync, mkdirSync, rmSync, statSync } from 'node:fs'
import { resolve } from 'node:path'

const [srcArg, dstArg] = process.argv.slice(2)

if (!srcArg || !dstArg) {
  console.error('usage: node scripts/copy-dir.mjs <src> <dst>')
  process.exit(2)
}

const src = resolve(srcArg)
const dst = resolve(dstArg)

if (!existsSync(src) || !statSync(src).isDirectory()) {
  console.error(`source directory not found: ${src}`)
  process.exit(2)
}

rmSync(dst, { recursive: true, force: true })
mkdirSync(dst, { recursive: true })
cpSync(src, dst, { recursive: true, force: true })
console.log(`copied ${src} -> ${dst}`)
