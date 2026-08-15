import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { sha256Hex } from './sha256'

// Canonical SHA-256 vectors.
const EMPTY_HEX = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'
const ABC_HEX = 'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad'

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

function toBuffer(s: string): ArrayBuffer {
  return new TextEncoder().encode(s).buffer
}

describe('sha256Hex', () => {
  describe('native Web Crypto path (secure context)', () => {
    it('hashes empty input to the canonical digest', async () => {
      expect(await sha256Hex(new ArrayBuffer(0))).toBe(EMPTY_HEX)
    })

    it('hashes "abc" to the canonical digest', async () => {
      expect(await sha256Hex(toBuffer('abc'))).toBe(ABC_HEX)
    })
  })

  describe('fallback when subtle is unavailable (insecure origin)', () => {
    beforeEach(() => {
      // Reproduces the v1.2.0 crash: crypto exists, subtle does not.
      stubCrypto({ getRandomValues: realCrypto.getRandomValues.bind(realCrypto) })
    })

    it('does not throw', async () => {
      await expect(sha256Hex(toBuffer('abc'))).resolves.not.toThrow()
    })

    it('matches the canonical digest for "abc"', async () => {
      expect(await sha256Hex(toBuffer('abc'))).toBe(ABC_HEX)
    })

    it('matches the canonical digest for empty input', async () => {
      expect(await sha256Hex(new ArrayBuffer(0))).toBe(EMPTY_HEX)
    })
  })

  describe('fallback with no global crypto at all', () => {
    beforeEach(() => {
      stubCrypto(null)
    })

    it('still hashes via @noble/hashes', async () => {
      expect(await sha256Hex(toBuffer('abc'))).toBe(ABC_HEX)
    })
  })

  it('native and fallback results agree on a longer input', async () => {
    const input = toBuffer('GCash statement fingerprint check with spaces, commas, 1,234.56 and symbols!')
    const native = await sha256Hex(input) // secure context: real crypto present

    stubCrypto({ getRandomValues: realCrypto.getRandomValues.bind(realCrypto) })
    const fallback = await sha256Hex(input)

    expect(fallback).toBe(native)
    expect(fallback).toMatch(/^[0-9a-f]{64}$/)
  })
})
