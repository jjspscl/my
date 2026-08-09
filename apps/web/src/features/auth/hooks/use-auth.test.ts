import { afterEach, describe, expect, it, vi } from 'vitest'
import { clearApiCache } from '@/shared/sync/api-cache'

describe('clearApiCache', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('deletes the Workbox API cache', async () => {
    const deleteCache = vi.fn().mockResolvedValue(true)
    vi.stubGlobal('caches', { delete: deleteCache })

    await clearApiCache()

    expect(deleteCache).toHaveBeenCalledWith('api-cache')
  })

  it('does nothing when CacheStorage is unavailable', async () => {
    vi.stubGlobal('caches', undefined)

    await expect(clearApiCache()).resolves.toBeUndefined()
  })
})
