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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { useUiMode } from '@/context/ui-mode-provider'
import { useStatus } from '@/hooks/use-status'
import { parseHeaderNavModulesFromStatus } from '@/lib/nav-modules'
import {
  OFFICIAL_SITE_URL,
  resolveModelLibraryLink,
} from '@/lib/official-site'
import { useAuthStore } from '@/stores/auth-store'

export type TopNavLink = {
  title: string
  href: string
  disabled?: boolean
  requiresAuth?: boolean
  external?: boolean
  /** External links open in a new tab by default; sameTab forces the current tab. */
  sameTab?: boolean
}

/**
 * Generate top navigation links based on HeaderNavModules configuration from backend /api/status
 * Backend format example (stringified JSON):
 * {
 *   home: true,
 *   console: true,
 *   pricing: { enabled: true, requireAuth: false },
 *   rankings: { enabled: true, requireAuth: false },
 *   docs: true,
 *   about: true
 * }
 */
export function useTopNavLinks(): TopNavLink[] {
  const { t } = useTranslation()
  const { status } = useStatus()
  const { auth } = useAuthStore()
  const { uiMode } = useUiMode()

  // Parse HeaderNavModules
  const modules = useMemo(() => {
    return parseHeaderNavModulesFromStatus(
      status as Record<string, unknown> | null
    )
  }, [status])

  // Documentation link (may be external)
  const docsLink: string | undefined = status?.docs_link as string | undefined

  const isAuthed = !!auth?.user

  const links: TopNavLink[] = []

  // Home (official website; navigates in the current tab)
  if (modules?.home !== false) {
    links.push({
      title: t('Home'),
      href: OFFICIAL_SITE_URL,
      external: true,
      sameTab: true,
    })
  }

  // Console -> /dashboard (new console path)
  if (modules?.console !== false) {
    links.push({ title: t('Console'), href: '/dashboard' })
  }

  // Model Library: follows the effective UI mode (version-sync final
  // fix). In `botsrouter` mode the single maintained catalog lives on
  // the official website and this entry jumps there in the current tab;
  // in `newapi` mode it stays inside the console on /pricing, which
  // renders the upstream native pricing page. Pointing here at the
  // website unconditionally used to bypass the route-level split.
  const pricing = modules?.pricing
  if (pricing && typeof pricing === 'object' && pricing.enabled) {
    const requiresAuth = pricing.requireAuth && !isAuthed
    const modelLibrary = resolveModelLibraryLink(uiMode)
    links.push({
      title: t('Model Library'),
      href: modelLibrary.href,
      external: modelLibrary.external,
      sameTab: true,
      requiresAuth,
    })
  }

  // Rankings
  const rankings = modules?.rankings
  if (rankings && typeof rankings === 'object' && rankings.enabled) {
    const requiresAuth = rankings.requireAuth && !isAuthed
    links.push({ title: t('Rankings'), href: '/rankings', requiresAuth })
  }

  // Docs (supports external links). A docs link that points at the
  // official website stays in the current tab like every other
  // in-site route; a genuinely external docs host keeps the new tab.
  if (modules?.docs !== false) {
    if (docsLink) {
      const docsOnOfficialSite = docsLink.startsWith(OFFICIAL_SITE_URL)
      links.push({
        title: t('Docs'),
        href: docsLink,
        external: true,
        sameTab: docsOnOfficialSite ? true : undefined,
      })
    } else {
      links.push({ title: t('Docs'), href: '/docs' })
    }
  }

  // About. BotsRouter fork patch: the panel's built-in about page is
  // upstream project content, so this links to the official website's
  // about section instead.
  if (modules?.about !== false) {
    links.push({
      title: t('About'),
      href: `${OFFICIAL_SITE_URL}/#/about`,
      external: true,
      sameTab: true,
    })
  }

  return links
}
