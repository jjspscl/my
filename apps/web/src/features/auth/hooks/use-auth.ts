import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { authKeys } from '../api/auth.keys'
import { clearApiCache } from '@/shared/sync/api-cache'
import {
  getCurrentUser,
  logout as logoutApi,
  requestMagicLink as requestMagicLinkApi,
  verifyToken as verifyTokenApi,
} from '../api/auth.api'

export function useCurrentUser() {
  return useQuery({
    queryKey: authKeys.me(),
    queryFn: getCurrentUser,
    retry: false,
    staleTime: 1000 * 60 * 5,
  })
}

export function useRequestMagicLink() {
  return useMutation({
    mutationFn: requestMagicLinkApi,
  })
}

export function useVerifyToken() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  return useMutation({
    mutationFn: verifyTokenApi,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: authKeys.all })
      navigate({ to: '/' })
    },
  })
}

export function useLogout() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  return useMutation({
    mutationFn: logoutApi,
    onSuccess: () => {
      queryClient.clear()
      void clearApiCache().catch(() => undefined)
      navigate({ to: '/login' })
    },
  })
}
