import { ZodSchema } from 'zod'

export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
    this.name = 'ApiError'
  }
}

function getCSRFToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)my_csrf=([^;]*)/)
  return match?.[1] ? decodeURIComponent(match[1]) : ''
}

export async function apiClient<T>(
  url: string,
  schema: ZodSchema<T>,
  options: RequestInit = {},
): Promise<T> {
  const method = (options.method || 'GET').toUpperCase()
  const isMutation = !['GET', 'HEAD', 'OPTIONS'].includes(method)

  const reqHeaders = new Headers(options.headers)

  if (options.body && !reqHeaders.has('Content-Type')) {
    reqHeaders.set('Content-Type', 'application/json')
  }

  if (isMutation) {
    reqHeaders.set('X-CSRF-Token', getCSRFToken())
  }

  const res = await fetch(url, {
    ...options,
    headers: reqHeaders,
    credentials: 'include',
  })

  const body = await res.json()

  if (!res.ok) {
    throw new ApiError(body.error || 'Request failed', res.status)
  }

  return schema.parse(body)
}