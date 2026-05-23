import { WifiOff, RefreshCw, AlertCircle } from 'lucide-react'
import { useNetworkStatus } from '@/shared/sync/network-status'
import { useSyncState, drainQueue } from '@/shared/sync/sync-engine'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/components/ui/button'

export function SyncStatus() {
  const isOnline = useNetworkStatus((s) => s.isOnline)
  const { status, pendingCount } = useSyncState()

  // Don't show anything when fully synced and online
  if (isOnline && status === 'idle' && pendingCount === 0) return null

  return (
    <div className={cn(
      'fixed bottom-16 left-2 right-2 z-40 md:bottom-2 md:left-auto md:right-4 md:w-auto',
      'flex items-center gap-2 rounded-lg border px-3 py-2 text-xs shadow-sm',
      !isOnline && 'border-amber-500/50 bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-200',
      isOnline && status === 'syncing' && 'border-blue-500/50 bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-200',
      isOnline && status === 'error' && 'border-destructive/50 bg-destructive/5 text-destructive',
    )}>
      {!isOnline && (
        <>
          <WifiOff className="h-3.5 w-3.5" />
          <span>Offline{pendingCount > 0 && ` · ${pendingCount} pending`}</span>
        </>
      )}
      {isOnline && status === 'syncing' && (
        <>
          <RefreshCw className="h-3.5 w-3.5 animate-spin" />
          <span>Syncing {pendingCount} change{pendingCount !== 1 ? 's' : ''}...</span>
        </>
      )}
      {isOnline && status === 'error' && (
        <>
          <AlertCircle className="h-3.5 w-3.5" />
          <span>{pendingCount} failed to sync</span>
          <Button variant="ghost" size="sm" className="h-5 px-1.5 text-xs" onClick={() => drainQueue()}>
            Retry
          </Button>
        </>
      )}
    </div>
  )
}