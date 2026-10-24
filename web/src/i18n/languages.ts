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
export const INTERFACE_LANGUAGE_OPTIONS = [
  { code: 'zhCN', label: '简体中文' },
  { code: 'en', label: 'English' },
  { code: 'fr', label: 'Français' },
  { code: 'ru', label: 'Русский' },
  { code: 'ja', label: '日本語' },
  { code: 'vi', label: 'Tiếng Việt' },
  { code: 'zhTW', label: '繁體中文' },
] as const

export type InterfaceLanguageCode =
  (typeof INTERFACE_LANGUAGE_OPTIONS)[number]['code']

export function normalizeInterfaceLanguage(value?: string | null): string {
  if (!value) return 'en'

  let normalized = value.trim().replaceAll('_', '-').toLowerCase()
  if (
    value === 'zh-TW' ||
    value === 'zh-HK' ||
    value === 'zh-MO' ||
    value === 'zhTW'
  ) {
    normalized = 'zhTW'
  }
  if (value === 'zh-CN' || value === 'zh-Hans' || value === 'zhCN') {
    normalized = 'zhCN'
  }

  return INTERFACE_LANGUAGE_OPTIONS.some((lang) => lang.code === normalized)
    ? normalized
    : 'en'
}

/**
 * Map a browser-detected locale onto the interface language codes this project
 * uses with i18next (`zhCN` / `zhTW`).
 *
 * Browsers report standard BCP-47 tags (`zh-CN`, `zh-TW`, `zh-Hant`, `zh`, ...),
 * but `supportedLngs`/resources use the non-standard camelCase codes, so without
 * this mapping a Chinese browser would never match and fall back to English.
 * Non-Chinese codes are returned unchanged so i18next's own `supportedLngs`
 * matching still applies (e.g. `fr-FR` -> `fr`, `ja` -> `ja`).
 */
export function convertDetectedLanguage(value: string): string {
  const lower = value.trim().replaceAll('_', '-').toLowerCase()
  if (!lower.startsWith('zh')) return value
  if (
    lower === 'zhtw' ||
    lower === 'zh-tw' ||
    lower === 'zh-hk' ||
    lower === 'zh-mo' ||
    lower.startsWith('zh-hant')
  ) {
    return 'zhTW'
  }
  return 'zhCN'
}

/**
 * Cross-subdomain language sync with the official site (review F-17).
 *
 * The official site (botsrouter.botsarena.tech) and this console
 * (dashboard.botsarena.tech) cannot share localStorage, so the language
 * preference is mirrored into a `br-lang` cookie scoped to the parent
 * domain. The official site only knows `en` / `zh`; the mapping below is
 * the single source of truth for both directions:
 *
 * - console -> shared: `zhCN` -> `zh`, `zhTW` -> `zh` (the official site
 *   has no Traditional Chinese variant), every other console code is
 *   written as-is (the official site ignores values it does not know).
 * - shared -> console: `zh` -> `zhCN`; values that already are console
 *   codes pass through; anything else yields `null` (ignore the cookie
 *   and keep the existing local preference / English fallback).
 */
export const SHARED_LANGUAGE_COOKIE = 'br-lang'

export function toSharedLanguageCode(consoleCode: string): string {
  const normalized = normalizeInterfaceLanguage(consoleCode)
  if (normalized === 'zhCN' || normalized === 'zhTW') return 'zh'
  return normalized
}

export function fromSharedLanguageCode(
  value?: string | null
): InterfaceLanguageCode | null {
  if (!value) return null
  const trimmed = value.trim()
  if (trimmed === 'zh') return 'zhCN'
  return INTERFACE_LANGUAGE_OPTIONS.some((lang) => lang.code === trimmed)
    ? (trimmed as InterfaceLanguageCode)
    : null
}

export function readSharedLanguageCookie(): string | null {
  try {
    const match = document.cookie.match(
      new RegExp(`(?:^|;\\s*)${SHARED_LANGUAGE_COOKIE}=([^;]*)`)
    )
    return match ? decodeURIComponent(match[1]) : null
  } catch {
    return null
  }
}

export function writeSharedLanguageCookie(consoleCode: string): void {
  try {
    const shared = toSharedLanguageCode(consoleCode)
    const domain = window.location.hostname.endsWith('botsarena.tech')
      ? '; Domain=botsarena.tech'
      : ''
    document.cookie = `${SHARED_LANGUAGE_COOKIE}=${encodeURIComponent(shared)}; Path=/; Max-Age=31536000; SameSite=Lax; Secure${domain}`
  } catch {
    // Cookie unavailable (non-browser environment): sync is best effort.
  }
}

/**
 * Convert an interface language code (the values i18next uses, such as `zhCN` /
 * `zhTW`) into a valid BCP-47 locale tag that the `Intl.*` APIs accept.
 *
 * `new Intl.NumberFormat('zhCN')` throws `RangeError: Invalid language tag`, so
 * any locale derived from `i18n.language` / `i18n.resolvedLanguage` MUST be run
 * through this before it reaches an `Intl` constructor. Unknown values fall back
 * to `undefined`, which makes `Intl` use the runtime default locale.
 */
export function toIntlLocale(value?: string | null): string | undefined {
  if (!value) return undefined
  switch (value) {
    case 'zhCN':
      return 'zh-CN'
    case 'zhTW':
      return 'zh-TW'
    default:
      break
  }
  try {
    return Intl.getCanonicalLocales(value)[0]
  } catch {
    return undefined
  }
}
