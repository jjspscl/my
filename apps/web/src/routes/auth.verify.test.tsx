import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { VerifyPage } from './auth.verify'

// Controllable useVerifyToken state so each test renders the route in a
// specific mutation phase.
const state = vi.hoisted(() => ({
  current: {
    isIdle: true,
    isPending: false,
    isError: false,
    error: null as Error | null,
    mutate: vi.fn(),
  },
}))

vi.mock('@/features/auth/hooks/use-auth', () => ({
  useVerifyToken: () => state.current,
}))

vi.mock('@tanstack/react-router', () => ({
  createFileRoute: () => () => ({
    useSearch: () => ({ token: 'tok-1' }),
  }),
  useNavigate: () => vi.fn(),
}))

describe('VerifyPage', () => {
  beforeEach(() => {
    state.current = {
      isIdle: true,
      isPending: false,
      isError: false,
      error: null,
      mutate: vi.fn(),
    }
  })

  it('renders the pending card before the mutation resolves (no blank frame)', () => {
    // Regression: the component used to match no branch while the mutation
    // was idle, returning null and flashing a blank screen on first paint.
    render(<VerifyPage />)
    expect(screen.getByText('Verifying...')).toBeInTheDocument()
    expect(state.current.mutate).toHaveBeenCalledWith({ token: 'tok-1' })
  })

  it('renders the error card when verification fails', () => {
    state.current = {
      isIdle: false,
      isPending: false,
      isError: true,
      error: new Error('invalid token'),
      mutate: vi.fn(),
    }

    render(<VerifyPage />)
    expect(screen.getByText('Link expired or invalid')).toBeInTheDocument()
    expect(screen.getByText('invalid token')).toBeInTheDocument()
  })
})
