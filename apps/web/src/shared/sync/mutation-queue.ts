import { get, set, del, keys, createStore } from 'idb-keyval'
import { z } from 'zod'

const mutationQueueStore = createStore('my-sync', 'mutations')

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
})

export type QueuedMutation = z.infer<typeof QueuedMutationSchema>

export async function enqueue(mutation: Omit<QueuedMutation, 'id' | 'schemaVersion' | 'createdAt' | 'retries' | 'maxRetries'>): Promise<string> {
  const id = crypto.randomUUID()
  const entry: QueuedMutation = {
    ...mutation,
    id,
    schemaVersion: SCHEMA_VERSION,
    createdAt: new Date().toISOString(),
    retries: 0,
    maxRetries: 5,
  }
  await set(id, entry, mutationQueueStore)
  return id
}

export async function dequeue(id: string): Promise<void> {
  await del(id, mutationQueueStore)
}

export async function getAll(): Promise<QueuedMutation[]> {
  const allKeys = await keys(mutationQueueStore)
  const entries: QueuedMutation[] = []
  for (const key of allKeys) {
    const raw = await get(key, mutationQueueStore)
    const parsed = QueuedMutationSchema.safeParse(raw)
    if (parsed.success) {
      entries.push(parsed.data)
    } else {
      // Invalid schema version or corrupted — discard
      await del(key, mutationQueueStore)
    }
  }
  // Sort FIFO
  return entries.sort((a, b) => a.createdAt.localeCompare(b.createdAt))
}

export async function incrementRetry(id: string): Promise<QueuedMutation | null> {
  const raw = await get(id, mutationQueueStore)
  const parsed = QueuedMutationSchema.safeParse(raw)
  if (!parsed.success) return null
  const updated = { ...parsed.data, retries: parsed.data.retries + 1 }
  if (updated.retries >= updated.maxRetries) {
    await del(id, mutationQueueStore)
    return null
  }
  await set(id, updated, mutationQueueStore)
  return updated
}

export async function queueSize(): Promise<number> {
  return (await keys(mutationQueueStore)).length
}