// Browser-safe SHA-256 hashing.
//
// `crypto.subtle` is only available in secure contexts (HTTPS or loopback).
// The dashboard is intentionally served over plain HTTP on a private tailnet
// address, which is NOT a secure context — `crypto` and `getRandomValues`
// exist there, but `crypto.subtle` is undefined (the same restriction behind
// the v1.1.1 randomUUID fallback). When that happens we fall back to the
// audited @noble/hashes implementation, which is loaded lazily so the
// dashboard bundle never pays for it on secure origins.
//
// Hashing stays entirely browser-local; the digest is used as a fingerprint
// for import replay detection.

/**
 * SHA-256 hex digest of an ArrayBuffer, using Web Crypto when available and
 * @noble/hashes otherwise.
 */
export async function sha256Hex(buffer: ArrayBuffer): Promise<string> {
  const subtle = globalThis.crypto?.subtle
  if (subtle && typeof subtle.digest === 'function') {
    const digest = await subtle.digest('SHA-256', buffer)
    return bytesToHex(new Uint8Array(digest))
  }

  const { sha256 } = await import('@noble/hashes/sha2.js')
  const { bytesToHex: nobleBytesToHex } = await import('@noble/hashes/utils.js')
  return nobleBytesToHex(sha256(new Uint8Array(buffer)))
}

function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
}
