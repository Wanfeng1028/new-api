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
 * BotsRouter fork: the console ships two visual identities.
 *
 * - `botsrouter` — the privatized BotsRouter theme (warm paper / ink /
 *   signal orange palette, BR Serif, capsule controls, light-only, plus
 *   the brand restyle of the pricing and rankings pages). This is the
 *   default and the only mode non-admin visitors ever see.
 * - `newapi` — the upstream New API appearance: the original default
 *   palette (light and dark), the upstream font stack, and the full
 *   upstream theme system (light / dark / system mode, color presets,
 *   sidebar variants). Only admin accounts may switch into it.
 *
 * The choice is a per-browser preference (localStorage), mirrored onto
 * `<html data-ui-mode="…">` by the boot script in index.html before first
 * paint and kept in sync by UiModeProvider. Brand identity (system name,
 * logo, favicon) stays BotsRouter in both modes — the switch covers the
 * visual theme layer only.
 */

import { readThemePreference, writeThemePreference } from '@/lib/theme-storage'

export type UiMode = 'botsrouter' | 'newapi'

export const UI_MODE_STORAGE_KEY = 'botsrouter:ui-mode:v1'

export const DEFAULT_UI_MODE: UiMode = 'botsrouter'

export const UI_MODE_VALUES: ReadonlySet<UiMode> = new Set([
  'botsrouter',
  'newapi',
])

export function readUiModePreference(): UiMode {
  return readThemePreference(
    UI_MODE_STORAGE_KEY,
    UI_MODE_VALUES,
    DEFAULT_UI_MODE
  )
}

export function writeUiModePreference(mode: UiMode): void {
  writeThemePreference(
    UI_MODE_STORAGE_KEY,
    mode === DEFAULT_UI_MODE ? null : mode
  )
}
