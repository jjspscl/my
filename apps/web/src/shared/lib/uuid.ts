// Browser-safe UUID v4 generation.
//
// `crypto.randomUUID` is only available in secure contexts (HTTPS or
// loopback). The dashboard is intentionally served over plain HTTP on a
// private tailnet address, which is NOT a secure context, so we fall back to
// RFC 4122 v4 generation from `crypto.getRandomValues`. These UUIDs are used
// as idempotency keys, so we never fall back to Math.random().

const UUID_V4_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/

export function randomUUID(): string {
  const cryptoObj = globalThis.crypto

  if (typeof cryptoObj?.randomUUID === 'function') {
    return cryptoObj.randomUUID()
  }

  if (!cryptoObj || typeof cryptoObj.getRandomValues !== 'function') {
    throw new Error(
      'crypto.getRandomValues is unavailable — cannot generate a secure UUID. ' +
        'Serve the app over HTTPS or a loopback origin.',
    )
  }

  const bytes = new Uint8Array(16)
  cryptoObj.getRandomValues(bytes)

  // RFC 4122 v4: version nibble 4, variant bits 10xx.
  bytes[6] = (bytes[6]! & 0x0f) | 0x40
  bytes[8] = (bytes[8]! & 0x3f) | 0x80

  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
  const uuid = `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`

  // Defensive: the fallback must never emit a malformed UUID.
  if (!UUID_V4_RE.test(uuid)) {
    throw new Error('generated UUID failed validation')
  }
  return uuid
}