/*
Copyright (C) 2023-2026 QuantumNous

This program is free software; you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 or (at your
option) any later version.
*/

import type { ReactNode } from 'react'

import { useUiMode } from '@/context/ui-mode-provider'

/**
 * BotsRouter fork: the single routing switch for the three catalog pages
 * (model library, rankings, pricing). Every catalog route renders through
 * this shell so the branch it shows is derived from exactly one source —
 * `UiModeProvider`'s effective `uiMode` — and never from a locally
 * hardcoded component choice. A route that bypasses this shell and renders
 * a fixed variant is a version-sync bug by construction.
 *
 * The `data-version-branch` marker exists for the regression matrix: it
 * records which branch actually rendered, so tests and the live bundle
 * can prove the switch instead of trusting the route code.
 *
 * Rankings is intentionally unified: both branches render the same merged
 * rankings page (see features/rankings). A theme switch must never drop
 * or swap rankings sections, so there is no second rankings variant to
 * branch to — the shell still wraps it so a future divergence has to go
 * through this one place.
 */
export function VersionedPage(props: {
  botsrouter: ReactNode
  newapi: ReactNode
}) {
  const { uiMode } = useUiMode()
  return (
    <div data-version-branch={uiMode} className='contents'>
      {uiMode === 'newapi' ? props.newapi : props.botsrouter}
    </div>
  )
}
