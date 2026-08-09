import { useState } from 'react'
import { WifiOff, RefreshCw, AlertCircle, ChevronDown, Trash2, RotateCcw } from 'lucide-react'
import { useNetworkStatus } from '@/shared/sync/network-status'
import {
  useSyncState,
  drainQueue,
  retryFailed,
  discardFailed,
  discardCorruptAll,
} from '@/shared/sync/sync-engine'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/components/ui/button'

export function SyncStatus() {
  const isOnline = useNetworkStatus((s) => s.isOnline)
  const { status, pendingCount, failedCount, corruptCount, failedItems } = useSyncState()
  const [showFailed, setShowFailed] = useState(false)

  // Don't show anything when fully synced and online
  if (isOnline && status === 'idle' && pendingCount === 0 && failedCount === 0 && corruptCount === 0) {
    return null
  }

  const hasDeadLetters = failedCount > 0 || corruptCount > 0

  return (
    <div
      className={cn(
        'fixed bottom-16 left-2 right-2 z-40 md:bottom-2 md:left-auto md:right-4 md:w-auto',
        'rounded-lg border px-3 py-2 text-xs shadow-sm',
        !isOnline && 'border-amber-500/50 bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-200',
        isOnline && status === 'syncing' && 'border-blue-500/50 bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-200',
        isOnline && hasDeadLetters && 'border-destructive/50 bg-destructive/5 text-destructive',
        isOnline && !hasDeadLetters && status === 'error' && 'border-destructive/50 bg-destructive/5 text-destructive',
      )}
    >
      {!isOnline && (
        <div className="flex items-center gap-2">
          <WifiOff className="h-3.5 w-3.5" />
          <span>Offline{pendingCount > 0 && ` · ${pendingCount} pending`}</span>
        </div>
      )}
      {isOnline && status === 'syncing' && (
        <div className="flex items-center gap-2">
          <RefreshCw className="h-3.5 w-3.5 animate-spin" />
          <span>Syncing {pendingCount} change{pendingCount !== 1 ? 's' : ''}...</span>
        </div>
      )}
      {isOnline && hasDeadLetters && (
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-3.5 w-3.5" />
            <span>
              {failedCount > 0 && `${failedCount} change${failedCount !== 1 ? 's' : ''} failed`}
              {failedCount > 0 && corruptCount > 0 && ' · '}
              {corruptCount > 0 && `${corruptCount} unreadable`}
            </span>
            <Button variant="ghost" size="sm" className="h-5 px-1.5 text-xs" onClick={() => drainQueue()}>
              Retry
            </Button>
            <button
              type="button"
              aria-label={showFailed ? 'Hide failed changes' : 'Show failed changes'}
              className="text-muted-foreground hover:text-foreground"
              onClick={() => setShowFailed((v) => !v)}
            >
              <ChevronDown className={cn('h-3.5 w-3.5 transition-transform', showFailed && 'rotate-180')} />
            </button>
          </div>
          {showFailed && (
            <ul className="space-y-1 pt-1">
              {failedItems.map((item) => (
                <li key={item.id} className="flex items-center gap-2 rounded border border-destructive/20 px-2 py-1">
                  <span className="min-w-0 flex-1 truncate">
                    <span className="font-medium">{item.method}</span> {item.url}
                    <span className="text-muted-foreground"> — {item.failedReason}</span>
                  </span>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-5 px-1.5"
                    title="Retry this change"
                    onClick={() => retryFailed(item.id)}
                  >
                    <RotateCcw className="h-3 w-3" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-5 px-1.5 text-destructive"
                    title="Discard this change"
                    onClick={() => discardFailed(item.id)}
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </li>
              ))}
              {corruptCount > 0 && (
                <li className="flex items-center gap-2 rounded border border-destructive/20 px-2 py-1">
                  <span className="min-w-0 flex-1">
                    {corruptCount} unreadable offline change{corruptCount !== 1 ? 's' : ''} (could not be
                    replayed)
                  </span>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-5 px-1.5 text-destructive"
                    title="Discard unreadable changes"
                    onClick={() => discardCorruptAll()}
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </li>
              )}
            </ul>
          )}
        </div>
      )}
      {isOnline && !hasDeadLetters && status === 'error' && (
        <div className="flex items-center gap-2">
          <AlertCircle className="h-3.5 w-3.5" />
          <span>{pendingCount} failed to sync</span>
          <Button variant="ghost" size="sm" className="h-5 px-1.5 text-xs" onClick={() => drainQueue()}>
            Retry
          </Button>
        </div>
      )}
    </div>
  )
}
