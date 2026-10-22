/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { Resvg } from '@resvg/resvg-js'

// Generates the raster icon set used for browser tabs and touch icons
// from the paper-plane artwork in public/logo.svg. Safari (macOS, iOS,
// iPadOS) does not render SVG favicons, so the tab icons declared in
// index.html point at this PNG/ICO set. The files are written to
// public/static/icons/ so the built app serves them under /static/,
// which stays reachable on hosts where root asset paths redirect.
//
// Run from the web/ package root:  node scripts/generate-icons.mjs

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const sourcePath = path.join(webRoot, 'public/logo.svg')
const outDir = path.join(webRoot, 'public/static/icons')

const PAPER = '#FAF7F1'
// The artwork viewBox is 280x210; centre it on a square 280x280 canvas
// at 90% scale over an opaque paper tile.
const SCALE = 0.9
const TX = (280 - 280 * SCALE) / 2
const TY = (280 - 210 * SCALE) / 2

const source = fs.readFileSync(sourcePath, 'utf8')
const inner = source
  .replace(/<svg[^>]*>/, '')
  .replace(/<\/svg>\s*$/, '')
  .trim()

const composed = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 280 280">
  <rect width="280" height="280" fill="${PAPER}"/>
  <g transform="translate(${TX} ${TY}) scale(${SCALE})">
${inner}
  </g>
</svg>`

function renderPng(size) {
  const resvg = new Resvg(composed, {
    fitTo: { mode: 'width', value: size },
    background: PAPER,
  })
  return Buffer.from(resvg.render().asPng())
}

// ICO container with PNG-compressed entries (supported since Windows
// Vista and by all browsers that read .ico files).
function buildIco(entries) {
  const header = Buffer.alloc(6)
  header.writeUInt16LE(0, 0) // reserved
  header.writeUInt16LE(1, 2) // type: icon
  header.writeUInt16LE(entries.length, 4)
  const directory = Buffer.alloc(16 * entries.length)
  let offset = 6 + directory.length
  entries.forEach((entry, i) => {
    const at = i * 16
    directory.writeUInt8(entry.size >= 256 ? 0 : entry.size, at)
    directory.writeUInt8(entry.size >= 256 ? 0 : entry.size, at + 1)
    directory.writeUInt8(0, at + 2) // palette colours
    directory.writeUInt8(0, at + 3) // reserved
    directory.writeUInt16LE(1, at + 4) // colour planes
    directory.writeUInt16LE(32, at + 6) // bits per pixel
    directory.writeUInt32LE(entry.png.length, at + 8)
    directory.writeUInt32LE(offset, at + 12)
    offset += entry.png.length
  })
  return Buffer.concat([header, directory, ...entries.map((e) => e.png)])
}

fs.mkdirSync(outDir, { recursive: true })

const targets = [
  { file: 'apple-touch-icon.png', size: 180 },
  { file: 'icon-192.png', size: 192 },
  { file: 'icon-512.png', size: 512 },
  { file: 'favicon-32.png', size: 32 },
]
for (const target of targets) {
  fs.writeFileSync(path.join(outDir, target.file), renderPng(target.size))
  console.log(`wrote ${target.file} (${target.size}x${target.size})`)
}

const ico = buildIco(
  [16, 32, 48].map((size) => ({ size, png: renderPng(size) }))
)
fs.writeFileSync(path.join(outDir, 'favicon.ico'), ico)
console.log('wrote favicon.ico (16/32/48)')
