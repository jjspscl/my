import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useEffect, useRef } from 'react'
import { z } from 'zod'
import { useVerifyToken } from '@/features/auth/hooks/use-auth'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const VerifySearchSchema = z.object({
  token: z.string().min(1),
})

export const Route = createFileRoute('/auth/verify')({
  validateSearch: VerifySearchSchema,
  component: VerifyPage,
})

export function VerifyPage() {
  const { token } = Route.useSearch()
  const verifyToken = useVerifyToken()
  const navigate = useNavigate()
// Single-flight guard: StrictMode double-invokes effects, and the mutation
// object identity changes every render. calledRef keeps exactly one POST per
// token. The mutation itself is idempotent server-side (tokens are
// single-use), so this is belt-and-braces.
  const calledRef = useRef(false)

  useEffect(() => {
    if (!calledRef.current && !verifyToken.isSuccess && !verifyToken.isError) {
      calledRef.current = true
      verifyToken.mutate({ token })
    }
  }, [token, verifyToken])

  if (verifyToken.isIdle || verifyToken.isPending) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <Card className="w-full max-w-sm">
          <CardHeader>
            <CardTitle className="text-center">Verifying...</CardTitle>
            <CardDescription className="text-center">
              Checking your magic link.
            </CardDescription>
          </CardHeader>
        </Card>
      </div>
    )
  }

  if (verifyToken.isError) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <Card className="w-full max-w-sm">
          <CardHeader>
            <CardTitle className="text-center">Link expired or invalid</CardTitle>
            <CardDescription className="text-center">
              {verifyToken.error?.message || 'Could not verify this magic link.'}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              className="w-full"
              onClick={() => navigate({ to: '/login' })}
            >
              Back to sign in
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return null
}