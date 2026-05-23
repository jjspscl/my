import { z } from 'zod'

export const MagicLinkRequestSchema = z.object({
  email: z.string().email('Enter a valid email address'),
})
export type MagicLinkRequest = z.infer<typeof MagicLinkRequestSchema>

export const VerifyTokenRequestSchema = z.object({
  token: z.string().min(1),
})
export type VerifyTokenRequest = z.infer<typeof VerifyTokenRequestSchema>

export const UserSchema = z.object({
  email: z.string().email(),
})
export type User = z.infer<typeof UserSchema>

export const AuthResponseSchema = z.object({
  ok: z.literal(true).optional(),
  error: z.string().optional(),
})
export type AuthResponse = z.infer<typeof AuthResponseSchema>