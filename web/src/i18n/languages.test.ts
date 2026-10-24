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
import { describe, expect, test } from 'vitest'

import {
  convertDetectedLanguage,
  fromSharedLanguageCode,
  toSharedLanguageCode,
} from './languages'

describe('convertDetectedLanguage', () => {
  test('maps browser BCP-47 Chinese tags onto interface codes', () => {
    expect(convertDetectedLanguage('zh-TW')).toBe('zhTW')
    expect(convertDetectedLanguage('zh-HK')).toBe('zhTW')
    expect(convertDetectedLanguage('zh-MO')).toBe('zhTW')
    expect(convertDetectedLanguage('zh-Hant-TW')).toBe('zhTW')
    expect(convertDetectedLanguage('zh')).toBe('zhCN')
    expect(convertDetectedLanguage('zh-CN')).toBe('zhCN')
    expect(convertDetectedLanguage('zh-Hans')).toBe('zhCN')
  })

  test('keeps already-normalized interface codes stable (localStorage round-trip)', () => {
    // i18next caches `zhTW`/`zhCN` (the supportedLngs codes) to localStorage,
    // and the detector runs this converter on the cached value at every page
    // load — if `zhTW` does not survive the round-trip, a user who picked
    // Traditional Chinese is flipped to Simplified on the next load and the
    // cache is overwritten, making the flip permanent.
    expect(convertDetectedLanguage('zhTW')).toBe('zhTW')
    expect(convertDetectedLanguage('zhCN')).toBe('zhCN')
  })

  test('passes non-Chinese values through unchanged', () => {
    expect(convertDetectedLanguage('en')).toBe('en')
    expect(convertDetectedLanguage('fr-FR')).toBe('fr-FR')
    expect(convertDetectedLanguage('ja')).toBe('ja')
  })
})

describe('shared br-lang cookie mapping (review F-17)', () => {
  test('maps console codes onto the shared vocabulary', () => {
    expect(toSharedLanguageCode('zhCN')).toBe('zh')
    // The official site has no Traditional variant: closest is `zh`.
    expect(toSharedLanguageCode('zhTW')).toBe('zh')
    expect(toSharedLanguageCode('en')).toBe('en')
    // Console-only languages pass through; the official site ignores them.
    expect(toSharedLanguageCode('fr')).toBe('fr')
  })

  test('maps shared values back onto console codes', () => {
    expect(fromSharedLanguageCode('zh')).toBe('zhCN')
    expect(fromSharedLanguageCode('zhCN')).toBe('zhCN')
    expect(fromSharedLanguageCode('zhTW')).toBe('zhTW')
    expect(fromSharedLanguageCode('en')).toBe('en')
    expect(fromSharedLanguageCode('fr')).toBe('fr')
  })

  test('rejects unknown shared values so the local fallback stays in charge', () => {
    expect(fromSharedLanguageCode('xx')).toBeNull()
    expect(fromSharedLanguageCode('')).toBeNull()
    expect(fromSharedLanguageCode(null)).toBeNull()
    expect(fromSharedLanguageCode(undefined)).toBeNull()
  })
})
