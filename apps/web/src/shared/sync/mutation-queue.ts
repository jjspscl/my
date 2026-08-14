import { get, set, del, keys, createStore } from 'idb-keyval'
import { z } from 'zod'
import { randomUUID } from '@/shared/lib/uuid'

const mutationQueueStore = createStore('my-sync', 'mutations')
// Raw entries that no longer parse (corruption, foreign writes). Kept instead
// of deleted: the queue holds unsent user data, and silent deletion is data
// loss. The UI surfaces these with a discard action.
//
// Separate database on purpose: idb-keyval creates its object store only
// when the database is first created, so a second store on the already-
// existing 'my-sync' database would never exist ("No objectStore named
// corrupt" on every access, breaking the drain).
const corruptStore = createStore('my-sync-corrupt', 'corrupt')

const SCHEMA_VERSION = 1

export const QueuedMutationSchema = z.object({
  id: z.string(),
  schemaVersion: z.number(),
  method: z.enum(['POST', 'PATCH', 'PUT', 'DELETE']),
  url: z.string(),
  body: z.string().nullable(),
  createdAt: z.string(),
  retries: z.number(),
  maxRetries: z.number(),
  // 'failed' = dead letter: permanently rejected by the server (4xx) or
  // retry-exhausted. Kept for the user to retry or discard, never deleted.
  state: z.enum(['pending', 'failed']).default('pending'),
  failedReason: z.string().nullable().default(null),
  failedAt: z.string().nullable().default(null),
})

export type QueuedMutation = z.infer<typeof QueuedMutationSchema>

export type FailedMutation = QueuedMutation & {
  state: 'failed'
  failedReason: string
  failedAt: string
}

type EnqueueInput = Omit<
  QueuedMutation,
  'id' | 'schemaVersion' | 'createdAt' | 'retries' | 'maxRetries' | 'state' | 'failedReason' | 'failedAt'
> & { maxRetries?: number }

export async function enqueue(mutation: EnqueueInput): Promise<string> {
  const id = randomUUID()
  const entry: QueuedMutation = {
    ...mutation,
    id,
    schemaVersion: SCHEMA_VERSION,
    createdAt: new Date().toISOString(),
    retries: 0,
    maxRetries: mutation.maxRetries ?? 5,
    state: 'pending',
    failedReason: null,
    failedAt: null,
  }
  await set(id, entry, mutationQueueStore)
  return id
}

export async function dequeue(id: string): Promise<void> {
  await del(id, mutationQueueStore)
}

export async function markFailed(id: string, reason: string): Promise<void> {
  const raw = await get(id, mutationQueueStore)
  if (raw == null) return
  const parsed = QueuedMutationSchema.safeParse(raw)
  if (!parsed.success) return
  await set(
    id,
    {
      ...parsed.data,
      state: 'failed',
      failedReason: reason,
      failedAt: new Date().toISOString(),
    },
    mutationQueueStore,
  )
}

export async function markPending(id: string): Promise<void> {
  const raw = await get(id, mutationQueueStore)
  if (raw == null) return
  const parsed = QueuedMutationSchema.safeParse(raw)
  if (!parsed.success) return
  await set(
    id,
    { ...parsed.data, state: 'pending', retries: 0, failedReason: null, failedAt: null },
    mutationQueueStore,
  )
}

async function readEntries(store: typeof mutationQueueStore, corrupt: typeof corruptStore): Promise<{ entries: QueuedMutation[]; corrupt: unknown[] }> {
  const allKeys = await keys(store)
  const entries: QueuedMutation[] = []
  const corruptRaw: unknown[] = []
  for (const key of allKeys) {
    const raw = await get(key, store)
    const parsed = QueuedMutationSchema.safeParse(raw)
    if (parsed.success) {
      entries.push(parsed.data)
    } else {
      // Never delete unparseable entries — park them in the corrupt store.
      await set(key, raw, corrupt)
      await del(key, store)
      corruptRaw.push(raw)
    }
  }
  // Sort FIFO
  entries.sort((a, b) => a.createdAt.localeCompare(b.createdAt))
  return { entries, corrupt: corruptRaw }
}

/** Pending (replayable) mutations, FIFO. */
export async function getAll(): Promise<QueuedMutation[]> {
  const { entries } = await readEntries(mutationQueueStore, corruptStore)
  return entries.filter((e) => e.state === 'pending')
}

/** Dead-letter mutations: failed permanently, awaiting user decision. */
export async function getAllFailed(): Promise<FailedMutation[]> {
  const { entries } = await readEntries(mutationQueueStore, corruptStore)
  return entries.filter((e): e is FailedMutation => e.state === 'failed' && e.failedReason != null && e.failedAt != null)
}

/** Unparseable entries parked instead of deleted. */
export async function getCorruptCount(): Promise<number> {
  return (await keys(corruptStore)).length
}

export async function discardCorrupt(): Promise<void> {
  for (const key of await keys(corruptStore)) {
    await del(key, corruptStore)
  }
}

export async function incrementRetry(id: string): Promise<QueuedMutation | null> {
  const raw = await get(id, mutationQueueStore)
  const parsed = QueuedMutationSchema.safeParse(raw)
  if (!parsed.success) return null
  const updated = { ...parsed.data, retries: parsed.data.retries + 1 }
  // Exhaustion is the caller's decision (markFailed) — never delete here.
  await set(id, updated, mutationQueueStore)
  return updated
}

export async function queueSize(): Promise<number> {
  return (await getAll()).length
}
