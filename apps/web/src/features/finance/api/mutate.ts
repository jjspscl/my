import { offlineMutate } from '@/shared/sync'
import { ApiError } from '@/shared/api/client'
import type { ZodSchema } from 'zod'

export type QueuedResult = { queued: true }
export type MutateResult<T> = T | QueuedResult

export function isQueued<T>(r: MutateResult<T>): r is QueuedResult {
  return (r as QueuedResult).queued === true
}

/**
 * POST helper with offline queueing and idempotency.
 *
 * The caller generates an idempotency key once per logical mutation and
 * includes it in the body. `offlineMutate` stores the body verbatim in the
 * mutation queue and the sync engine replays it verbatim, so every retry of
 * the same mutation carries the same key and the server dedupes it.
 *
 * When offline or after a transient server failure the mutation is queued and
 * this resolves with `{ queued: true }` instead of throwing — the caller
 * treats that as success from the UI's perspective.
 */
export async function financeMutate<T>(
  url: string,
  body: unknown,
  schema: ZodSchema<T>,
): Promise<MutateResult<T>> {
  const res = await offlineMutate(url, { method: 'POST', body })
  if (res === 'queued') return { queued: true }

  const data = await res.json()
  if (!res.ok) {
    throw new ApiError(data.error || 'Request failed', res.status)
  }
  return schema.parse(data)
}
