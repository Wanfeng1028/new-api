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
import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { SignIn } from '@/features/auth/sign-in'

const searchSchema = z.object({
  redirect: z.string().optional(),
})

export const Route = createFileRoute('/(auth)/sign-in')({
  component: SignIn,
  validateSearch: searchSchema,
  beforeLoad: async () => {
    // BotsRouter fork patch: the panel's built-in sign-in page is retired.
    // Every unauthenticated flow lands here — sign-out, the http client's
    // session-expiry replace, legacy /login links — so send the browser
    // to the official website login with a full-page navigation instead
    // of rendering the built-in form. (Local 127.0.0.1 acceptance hits the
    // same jump; the website login bridges back into this console under
    // the same account system.) OAuth callbacks live under /oauth/* and
    // are unaffected.
    window.location.replace('https://botsrouter.botsarena.tech/#/login')
    // The browser is leaving; keep this route pending so the built-in
    // form never mounts while the navigation is in flight.
    await new Promise<never>(() => {})
  },
})
