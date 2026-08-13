const API_CACHE_NAME = 'api-cache'

/**
 * Purges the service worker's API response cache. Called on logout and after
 * a successful sync drain: without it, the SW keeps serving pre-mutation GET
 * responses for up to 10 minutes (NetworkFirst TTL), showing stale lists that
 * invite duplicate offline entries.
 */
export async function clearApiCache(): Promise<void> {
  if (typeof caches === 'undefined') return
  await caches.delete(API_CACHE_NAME)
}
