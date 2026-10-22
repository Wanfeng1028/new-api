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

import { ROLE } from '@/lib/roles'
import {
  DEFAULT_UI_MODE,
  readUiModePreference,
  writeUiModePreference,
  type UiMode,
} from '@/lib/ui-mode'
import { useAuthStore } from '@/stores/auth-store'

type UiModeContextType = {
  /** The mode actually rendered right now (admin gate applied). */
  uiMode: UiMode
  /** The admin's saved choice, before the admin gate is applied. */
  uiModePreference: UiMode
  /** True when the signed-in user may switch modes (admin accounts). */
  canSwitchUiMode: boolean
  setUiModePreference: (mode: UiMode) => void
  resetUiModePreference: () => void
}

// Fallback used when a consumer renders outside the provider (e.g. tests
// or an error route mounted before providers are ready): behave as the
// default privatized theme and never offer the switch.
const FALLBACK_CONTEXT: UiModeContextType = {
  uiMode: DEFAULT_UI_MODE,
  uiModePreference: DEFAULT_UI_MODE,
  canSwitchUiMode: false,
  setUiModePreference: () => {},
  resetUiModePreference: () => {},
}

const UiModeContext = createContext<UiModeContextType>(FALLBACK_CONTEXT)

export function UiModeProvider(props: { children: React.ReactNode }) {
  const [preference, _setPreference] = useState<UiMode>(() =>
    readUiModePreference()
  )
  const user = useAuthStore((state) => state.auth.user)
  const bootstrapState = useAuthStore((state) => state.auth.bootstrapState)

  const isAdmin = !!user && user.role >= ROLE.ADMIN
  const canSwitchUiMode = isAdmin

  // Admin gate: once the session bootstrap has settled, anyone who is
  // not an admin (including signed-out visitors) always gets the
  // privatized theme, whatever preference is stored in this browser.
  // While the bootstrap is still in flight we honour the stored
  // preference so an admin does not see the theme flip on every load.
  const uiMode: UiMode =
    bootstrapState === 'complete' && !isAdmin ? DEFAULT_UI_MODE : preference

  useEffect(() => {
    if (typeof document === 'undefined') return
    document.documentElement.setAttribute('data-ui-mode', uiMode)
  }, [uiMode])

  const setUiModePreference = useCallback(
    (mode: UiMode) => {
      if (bootstrapState === 'complete' && !isAdmin) return
      _setPreference(mode)
      writeUiModePreference(mode)
    },
    [bootstrapState, isAdmin]
  )

  const resetUiModePreference = useCallback(() => {
    _setPreference(DEFAULT_UI_MODE)
    writeUiModePreference(DEFAULT_UI_MODE)
  }, [])

  const value = useMemo<UiModeContextType>(
    () => ({
      uiMode,
      uiModePreference: preference,
      canSwitchUiMode,
      setUiModePreference,
      resetUiModePreference,
    }),
    [
      uiMode,
      preference,
      canSwitchUiMode,
      setUiModePreference,
      resetUiModePreference,
    ]
  )

  return <UiModeContext value={value}>{props.children}</UiModeContext>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useUiMode() {
  return useContext(UiModeContext)
}
