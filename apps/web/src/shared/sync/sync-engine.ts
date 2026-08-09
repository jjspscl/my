import {
  getAll,
  getAllFailed,
  getCorruptCount,
  dequeue,
  markFailed,
  markPending,
  discardCorrupt,
  incrementRetry,
  type FailedMutation,
  type QueuedMutation,
} from './mutation-queue'
import { useNetworkStatus } from './network-status'
import { clearApiCache } from './api-cache'
import { queryClient } from '@/shared/api/query-client'
import { create } from 'zustand'

type SyncStatus = 'idle' | 'syncing' | 'error'

interface SyncState {
  status: SyncStatus
  pendingCount: number
  failedCount: number
  corruptCount: number
  failedItems: FailedMutation[]
  lastSyncAt: string | null
  setStatus: (s: SyncStatus) => void
  setPendingCount: (n: number) => void
  setFailed: (items: FailedMutation[], corruptCount: number) => void
  setLastSyncAt: (d: string) => void
}

export const useSyncState = create<SyncState>((set) => ({
  status: 'idle',
  pendingCount: 0,
  failedCount: 0,
  corruptCount: 0,
  failedItems: [],
  lastSyncAt: null,
  setStatus: (status) => set({ status }),
  setPendingCount: (pendingCount) => set({ pendingCount }),
  setFailed: (failedItems, corruptCount) =>
    set({ failedItems, failedCount: failedItems.length, corruptCount }),
  setLastSyncAt: (lastSyncAt) => set({ lastSyncAt }),
}))

function getCSRFToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)my_csrf=([^;]*)/)
  return match?.[1] ? decodeURIComponent(match[1]) : ''
}

type ReplayResult = { outcome: 'success' | 'failed' | 'retry'; status?: number }

async function replayMutation(m: QueuedMutation): Promise<ReplayResult> {
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
    if (res.ok) return { outcome: 'success' }
    // 4xx = the server rejected the payload permanently. The mutation moves
    // to the dead-letter state — the user decides (retry or discard) instead
    // of it vanishing. 5xx = transient, retry later.
    if (res.status >= 400 && res.status < 500) {
      return { outcome: 'failed', status: res.status }
    }
    return { outcome: 'retry' }
  } catch {
    return { outcome: 'retry' } // network error, retry later
  }
}

export async function drainQueue(): Promise<void> {
  const { setStatus, setPendingCount, setFailed, setLastSyncAt } = useSyncState.getState()
  const isOnline = useNetworkStatus.getState().isOnline
  if (!isOnline) return

  const pending = await getAll()
  const failed = await getAllFailed()
  const corruptCount = await getCorruptCount()
  if (pending.length === 0) {
    setPendingCount(0)
    setFailed(failed, corruptCount)
    // Dead letters alone still warrant the error state — the user has work
    // to review.
    setStatus(failed.length > 0 || corruptCount > 0 ? 'error' : 'idle')
    return
  }

  setStatus('syncing')
  setPendingCount(pending.length)

  for (const mutation of pending) {
    const { outcome, status } = await replayMutation(mutation)
    if (outcome === 'success') {
      await dequeue(mutation.id)
      setPendingCount(Math.max(0, useSyncState.getState().pendingCount - 1))
      continue
    }
    if (outcome === 'failed') {
      await markFailed(mutation.id, `HTTP ${status ?? '4xx'}`)
      setPendingCount(Math.max(0, useSyncState.getState().pendingCount - 1))
      continue
    }
    const updated = await incrementRetry(mutation.id)
    if (updated && updated.retries >= updated.maxRetries) {
      // Retry-exhausted → dead letter, not deletion.
      await markFailed(mutation.id, 'max retries exceeded')
      setPendingCount(Math.max(0, useSyncState.getState().pendingCount - 1))
    }
  }

  const remaining = await getAll()
  const failedAfter = await getAllFailed()
  const corruptAfter = await getCorruptCount()
  setPendingCount(remaining.length)
  setFailed(failedAfter, corruptAfter)
  setStatus(remaining.length > 0 || failedAfter.length > 0 || corruptAfter > 0 ? 'error' : 'idle')
  if (remaining.length === 0 && failedAfter.length === 0 && corruptAfter === 0) {
    setLastSyncAt(new Date().toISOString())
    // Everything replayed: refetch from the server and purge the SW's cached
    // API responses, or stale GETs (NetworkFirst, 10 min TTL) would keep
    // showing pre-mutation data and invite duplicate entries.
    queryClient.invalidateQueries()
    void clearApiCache().catch(() => undefined)
  }
}

/** Retry a dead-letter mutation: back to pending, then drain. */
export async function retryFailed(id: string): Promise<void> {
  await markPending(id)
  void drainQueue()
}

/** Permanently discard a dead-letter mutation. */
export async function discardFailed(id: string): Promise<void> {
  await dequeue(id)
  const failed = await getAllFailed()
  const corruptCount = await getCorruptCount()
  useSyncState.getState().setFailed(failed, corruptCount)
}

/** Discard unparseable parked entries. */
export async function discardCorruptAll(): Promise<void> {
  await discardCorrupt()
  const failed = await getAllFailed()
  useSyncState.getState().setFailed(failed, 0)
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
