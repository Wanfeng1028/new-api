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
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ThemeProvider, useTheme } from '@/context/theme-provider'
import { UiModeProvider, useUiMode } from '@/context/ui-mode-provider'
import { ROLE } from '@/lib/roles'
import { THEME_STORAGE_KEYS } from '@/lib/theme-storage'
import { UI_MODE_STORAGE_KEY } from '@/lib/ui-mode'
import {
  useAuthStore,
  type AuthBootstrapState,
  type AuthUser,
} from '@/stores/auth-store'

function setAuth(user: AuthUser | null, bootstrapState: AuthBootstrapState) {
  useAuthStore.setState((state) => ({
    auth: { ...state.auth, user, bootstrapState },
  }))
}

const adminUser: AuthUser = { id: 1, username: 'root', role: ROLE.SUPER_ADMIN }
const regularUser: AuthUser = { id: 2, username: 'member', role: ROLE.USER }

function UiModeControls() {
  const { uiMode, canSwitchUiMode, setUiModePreference } = useUiMode()
  const theme = useTheme()

  return (
    <>
      <output aria-label='UI mode'>{uiMode}</output>
      <output aria-label='Can switch UI mode'>
        {canSwitchUiMode ? 'yes' : 'no'}
      </output>
      <output aria-label='Theme mode'>{theme.theme}</output>
      <button type='button' onClick={() => setUiModePreference('newapi')}>
        Switch to New API
      </button>
      <button type='button' onClick={() => theme.setTheme('dark')}>
        Dark theme
      </button>
    </>
  )
}

function UiModeFixture() {
  return (
    <UiModeProvider>
      <ThemeProvider>
        <UiModeControls />
      </ThemeProvider>
    </UiModeProvider>
  )
}

beforeEach(() => {
  localStorage.clear()
  setAuth(null, 'idle')
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  localStorage.clear()
  setAuth(null, 'idle')
  document.documentElement.classList.remove('light', 'dark')
  document.documentElement.removeAttribute('data-ui-mode')
})

describe('UI mode (BotsRouter fork)', () => {
  it('honours a stored newapi preference while the session bootstrap is in flight and restores the upstream theme behaviour', async () => {
    const user = userEvent.setup()
    localStorage.setItem(UI_MODE_STORAGE_KEY, 'newapi')

    render(<UiModeFixture />)

    expect(screen.getByLabelText('UI mode')).toHaveTextContent('newapi')
    expect(document.documentElement).toHaveAttribute('data-ui-mode', 'newapi')
    expect(screen.getByLabelText('Theme mode')).toHaveTextContent('system')

    await user.click(screen.getByRole('button', { name: 'Dark theme' }))

    expect(screen.getByLabelText('Theme mode')).toHaveTextContent('dark')
    expect(document.documentElement).toHaveClass('dark')
    expect(localStorage.getItem(THEME_STORAGE_KEYS.mode)).toBe('dark')
  })

  it('forces the botsrouter mode for signed-in non-admin users once the bootstrap completes', async () => {
    const user = userEvent.setup()
    localStorage.setItem(UI_MODE_STORAGE_KEY, 'newapi')
    setAuth(regularUser, 'complete')

    render(<UiModeFixture />)

    expect(screen.getByLabelText('UI mode')).toHaveTextContent('botsrouter')
    expect(screen.getByLabelText('Can switch UI mode')).toHaveTextContent('no')
    expect(document.documentElement).toHaveAttribute(
      'data-ui-mode',
      'botsrouter'
    )
    // The privatized mode stays light-only.
    expect(screen.getByLabelText('Theme mode')).toHaveTextContent('light')
    expect(document.documentElement).toHaveClass('light')

    await user.click(screen.getByRole('button', { name: 'Dark theme' }))

    expect(screen.getByLabelText('Theme mode')).toHaveTextContent('light')
    expect(document.documentElement).not.toHaveClass('dark')

    await user.click(screen.getByRole('button', { name: 'Switch to New API' }))

    expect(screen.getByLabelText('UI mode')).toHaveTextContent('botsrouter')
  })

  it('lets an admin keep the newapi mode after the bootstrap completes', async () => {
    const user = userEvent.setup()
    localStorage.setItem(UI_MODE_STORAGE_KEY, 'newapi')
    setAuth(adminUser, 'complete')

    render(<UiModeFixture />)

    expect(screen.getByLabelText('UI mode')).toHaveTextContent('newapi')
    expect(screen.getByLabelText('Can switch UI mode')).toHaveTextContent('yes')

    await user.click(screen.getByRole('button', { name: 'Dark theme' }))

    expect(screen.getByLabelText('Theme mode')).toHaveTextContent('dark')
    expect(localStorage.getItem(THEME_STORAGE_KEYS.mode)).toBe('dark')
  })

  it('lets an admin switch from the default botsrouter mode into newapi and back via storage', async () => {
    const user = userEvent.setup()
    setAuth(adminUser, 'complete')

    render(<UiModeFixture />)

    expect(screen.getByLabelText('UI mode')).toHaveTextContent('botsrouter')
    expect(localStorage.getItem(UI_MODE_STORAGE_KEY)).toBeNull()

    await user.click(screen.getByRole('button', { name: 'Switch to New API' }))

    expect(screen.getByLabelText('UI mode')).toHaveTextContent('newapi')
    expect(document.documentElement).toHaveAttribute('data-ui-mode', 'newapi')
    expect(localStorage.getItem(UI_MODE_STORAGE_KEY)).toBe('newapi')
  })
})
