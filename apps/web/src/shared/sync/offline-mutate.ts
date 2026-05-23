import { useNetworkStatus } from './network-status'
import { enqueue } from './mutation-queue'

/**
 * Wraps a mutation call: if online, execute immediately.
 * If offline, queue for later replay and throw a known error.
 */
export async function offlineMutate(
  url: string,
  options: { method: 'POST' | 'PATCH' | 'PUT' | 'DELETE'; body?: unknown },
): Promise<Response | 'queued'> {
  const isOnline = useNetworkStatus.getState().isOnline

  if (isOnline) {
    // Attempt live request
    try {
      const headers: Record<string, string> = {
        'X-CSRF-Token': getCSRFToken(),
      }
      if (options.body) {
        headers['Content-Type'] = 'application/json'
      }
      const res = await fetch(url, {
        method: options.method,
        headers,
        body: options.body ? JSON.stringify(options.body) : null,
        credentials: 'include',
      })
      if (!res.ok && res.status >= 500) {
        // Server error — queue for retry
        await enqueue({
          method: options.method,
          url,
          body: options.body ? JSON.stringify(options.body) : null,
        })
        return 'queued'
      }
      return res
    } catch {
      // Network error despite navigator.onLine — queue it
      await enqueue({
        method: options.method,
        url,
        body: options.body ? JSON.stringify(options.body) : null,
      })
      return 'queued'
    }
  }

  // Offline — queue immediately
  await enqueue({
    method: options.method,
    url,
    body: options.body ? JSON.stringify(options.body) : null,
  })
  return 'queued'
}

function getCSRFToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)my_csrf=([^;]*)/)
  return match?.[1] ? decodeURIComponent(match[1]) : ''
}