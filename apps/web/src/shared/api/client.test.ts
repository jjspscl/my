import { describe, it, expect, vi, beforeEach } from 'vitest'
import { z } from 'zod'
import { apiClient, ApiError } from './client'

const TestSchema = z.object({
  id: z.string(),
  name: z.string(),
})

// Helper: extract opts from mocked fetch call
function getFetchOpts(mock: ReturnType<typeof vi.fn>): RequestInit {
  const call = mock.mock.calls[0]
  return call ? call[1] as RequestInit : {}
}

describe('apiClient', () => {
  beforeEach(() => {
    vi.stubGlobal('document', {
      cookie: '',
    })
  })

  it('returns parsed data from successful GET', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ id: '1', name: 'test' }),
      }),
    )

    const result = await apiClient('/api/test', TestSchema)
    expect(result).toEqual({ id: '1', name: 'test' })
  })

  it('sends credentials: include', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ id: '1', name: 'test' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    await apiClient('/api/test', TestSchema)

    const opts = getFetchOpts(mockFetch)
    expect(opts.credentials).toBe('include')
  })

  it('4xx response throws ApiError with server message', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 400,
        json: () => Promise.resolve({ error: 'invalid input' }),
      }),
    )

    await expect(apiClient('/api/test', TestSchema)).rejects.toThrow(ApiError)
    await expect(apiClient('/api/test', TestSchema)).rejects.toThrow('invalid input')
  })

  it('4xx response preserves status code', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: () => Promise.resolve({ error: 'unauthorized' }),
      }),
    )

    try {
      await apiClient('/api/test', TestSchema)
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError)
      expect((e as ApiError).status).toBe(401)
    }
  })

  it('network error propagates', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockRejectedValue(new Error('Network connection failed')),
    )

    await expect(apiClient('/api/test', TestSchema)).rejects.toThrow('Network connection failed')
  })

  it('POST includes X-CSRF-Token header from cookie', async () => {
    vi.stubGlobal('document', {
      cookie: 'my_csrf=csrf-token-value; other=stuff',
    })

    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ id: '1', name: 'test' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    await apiClient('/api/test', TestSchema, { method: 'POST', body: JSON.stringify({}) })

    const opts = getFetchOpts(mockFetch)
    const headers = new Headers(opts.headers)
    expect(headers.get('X-CSRF-Token')).toBe('csrf-token-value')
  })

  it('GET does not include X-CSRF-Token header', async () => {
    vi.stubGlobal('document', {
      cookie: 'my_csrf=csrf-token-value',
    })

    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ id: '1', name: 'test' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    await apiClient('/api/test', TestSchema)

    const opts = getFetchOpts(mockFetch)
    const headers = new Headers(opts.headers)
    expect(headers.get('X-CSRF-Token')).toBeNull()
  })

  it('POST sets Content-Type to application/json when body is present', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ id: '1', name: 'test' }),
    })
    vi.stubGlobal('fetch', mockFetch)

    await apiClient('/api/test', TestSchema, { method: 'POST', body: JSON.stringify({ name: 'test' }) })

    const opts = getFetchOpts(mockFetch)
    const headers = new Headers(opts.headers)
    expect(headers.get('Content-Type')).toBe('application/json')
  })

  it('returns parsed data matching schema', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ id: '42', name: 'hello' }),
      }),
    )

    const result = await apiClient('/api/test', TestSchema)
    expect(result.id).toBe('42')
    expect(result.name).toBe('hello')
  })

  it('500 response throws with server error message', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: () => Promise.resolve({ error: 'internal server error' }),
      }),
    )

    await expect(apiClient('/api/test', TestSchema)).rejects.toThrow('internal server error')
  })
})