import { Link, useLocation } from '@tanstack/react-router'
import { LayoutDashboard, Wallet, CheckSquare, Settings } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

const navItems = [
  { to: '/' as const, label: 'Home', icon: LayoutDashboard },
  { to: '/finance' as const, label: 'Finance', icon: Wallet },
  { to: '/habits' as const, label: 'Habits', icon: CheckSquare },
  { to: '/settings' as const, label: 'Settings', icon: Settings },
]

export function BottomNav() {
  const location = useLocation()

  return (
    <nav className="fixed inset-x-0 bottom-0 z-50 border-t bg-background md:hidden">
      <div className="flex h-14 items-center justify-around">
        {navItems.map((item) => {
          const Icon = item.icon
          const active = location.pathname === item.to
          return (
            <Link
              key={item.to}
              to={item.to}
              className={cn(
                'flex flex-col items-center gap-0.5 px-3 py-1 text-xs transition-colors',
                active
                  ? 'text-foreground'
                  : 'text-muted-foreground',
              )}
            >
              <Icon className={cn('h-5 w-5', active && 'text-foreground')} />
              <span>{item.label}</span>
            </Link>
          )
        })}
      </div>
    </nav>
  )
}