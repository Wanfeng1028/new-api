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
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react'

import {
  readThemePreference,
  THEME_STORAGE_KEYS,
  writeThemePreference,
} from '@/lib/theme-storage'

import { useUiMode } from './ui-mode-provider'

type Theme = 'dark' | 'light' | 'system'
type ResolvedTheme = Exclude<Theme, 'system'>

// BotsRouter fork: two UI modes (see lib/ui-mode.ts). In the default
// `botsrouter` mode the console is light-only, as before. In the
// admin-only `newapi` mode the upstream behaviour applies in full:
// the default theme is `system` and light / dark / system preferences
// are stored and honoured.
const DEFAULT_THEME: Theme = 'system'
const THEMES = new Set<Theme>(['dark', 'light', 'system'])

type ThemeProviderProps = {
  children: React.ReactNode
  defaultTheme?: Theme
  storageKey?: string
}

type ThemeProviderState = {
  defaultTheme: Theme
  resolvedTheme: ResolvedTheme
  theme: Theme
  setTheme: (theme: Theme) => void
  resetTheme: () => void
}

const initialState: ThemeProviderState = {
  defaultTheme: DEFAULT_THEME,
  resolvedTheme: 'light',
  theme: DEFAULT_THEME,
  setTheme: () => null,
  resetTheme: () => null,
}

const ThemeContext = createContext<ThemeProviderState>(initialState)

function getSystemTheme(): ResolvedTheme {
  if (typeof window === 'undefined') return 'light'
  return window.matchMedia('(prefers-color-scheme: dark)').matches
    ? 'dark'
    : 'light'
}

function resolveTheme(theme: Theme): ResolvedTheme {
  return theme === 'system' ? getSystemTheme() : theme
}

export function ThemeProvider({
  children,
  defaultTheme = DEFAULT_THEME,
  storageKey = THEME_STORAGE_KEYS.mode,
  ...props
}: ThemeProviderProps) {
  const { uiMode } = useUiMode()
  const privatized = uiMode === 'botsrouter'

  const [theme, _setTheme] = useState<Theme>(() =>
    readThemePreference(storageKey, THEMES, defaultTheme)
  )
  const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>(() =>
    privatized ? 'light' : resolveTheme(theme)
  )

  // When an admin switches into the New API mode, re-read the stored
  // mode preference: writes made while privatized were coerced to
  // 'light', and a preference saved during an earlier New API session
  // must take effect again.
  useEffect(() => {
    if (!privatized) {
      _setTheme(readThemePreference(storageKey, THEMES, defaultTheme))
    }
  }, [privatized, storageKey, defaultTheme])

  useEffect(() => {
    const root = window.document.documentElement

    if (privatized) {
      // BotsRouter mode: pin every surface to light.
      root.classList.remove('dark')
      root.classList.add('light')
      setResolvedTheme('light')
      return
    }

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    const applyTheme = () => {
      const nextResolvedTheme = resolveTheme(theme)
      root.classList.remove('light', 'dark')
      root.classList.add(nextResolvedTheme)
      setResolvedTheme(nextResolvedTheme)
    }

    applyTheme()
    mediaQuery.addEventListener('change', applyTheme)
    return () => mediaQuery.removeEventListener('change', applyTheme)
  }, [privatized, theme])

  const setTheme = useCallback(
    (nextTheme: Theme) => {
      if (privatized) {
        // Light-only: coerce and persist 'light' so a stale dark
        // preference normalizes instead of resurfacing.
        writeThemePreference(storageKey, 'light')
        return
      }
      writeThemePreference(storageKey, nextTheme)
      _setTheme(nextTheme)
    },
    [privatized, storageKey]
  )

  const resetTheme = useCallback(() => {
    if (privatized) {
      writeThemePreference(storageKey, 'light')
      return
    }
    writeThemePreference(storageKey, null)
    _setTheme(defaultTheme)
  }, [privatized, storageKey, defaultTheme])

  const contextValue = useMemo(
    () => ({
      defaultTheme,
      resolvedTheme: privatized ? ('light' as ResolvedTheme) : resolvedTheme,
      resetTheme,
      theme: privatized ? ('light' as Theme) : theme,
      setTheme,
    }),
    [defaultTheme, privatized, resolvedTheme, resetTheme, theme, setTheme]
  )

  return (
    <ThemeContext value={contextValue} {...props}>
      {children}
    </ThemeContext>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export const useTheme = () => {
  const context = useContext(ThemeContext)

  if (!context) throw new Error('useTheme must be used within a ThemeProvider')

  return context
}
