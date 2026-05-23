import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { AppSidebar } from '@/components/layout/app-sidebar'
import { BottomNav } from '@/components/layout/bottom-nav'
import { SyncStatus } from '@/components/layout/sync-status'
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
  pendingComponent: PageSkeleton,
  errorComponent: PageError,
})

function AuthenticatedLayout() {
  return (
    <div className="flex min-h-screen">
      <AppSidebar />
      <main className="flex-1 pb-16 md:pb-0">
        <Outlet />
      </main>
      <SyncStatus />
      <BottomNav />
    </div>
  )
}

function PageSkeleton() {
  return (
    <div className="flex min-h-screen">
      <AppSidebar />
      <main className="flex-1 p-4">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="h-40 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      </main>
    </div>
  )
}

function PageError({ error }: { error: Error }) {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center space-y-2">
        <p className="text-sm text-muted-foreground">Something went wrong</p>
        <p className="text-xs text-destructive">{error.message}</p>
      </div>
    </div>
  )
}