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

import { getModuleAccessForGuard } from '@/lib/nav-modules'
import { OFFICIAL_MODEL_LIBRARY_URL } from '@/lib/official-site'
import { useAuthStore } from '@/stores/auth-store'

/**
 * The model library is maintained in exactly one place: the official
 * website. This route only forwards there (same tab), so old console
 * bookmarks and deep links keep working without a second catalog copy.
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

export const Route = createFileRoute('/pricing/')({
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
  component: ModelLibraryRedirect,
})
