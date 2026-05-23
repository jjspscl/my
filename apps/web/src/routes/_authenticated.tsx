import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { AppSidebar } from '@/components/layout/app-sidebar'
import { getCurrentUser } from '@/features/auth/api/auth.api'
import { authKeys } from '@/features/auth/api/auth.keys'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: async ({ context, location }) => {
    const { queryClient } = context
    try {
      await queryClient.ensureQueryData({
        queryKey: authKeys.me(),
        queryFn: getCurrentUser,
      })
    } catch {
      throw redirect({
        to: '/login',
        search: { redirect: location.pathname },
      })
    }
  },
  component: AuthenticatedLayout,
})

function AuthenticatedLayout() {
  return (
    <div className="flex min-h-screen">
      <AppSidebar />
      <main className="flex-1">
        <Outlet />
      </main>
    </div>
  )
}