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
// Bundled raster tab icons (see scripts/generate-icons.mjs). They are
// served from /static/icons/, which stays reachable on every console
// host, and unlike SVG artwork they also render in Safari.
const BUNDLED_FAVICON_URL = '/static/icons/icon-192.png'
const BUNDLED_TOUCH_ICON_URL = '/static/icons/apple-touch-icon.png'

function setIconLink(rel: string, href: string) {
  const existing = document.querySelectorAll<HTMLLinkElement>(
    `link[rel="${rel}"]`
  )
  if (existing.length === 1 && existing[0].getAttribute('href') === href) return
  const link = document.createElement('link')
  link.rel = rel
  link.href = href
  existing.forEach((l) => l.remove())
  document.head.appendChild(link)
}

export function applyFaviconToDom(url: string) {
  if (typeof document === 'undefined' || !url) return
  try {
    const { pathname } = new URL(url, window.location.href)
    // Safari does not render SVG favicons: when the configured logo is
    // an SVG, it stays as the in-page logo, but the tab icons fall back
    // to the bundled raster set. Bitmap logos are used as configured.
    const isSvg = pathname.toLowerCase().endsWith('.svg')
    setIconLink('icon', isSvg ? BUNDLED_FAVICON_URL : url)
    setIconLink('apple-touch-icon', isSvg ? BUNDLED_TOUCH_ICON_URL : url)
  } catch {
    // Ignore malformed URLs
  }
}
