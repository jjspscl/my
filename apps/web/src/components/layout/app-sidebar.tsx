import { Link, useLocation } from '@tanstack/react-router'
import {
  LayoutDashboard,
  Wallet,
  CheckSquare,
  Settings,
  LogOut,
} from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/shared/lib/utils'
import { useCurrentUser, useLogout } from '@/features/auth/hooks/use-auth'

const navItems = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/finance', label: 'Finance', icon: Wallet },
  { to: '/habits', label: 'Habits', icon: CheckSquare },
  { to: '/settings', label: 'Settings', icon: Settings },
]

function SidebarContent({ onNavigate }: { onNavigate?: () => void }) {
  const location = useLocation()
  const { data: user } = useCurrentUser()
  const logout = useLogout()

  return (
    <div className="flex h-full flex-col">
      <div className="flex h-14 items-center border-b px-4">
        <Link to="/" className="text-lg font-medium tracking-tight">
          my
        </Link>
      </div>

      <nav className="flex-1 space-y-1 p-2">
        {navItems.map((item) => {
          const Icon = item.icon
          const active = location.pathname === item.to
          return (
            <Link
              key={item.to}
              to={item.to}
              onClick={onNavigate}
              className={cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors',
                active
                  ? 'bg-muted font-medium text-foreground'
                  : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
              )}
            >
              <Icon className="h-4 w-4" />
              {item.label}
            </Link>
          )
        })}
      </nav>

      <Separator />

      <div className="p-2">
        {user && (
          <div className="px-3 py-2 text-xs text-muted-foreground truncate">
            {user.email}
          </div>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="w-full justify-start gap-3 text-sm text-muted-foreground"
          onClick={() => logout.mutate()}
          disabled={logout.isPending}
        >
          <LogOut className="h-4 w-4" />
          {logout.isPending ? 'Signing out...' : 'Sign out'}
        </Button>
      </div>
    </div>
  )
}

export function AppSidebar() {
  return (
    <aside className="hidden w-56 flex-shrink-0 border-r bg-background md:flex md:flex-col">
      <SidebarContent />
    </aside>
  )
}