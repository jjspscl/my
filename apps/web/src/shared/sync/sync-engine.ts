import { getAll, dequeue, incrementRetry, type QueuedMutation } from './mutation-queue'
import { useNetworkStatus } from './network-status'
import { create } from 'zustand'

type SyncStatus = 'idle' | 'syncing' | 'error'

interface SyncState {
  status: SyncStatus
  pendingCount: number
  lastSyncAt: string | null
  setStatus: (s: SyncStatus) => void
  setPendingCount: (n: number) => void
  setLastSyncAt: (d: string) => void
}

export const useSyncState = create<SyncState>((set) => ({
  status: 'idle',
  pendingCount: 0,
  lastSyncAt: null,
  setStatus: (status) => set({ status }),
  setPendingCount: (pendingCount) => set({ pendingCount }),
  setLastSyncAt: (lastSyncAt) => set({ lastSyncAt }),
}))

function getCSRFToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)my_csrf=([^;]*)/)
  return match?.[1] ? decodeURIComponent(match[1]) : ''
}

async function replayMutation(m: QueuedMutation): Promise<boolean> {
  try {
    const headers: Record<string, string> = {
      'X-CSRF-Token': getCSRFToken(),
    }
    if (m.body) {
      headers['Content-Type'] = 'application/json'
    }
    const res = await fetch(m.url, {
      method: m.method,
      headers,
      body: m.body,
      credentials: 'include',
    })
    // 2xx = success, 4xx = permanent failure (don't retry), 5xx = retry
    if (res.ok) return true
    if (res.status >= 400 && res.status < 500) {
      // Permanent failure — discard
      console.warn(`[sync] Discarding mutation ${m.id}: ${res.status}`)
      return true // remove from queue
    }
    return false // 5xx, retry later
  } catch {
    return false // network error, retry later
  }
}

export async function drainQueue(): Promise<void> {
  const { setStatus, setPendingCount, setLastSyncAt } = useSyncState.getState()
  const isOnline = useNetworkStatus.getState().isOnline
  if (!isOnline) return

  const pending = await getAll()
  if (pending.length === 0) {
    setPendingCount(0)
    return
  }

  setStatus('syncing')
  setPendingCount(pending.length)

  for (const mutation of pending) {
    const success = await replayMutation(mutation)
    if (success) {
      await dequeue(mutation.id)
      setPendingCount(Math.max(0, useSyncState.getState().pendingCount - 1))
    } else {
      const updated = await incrementRetry(mutation.id)
      if (!updated) {
        // Max retries exceeded, already removed
        setPendingCount(Math.max(0, useSyncState.getState().pendingCount - 1))
      }
    }
  }

  const remaining = await getAll()
  setPendingCount(remaining.length)
  setStatus(remaining.length > 0 ? 'error' : 'idle')
  if (remaining.length === 0) {
    setLastSyncAt(new Date().toISOString())
  }
}

let drainInterval: ReturnType<typeof setInterval> | null = null

export function startSyncEngine(): () => void {
  // Drain on startup
  drainQueue()

  // Drain when coming back online
  const unsub = useNetworkStatus.subscribe((state, prev) => {
    if (state.isOnline && !prev.isOnline) {
      drainQueue()
    }
  })

  // Periodic drain every 30s
  drainInterval = setInterval(drainQueue, 30_000)

  return () => {
    unsub()
    if (drainInterval) clearInterval(drainInterval)
  }
}