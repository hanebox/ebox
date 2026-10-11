import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { ServerCog } from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { api } from '@/api/endpoints'
import { Button } from '@/components/ui/button'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { toast } from '@/components/ui/toast'
import { authQuery, setupStatusQuery } from '@/hooks/use-auth'
import { ApiError } from '@/lib/api'

const schema = z
  .object({
    name: z.string().min(1, 'Your name is required.'),
    email: z.email('Enter a valid email address.'),
    password: z.string().min(10, 'Use at least 10 characters.'),
    confirm: z.string().min(1, 'Confirm your password.'),
    panel_domain: z.string().optional(),
    acme_email: z.string().optional(),
  })
  .refine((values) => values.password === values.confirm, {
    path: ['confirm'],
    message: 'Passwords do not match.',
  })

type FormValues = z.infer<typeof schema>

export function SetupPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [showAdvanced, setShowAdvanced] = useState(false)

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      email: '',
      password: '',
      confirm: '',
      panel_domain: '',
      acme_email: '',
    },
  })

  const mutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.setup({
        name: values.name,
        email: values.email,
        password: values.password,
        panel_domain: values.panel_domain?.trim() || undefined,
        acme_email: values.acme_email?.trim() || undefined,
      }),
    onSuccess: (data) => {
      queryClient.setQueryData(authQuery.queryKey, data)
      queryClient.setQueryData(setupStatusQuery.queryKey, { initialized: true })
      toast.success('Welcome to ebox', 'Your admin account is ready.')
      navigate({ to: '/' })
    },
    onError: (error) => {
      if (error instanceof ApiError && error.errors) {
        for (const [field, messages] of Object.entries(error.errors)) {
          if (messages.length > 0) {
            form.setError(field as keyof FormValues, { message: messages[0] })
          }
        }
      }
      toast.error('Setup failed', error instanceof ApiError ? error.message : 'Unexpected error')
    },
  })

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4 py-10">
      <div className="w-full max-w-md">
        <div className="mb-6 flex flex-col items-center gap-2 text-center">
          <div className="flex size-10 items-center justify-center rounded-xl bg-accent text-lg font-bold text-accent-foreground">
            e
          </div>
          <h1 className="text-xl font-semibold tracking-tight">Set up ebox</h1>
          <p className="max-w-sm text-sm text-muted">
            Create the admin account for this panel. There are no defaults — this account owns
            everything.
          </p>
        </div>

        <form
          className="flex flex-col gap-4 rounded-panel border border-border bg-surface p-5 shadow-sm"
          onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
        >
          <Field label="Name" error={form.formState.errors.name?.message}>
            {({ id, invalid, ...aria }) => (
              <Input
                id={id}
                invalid={invalid}
                autoComplete="name"
                placeholder="Your name"
                {...aria}
                {...form.register('name')}
              />
            )}
          </Field>

          <Field label="Email" error={form.formState.errors.email?.message}>
            {({ id, invalid, ...aria }) => (
              <Input
                id={id}
                invalid={invalid}
                type="email"
                autoComplete="email"
                placeholder="you@example.com"
                {...aria}
                {...form.register('email')}
              />
            )}
          </Field>

          <div className="grid grid-cols-2 gap-3">
            <Field label="Password" error={form.formState.errors.password?.message}>
              {({ id, invalid, ...aria }) => (
                <Input
                  id={id}
                  invalid={invalid}
                  type="password"
                  autoComplete="new-password"
                  {...aria}
                  {...form.register('password')}
                />
              )}
            </Field>
            <Field label="Confirm password" error={form.formState.errors.confirm?.message}>
              {({ id, invalid, ...aria }) => (
                <Input
                  id={id}
                  invalid={invalid}
                  type="password"
                  autoComplete="new-password"
                  {...aria}
                  {...form.register('confirm')}
                />
              )}
            </Field>
          </div>

          <div className="flex items-center justify-between rounded-lg border border-border px-3 py-2.5">
            <div className="flex flex-col">
              <span className="text-sm text-foreground">Advanced</span>
              <span className="text-xs text-subtle">Panel domain and TLS email (optional)</span>
            </div>
            <Switch
              checked={showAdvanced}
              onCheckedChange={setShowAdvanced}
              aria-label="Show advanced settings"
            />
          </div>

          {showAdvanced ? (
            <>
              <Field
                label="Panel domain"
                description="Used for the HTTPS route to this panel once Traefik is configured."
                error={form.formState.errors.panel_domain?.message}
              >
                {({ id, invalid, ...aria }) => (
                  <Input
                    id={id}
                    invalid={invalid}
                    placeholder="panel.example.com"
                    {...aria}
                    {...form.register('panel_domain')}
                  />
                )}
              </Field>
              <Field
                label="ACME email"
                description="Let's Encrypt account email for automatic certificates."
                error={form.formState.errors.acme_email?.message}
              >
                {({ id, invalid, ...aria }) => (
                  <Input
                    id={id}
                    invalid={invalid}
                    type="email"
                    placeholder="you@example.com"
                    {...aria}
                    {...form.register('acme_email')}
                  />
                )}
              </Field>
            </>
          ) : null}

          <Button type="submit" variant="primary" size="lg" loading={mutation.isPending}>
            Create account
          </Button>
        </form>

        <p className="mt-4 flex items-center justify-center gap-1.5 text-xs text-subtle">
          <ServerCog aria-hidden className="size-3.5" />
          Designed for a single personal server
        </p>
      </div>
    </div>
  )
}
