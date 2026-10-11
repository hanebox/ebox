import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { api } from '@/api/endpoints'
import { Button } from '@/components/ui/button'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { LogoMark } from '@/components/ui/logo'
import { toast } from '@/components/ui/toast'
import { authQuery } from '@/hooks/use-auth'
import { ApiError } from '@/lib/api'

const schema = z.object({
  email: z.email('Enter a valid email address.'),
  password: z.string().min(1, 'Enter your password.'),
})

type FormValues = z.infer<typeof schema>

export function LoginPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: '', password: '' },
  })

  const mutation = useMutation({
    mutationFn: (values: FormValues) => api.login(values.email, values.password),
    onSuccess: (data) => {
      queryClient.setQueryData(authQuery.queryKey, data)
      navigate({ to: '/' })
    },
    onError: (error) => {
      toast.error(
        'Sign in failed',
        error instanceof ApiError ? error.message : 'Unexpected error',
      )
    },
  })

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden bg-background px-4 py-10">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 top-0 h-[520px] bg-[radial-gradient(65%_45%_at_50%_-5%,var(--accent-soft),transparent_72%)]"
      />

      <div className="relative w-full max-w-sm">
        <div className="mb-6 flex flex-col items-center gap-3 text-center">
          <LogoMark className="size-11" />
          <div className="flex flex-col gap-1">
            <h1 className="text-lg font-semibold tracking-[-0.02em] text-foreground">
              Sign in to ebox
            </h1>
            <p className="text-ui text-muted">Your personal server control plane</p>
          </div>
        </div>

        <form
          className="flex flex-col gap-4 rounded-panel border border-border bg-surface p-5"
          onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        >
          <Field label="Email" error={form.formState.errors.email?.message}>
            {({ id, invalid, ...aria }) => (
              <Input
                id={id}
                invalid={invalid}
                type="email"
                autoComplete="email"
                autoFocus
                placeholder="you@example.com"
                {...aria}
                {...form.register('email')}
              />
            )}
          </Field>

          <Field label="Password" error={form.formState.errors.password?.message}>
            {({ id, invalid, ...aria }) => (
              <Input
                id={id}
                invalid={invalid}
                type="password"
                autoComplete="current-password"
                {...aria}
                {...form.register('password')}
              />
            )}
          </Field>

          <Button type="submit" variant="primary" size="lg" loading={mutation.isPending}>
            Sign in
          </Button>
        </form>
      </div>
    </div>
  )
}
