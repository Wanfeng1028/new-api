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

/**
 * Canonical URLs of the BotsRouter official website. The console links to
 * the site (and to its model library, which is the single maintained model
 * catalog) instead of hosting a second copy of the catalog pages.
 */
export const OFFICIAL_SITE_URL = 'https://botsrouter.botsarena.tech'

/** The official website's model library (HashRouter route). */
export const OFFICIAL_MODEL_LIBRARY_URL = `${OFFICIAL_SITE_URL}/#/models`

/**
 * BotsRouter fork (version-sync final fix): where the top navigation's
 * "Model Library" entry points depends on the effective UI mode, and
 * nothing else. Previously it always pointed at the official website,
 * so switching the console to `newapi` mode never changed the model
 * library the user landed on — the route-level split in
 * routes/pricing was bypassed by the nav link itself.
 *
 * - `botsrouter` (privatized): the official website's model library,
 *   opened in the current tab (external to the console app).
 * - `newapi`: the console's own `/pricing` route, which renders the
 *   upstream native pricing page in that mode.
 */
export function resolveModelLibraryLink(mode: 'botsrouter' | 'newapi'): {
  href: string
  external: boolean
} {
  if (mode === 'newapi') {
    return { href: '/pricing', external: false }
  }
  return { href: OFFICIAL_MODEL_LIBRARY_URL, external: true }
}
