/*
Copyright (C) 2023-2026 QuantumNous

This program is free software; you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 or (at your
option) any later version.
*/

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { VersionedPage } from '@/components/versioned-page'
import { UiModeProvider } from '@/context/ui-mode-provider'
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

function VersionedFixture() {
  return (
    <UiModeProvider>
      <VersionedPage
        botsrouter={<p>BotsRouter branch</p>}
        newapi={<p>New API branch</p>}
      />
    </UiModeProvider>
  )
}

beforeEach(() => {
  localStorage.clear()
  setAuth(null, 'idle')
})

afterEach(() => {
  cleanup()
  localStorage.clear()
  setAuth(null, 'idle')
  document.documentElement.removeAttribute('data-ui-mode')
})

describe('VersionedPage (version-sync final fix)', () => {
  it('renders the botsrouter branch and marks it when no preference is stored', () => {
    render(<VersionedFixture />)

    expect(screen.getByText('BotsRouter branch')).toBeInTheDocument()
    expect(screen.queryByText('New API branch')).not.toBeInTheDocument()
    expect(
      document.querySelector('[data-version-branch]')
    ).toHaveAttribute('data-version-branch', 'botsrouter')
  })

  it('renders the newapi branch and marks it when a newapi preference is stored', () => {
    localStorage.setItem(UI_MODE_STORAGE_KEY, 'newapi')
    render(<VersionedFixture />)

    expect(screen.getByText('New API branch')).toBeInTheDocument()
    expect(screen.queryByText('BotsRouter branch')).not.toBeInTheDocument()
    expect(
      document.querySelector('[data-version-branch]')
    ).toHaveAttribute('data-version-branch', 'newapi')
  })

  it('falls back to the botsrouter branch outside a provider', () => {
    render(
      <VersionedPage
        botsrouter={<p>BotsRouter branch</p>}
        newapi={<p>New API branch</p>}
      />
    )

    expect(screen.getByText('BotsRouter branch')).toBeInTheDocument()
    expect(
      document.querySelector('[data-version-branch]')
    ).toHaveAttribute('data-version-branch', 'botsrouter')
  })
})
