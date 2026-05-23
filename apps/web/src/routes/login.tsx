import { createFileRoute } from '@tanstack/react-router'
import { LoginForm } from '@/features/auth/components/login-form'
import { z } from 'zod'

const LoginSearchSchema = z.object({
  redirect: z.string().optional(),
})

export const Route = createFileRoute('/login')({
  validateSearch: LoginSearchSchema,
  component: LoginPage,
})

function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <LoginForm />
    </div>
  )
}