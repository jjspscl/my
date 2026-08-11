import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Mail } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { useRequestMagicLink } from '../hooks/use-auth'
import {
  MagicLinkRequestSchema,
  type MagicLinkRequest,
} from '../schemas/auth.schemas'

export function LoginForm() {
  const [sent, setSent] = useState(false)
  const requestMagicLink = useRequestMagicLink()

  const form = useForm<MagicLinkRequest>({
    resolver: zodResolver(MagicLinkRequestSchema),
    defaultValues: { email: '' },
  })

  function onSubmit(data: MagicLinkRequest) {
    requestMagicLink.mutate(data, {
      onSuccess: () => setSent(true),
      // A failed request (SMTP down, rate limit, server error) must not look
      // like "nothing happened".
      onError: (err) => {
        toast.error(err.message || 'Could not send the magic link. Try again.')
      },
    })
  }

  if (sent) {
    return (
      <Card className="w-full max-w-sm">
        <CardHeader>
          <div className="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-full bg-muted">
            <Mail className="h-6 w-6 text-muted-foreground" />
          </div>
          <CardTitle className="text-center">Check your email</CardTitle>
          <CardDescription className="text-center">
            If that email is registered, a magic link is on its way.
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <Card className="w-full max-w-sm">
      <CardHeader>
        <CardTitle className="text-center">Sign in to my</CardTitle>
        <CardDescription className="text-center">
          Enter your email to receive a magic link.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="email"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="you@example.com"
                      type="email"
                      autoComplete="email"
                      autoFocus
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button
              type="submit"
              className="w-full"
              disabled={requestMagicLink.isPending}
            >
              {requestMagicLink.isPending ? 'Sending...' : 'Send magic link'}
            </Button>
          </form>
        </Form>
      </CardContent>
    </Card>
  )
}