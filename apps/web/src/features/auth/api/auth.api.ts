import { apiClient } from '@/shared/api/client'
import {
  AuthResponseSchema,
  MagicLinkRequest,
  MagicLinkRequestSchema,
  UserSchema,
  VerifyTokenRequest,
  VerifyTokenRequestSchema,
} from '../schemas/auth.schemas'

export function requestMagicLink(data: MagicLinkRequest) {
  return apiClient('/api/v1/auth/magic-link', AuthResponseSchema, {
    method: 'POST',
    body: JSON.stringify(MagicLinkRequestSchema.parse(data)),
  })
}

export function verifyToken(data: VerifyTokenRequest) {
  return apiClient('/api/v1/auth/verify', AuthResponseSchema, {
    method: 'POST',
    body: JSON.stringify(VerifyTokenRequestSchema.parse(data)),
  })
}

export function logout() {
  return apiClient('/api/v1/auth/logout', AuthResponseSchema, {
    method: 'POST',
  })
}

export function getCurrentUser() {
  return apiClient('/api/v1/auth/me', UserSchema)
}