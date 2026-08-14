import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { randomUUID } from './uuid'

// Canonical RFC 4122 v4: version nibble 4, variant 10xx ([89ab]).
const UUID_V4_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/

const realCrypto = globalThis.crypto

function stubCrypto(impl: Record<string, unknown> | null) {
  if (impl === null) {
    // @ts-expect-error deliberately removing crypto to simulate a hostile env
    delete globalThis.crypto
  } else {
    Object.defineProperty(globalThis, 'crypto', {
      value: impl,
      configurable: true,
      writable: true,
    })
  }
}

afterEach(() => {
  vi.restoreAllMocks()
  Object.defineProperty(globalThis, 'crypto', {
    value: realCrypto,
    configurable: true,
    writable: true,
  })
})

describe('randomUUID', () => {
  it('uses native crypto.randomUUID when available', () => {
    const native = vi.fn(() => '11111111-2222-4333-8444-555555555555')
    stubCrypto({
      randomUUID: native,
      getRandomValues: realCrypto.getRandomValues.bind(realCrypto),
    })

    expect(randomUUID()).toBe('11111111-2222-4333-8444-555555555555')
    expect(native).toHaveBeenCalledTimes(1)
  })

  describe('fallback when randomUUID is unavailable (insecure origin)', () => {
    beforeEach(() => {
      // Reproduces the v1.1.0 crash: crypto exists, randomUUID does not.
      stubCrypto({ getRandomValues: realCrypto.getRandomValues.bind(realCrypto) })
    })

    it('does not throw', () => {
      expect(() => randomUUID()).not.toThrow()
    })

    it('returns the canonical lowercase UUID v4 format', () => {
      for (let i = 0; i < 100; i++) {
        expect(randomUUID()).toMatch(UUID_V4_RE)
      }
    })

    it('sets the version nibble to 4', () => {
      for (let i = 0; i < 100; i++) {
        const uuid = randomUUID()
        expect(uuid[14]).toBe('4')
      }
    })

    it('sets the RFC 4122 variant bits (10xx)', () => {
      for (let i = 0; i < 100; i++) {
        const uuid = randomUUID()
        expect('89ab').toContain(uuid[19])
      }
    })

    it('produces distinct values across calls', () => {
      const seen = new Set<string>()
      for (let i = 0; i < 100; i++) {
        seen.add(randomUUID())
      }
      expect(seen.size).toBe(100)
    })
  })

  it('throws a clear error when no secure randomness source exists', () => {
    stubCrypto({})
    expect(() => randomUUID()).toThrow(/crypto/i)
  })
})