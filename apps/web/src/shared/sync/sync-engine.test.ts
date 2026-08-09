import 'fake-indexeddb/auto'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clear, createStore } from 'idb-keyval'
import { enqueue, getAll, getAllFailed } from './mutation-queue'
import { drainQueue } from './sync-engine'
import { useNetworkStatus } from './network-status'

async function resetStores() {
  for (const [db, name] of [
    ['my-sync', 'mutations'],
    ['my-sync-corrupt', 'corrupt'],
  ] as const) {
    try {
      await clear(createStore(db, name))
    } catch {
      // store not created yet
    }
  }
}

async function seed(maxRetries?: number): Promise<string> {
  return enqueue({
    method: 'POST',
    url: '/api/v1/finance/transactions',
    body: JSON.stringify({ amount: 100 }),
    maxRetries,
  })
}

function fetchResponse(status: number) {
  return new Response(JSON.stringify({}), { status })
}

describe('sync-engine drainQueue', () => {
  beforeEach(async () => {
    await resetStores()
    useNetworkStatus.setState({ isOnline: true })
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(async () => {
    vi.unstubAllGlobals()
    await resetStores()
  })

  it('does nothing when offline', async () => {
    useNetworkStatus.setState({ isOnline: false })
    const id = await seed()
    await drainQueue()
    const pending = await getAll()
    expect(pending).toHaveLength(1)
    expect(pending[0]!.id).toBe(id)
  })

  it('dequeues on 2xx', async () => {
    vi.mocked(fetch).mockResolvedValue(fetchResponse(200))
    await seed()
    await drainQueue()
    expect(await getAll()).toHaveLength(0)
    expect(await getAllFailed()).toHaveLength(0)
  })

  it('moves a 4xx to the dead-letter state instead of deleting', async () => {
    vi.mocked(fetch).mockResolvedValue(fetchResponse(422))
    await seed()
    await drainQueue()

    expect(await getAll()).toHaveLength(0)
    const failed = await getAllFailed()
    expect(failed).toHaveLength(1)
    expect(failed[0]!.failedReason).toBe('HTTP 422')
  })

  it('keeps a 5xx pending and increments the retry counter', async () => {
    vi.mocked(fetch).mockResolvedValue(fetchResponse(500))
    await seed()
    await drainQueue()

    const pending = await getAll()
    expect(pending).toHaveLength(1)
    expect(pending[0]!.retries).toBe(1)
    expect(await getAllFailed()).toHaveLength(0)
  })

  it('moves retry-exhausted mutations to the dead-letter state', async () => {
    vi.mocked(fetch).mockResolvedValue(fetchResponse(500))
    // maxRetries 1: first drain increments to 1 (exhausted).
    await seed(1)
    await drainQueue()

    expect(await getAll()).toHaveLength(0)
    const failed = await getAllFailed()
    expect(failed).toHaveLength(1)
    expect(failed[0]!.failedReason).toBe('max retries exceeded')
  })

  it('recovers a dead letter via retry after markPending', async () => {
    vi.mocked(fetch)
      .mockResolvedValueOnce(fetchResponse(422))
      .mockResolvedValueOnce(fetchResponse(200))
    const id = await seed()

    await drainQueue()
    expect(await getAllFailed()).toHaveLength(1)

    // User hits Retry: back to pending, drain succeeds.
    const { markPending } = await import('./mutation-queue')
    await markPending(id)
    await drainQueue()

    expect(await getAll()).toHaveLength(0)
    expect(await getAllFailed()).toHaveLength(0)
  })
})
