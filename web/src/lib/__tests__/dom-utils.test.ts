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
import { beforeEach, describe, expect, test } from 'vitest'

import { applyFaviconToDom } from '../dom-utils'

function iconHrefs(rel: string): (string | null)[] {
  return [
    ...document.querySelectorAll<HTMLLinkElement>(`link[rel="${rel}"]`),
  ].map((link) => link.getAttribute('href'))
}

function seedIconLinks() {
  const icon = document.createElement('link')
  icon.rel = 'icon'
  icon.href = '/brand/logo.svg'
  document.head.appendChild(icon)
  const touch = document.createElement('link')
  touch.rel = 'apple-touch-icon'
  touch.href = '/brand/logo-email.png'
  document.head.appendChild(touch)
}

describe('applyFaviconToDom', () => {
  beforeEach(() => {
    document.head.innerHTML = ''
  })

  test('falls back to the bundled raster icons when the configured logo is an SVG', () => {
    seedIconLinks()

    applyFaviconToDom('https://dashboard.example.com/brand/logo.svg')

    expect(iconHrefs('icon')).toEqual(['/static/icons/icon-192.png'])
    expect(iconHrefs('apple-touch-icon')).toEqual([
      '/static/icons/apple-touch-icon.png',
    ])
  })

  test('detects SVG logos with query strings and uppercase extensions', () => {
    applyFaviconToDom('/assets/LOGO.SVG?v=3')

    expect(iconHrefs('icon')).toEqual(['/static/icons/icon-192.png'])
    expect(iconHrefs('apple-touch-icon')).toEqual([
      '/static/icons/apple-touch-icon.png',
    ])
  })

  test('uses the configured URL for bitmap logos', () => {
    seedIconLinks()

    applyFaviconToDom('https://cdn.example.com/logo.png')

    expect(iconHrefs('icon')).toEqual(['https://cdn.example.com/logo.png'])
    expect(iconHrefs('apple-touch-icon')).toEqual([
      'https://cdn.example.com/logo.png',
    ])
  })

  test('does nothing for an empty URL', () => {
    applyFaviconToDom('')

    expect(iconHrefs('icon')).toEqual([])
    expect(iconHrefs('apple-touch-icon')).toEqual([])
  })

  test('ignores malformed URLs without throwing', () => {
    expect(() => applyFaviconToDom('http://[')).not.toThrow()
    expect(iconHrefs('icon')).toEqual([])
  })
})
