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
import i18n from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'

import {
  convertDetectedLanguage,
  fromSharedLanguageCode,
  readSharedLanguageCookie,
  writeSharedLanguageCookie,
} from './languages'
import en from './locales/en.json'
import fr from './locales/fr.json'
import ja from './locales/ja.json'
import ru from './locales/ru.json'
import vi from './locales/vi.json'
import zhTW from './locales/zh-TW.json'
import zhCN from './locales/zh.json'

export const resources = {
  en,
  zhCN,
  fr,
  ru,
  ja,
  vi,
  zhTW,
} as const

// BotsRouter fork patch (review F-17): before i18n initialises, pull the
// shared `br-lang` cookie (written by the official site or by a previous
// console visit) into the detector's localStorage key, so a language
// chosen on either surface follows the user to the other. The boot
// script in index.html does the same before first paint; doing it here
// as well keeps the behaviour when the bundle is embedded elsewhere.
// Unknown cookie values are ignored, preserving the English fallback.
try {
  const shared = fromSharedLanguageCode(readSharedLanguageCookie())
  if (shared) {
    window.localStorage.setItem('i18nextLng', shared)
  }
} catch {
  // Storage unavailable: fall through to the normal detection order.
}

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: 'en',
    supportedLngs: ['en', 'zhCN', 'fr', 'ru', 'ja', 'vi', 'zhTW'],
    load: 'currentOnly',
    nsSeparator: false, // Allow literal colons in keys (e.g., URLs, labels)
    debug: import.meta.env.DEV,
    interpolation: {
      escapeValue: false, // not needed for react as it escapes by default
    },
    detection: {
      // BotsRouter fork patch: English is the default UI language. Only an
      // explicitly saved preference (localStorage) is honored; the browser
      // language no longer overrides the default, so fresh visitors land on
      // English via fallbackLng.
      order: ['localStorage'],
      caches: ['localStorage'],
      // Browsers report `zh-CN`/`zh-TW`/`zh`; map them onto our `zhCN`/`zhTW`
      // codes (non-Chinese codes pass through for normal supportedLngs matching).
      convertDetectedLanguage,
    },
  })

// BotsRouter fork patch (review F-17): every language change in the
// console (switcher, profile card, post-login saved preference) goes
// through i18n.changeLanguage, so mirroring it into the shared cookie
// here covers all entry points at once.
i18n.on('languageChanged', (lng) => {
  writeSharedLanguageCookie(lng)
})

export default i18n
