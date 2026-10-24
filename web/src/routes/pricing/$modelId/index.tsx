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
import { createFileRoute, redirect } from '@tanstack/react-router'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import z from 'zod'

import { useUiMode } from '@/context/ui-mode-provider'
import { ModelDetails } from '@/features/pricing/components/model-details'
import { getModuleAccessForGuard } from '@/lib/nav-modules'
import { OFFICIAL_MODEL_LIBRARY_URL } from '@/lib/official-site'
import { useAuthStore } from '@/stores/auth-store'

/**
 * Model detail pages live in the official website's model library (the
 * single maintained catalog). This route only forwards there (same tab).
 */
function ModelLibraryRedirect() {
  const { t } = useTranslation()

  useEffect(() => {
    window.location.replace(OFFICIAL_MODEL_LIBRARY_URL)
  }, [])

  return (
    <div className='text-muted-foreground flex min-h-[50vh] items-center justify-center px-6 text-sm'>
      {t('Redirecting to the model library…')}
    </div>
  )
}

const modelDetailsSearchSchema = z.object({
  search: z.string().optional(),
  sort: z.string().optional(),
  vendor: z.string().optional(),
  group: z.string().optional(),
  quotaType: z.string().optional(),
  endpointType: z.string().optional(),
  tag: z.string().optional(),
  tokenUnit: z.enum(['M', 'K']).optional(),
  view: z.enum(['card', 'table']).optional().catch(undefined),
  rechargePrice: z.boolean().optional(),
})

/**
 * BotsRouter fork patch (review F-08 residual, completed 2026-10-24):
 * like the pricing index route, the redirect to the official website
 * applies to the privatized UI mode only; the admin-only `newapi` mode
 * renders the upstream native model details page. The search schema is
 * the upstream one — without it, ModelDetails' search params are
 * untyped and lose tokenUnit / rechargePrice.
 */
function ModelDetailsRoute() {
  const { uiMode } = useUiMode()
  if (uiMode === 'newapi') {
    return <ModelDetails />
  }
  return <ModelLibraryRedirect />
}

export const Route = createFileRoute('/pricing/$modelId/')({
  validateSearch: modelDetailsSearchSchema,
  beforeLoad: async ({ context, location }) => {
    const access = await getModuleAccessForGuard(context.queryClient, 'pricing')
    if (!access.enabled) {
      throw redirect({ to: '/' })
    }
    if (access.requireAuth) {
      const { auth } = useAuthStore.getState()
      if (!auth.user) {
        throw redirect({
          to: '/sign-in',
          search: { redirect: location.href },
        })
      }
    }
  },
  component: ModelDetailsRoute,
})
