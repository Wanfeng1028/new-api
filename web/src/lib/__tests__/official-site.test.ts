/*
Copyright (C) 2023-2026 QuantumNous

This program is free software; you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 or (at your
option) any later version.
*/

import { describe, expect, it } from 'vitest'

import {
  OFFICIAL_MODEL_LIBRARY_URL,
  OFFICIAL_SITE_URL,
  resolveModelLibraryLink,
} from '@/lib/official-site'

describe('resolveModelLibraryLink (version-sync final fix)', () => {
  it('points the top-nav model library at the official website in botsrouter mode', () => {
    expect(resolveModelLibraryLink('botsrouter')).toEqual({
      href: OFFICIAL_MODEL_LIBRARY_URL,
      external: true,
    })
    expect(OFFICIAL_MODEL_LIBRARY_URL.startsWith(OFFICIAL_SITE_URL)).toBe(
      true
    )
  })

  it('points the top-nav model library at the console /pricing route in newapi mode', () => {
    expect(resolveModelLibraryLink('newapi')).toEqual({
      href: '/pricing',
      external: false,
    })
  })
})
