import { describe, it, expect } from 'vitest'
import { MagicLinkRequestSchema, VerifyTokenRequestSchema, UserSchema } from './auth.schemas'

describe('MagicLinkRequestSchema', () => {
  it('accepts valid email', () => {
    const result = MagicLinkRequestSchema.safeParse({ email: 'user@test.com' })
    expect(result.success).toBe(true)
  })

  it('rejects missing email', () => {
    const result = MagicLinkRequestSchema.safeParse({})
    expect(result.success).toBe(false)
  })

  it('rejects invalid email format', () => {
    const result = MagicLinkRequestSchema.safeParse({ email: 'not-an-email' })
    expect(result.success).toBe(false)
  })

  it('rejects empty email', () => {
    const result = MagicLinkRequestSchema.safeParse({ email: '' })
    expect(result.success).toBe(false)
  })

  it('rejects null email', () => {
    const result = MagicLinkRequestSchema.safeParse({ email: null })
    expect(result.success).toBe(false)
  })
})

describe('VerifyTokenRequestSchema', () => {
  it('accepts valid token', () => {
    const result = VerifyTokenRequestSchema.safeParse({ token: 'abc-123' })
    expect(result.success).toBe(true)
  })

  it('rejects empty token', () => {
    const result = VerifyTokenRequestSchema.safeParse({ token: '' })
    expect(result.success).toBe(false)
  })

  it('rejects missing token', () => {
    const result = VerifyTokenRequestSchema.safeParse({})
    expect(result.success).toBe(false)
  })
})

describe('UserSchema', () => {
  it('accepts valid user', () => {
    const result = UserSchema.safeParse({ email: 'user@test.com' })
    expect(result.success).toBe(true)
  })

  it('returns parsed email', () => {
    const user = UserSchema.parse({ email: 'user@test.com' })
    expect(user.email).toBe('user@test.com')
  })

  it('rejects missing email', () => {
    const result = UserSchema.safeParse({})
    expect(result.success).toBe(false)
  })

  it('rejects invalid email', () => {
    const result = UserSchema.safeParse({ email: 'bad' })
    expect(result.success).toBe(false)
  })
})