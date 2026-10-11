import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { api } from '@/api/endpoints'
import type { Source } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Copyable } from '@/components/ui/copyable'
import { Modal } from '@/components/ui/dialog'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'

const schema = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  app_id: z.coerce.number().int().positive('Enter the numeric App ID'),
  private_key: z.string().min(1, 'The private key is required'),
  webhook_secret: z.string().optional(),
  installation_id: z.string().optional(),
})

type FormValues = z.input<typeof schema>

export interface GithubAppDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated?: (source: Source) => void
}

export function GithubAppDialog({ open, onOpenChange, onCreated }: GithubAppDialogProps) {
  const queryClient = useQueryClient()
  const origin = window.location.origin

  const form = useForm<FormValues, unknown, z.output<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      app_id: undefined,
      private_key: '',
      webhook_secret: '',
      installation_id: '',
    },
  })

  const mutation = useMutation({
    mutationFn: (values: z.output<typeof schema>) =>
      api.createGithubSource({
        name: values.name,
        app_id: values.app_id,
        private_key: values.private_key,
        webhook_secret: values.webhook_secret || undefined,
        installation_id: values.installation_id || undefined,
      }),
    onSuccess: (source) => {
      void queryClient.invalidateQueries({ queryKey: ['sources'] })
      toast.success('GitHub App connected', source.app_slug)
      onOpenChange(false)
      form.reset()
      onCreated?.(source)
    },
    onError: (error) => {
      toast.error(
        'Could not connect the app',
        error instanceof ApiError ? error.message : undefined,
      )
    },
  })

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title="Connect GitHub App manually"
      description="Use this when you prefer to create the GitHub App yourself."
      size="lg"
    >
      <form
        className="flex flex-col gap-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
      >
        <details className="rounded-lg border border-border bg-surface-raised p-3 text-xs">
          <summary className="cursor-pointer font-medium text-foreground select-none">
            Setup guide: create the app on GitHub first
          </summary>
          <div className="mt-3 flex flex-col gap-2.5 text-muted">
            <p>
              Create a GitHub App (Settings → Developer settings → GitHub Apps → New GitHub App)
              and paste these URLs:
            </p>
            <div className="flex flex-col gap-1.5">
              <span className="text-2xs font-medium text-subtle">Homepage URL</span>
              <Copyable value={origin} />
              <span className="mt-1 text-2xs font-medium text-subtle">
                Callback URL
              </span>
              <Copyable value={`${origin}/api/v1/github/callback`} />
              <span className="mt-1 text-2xs font-medium text-subtle">
                Webhook URL
              </span>
              <Copyable value={`${origin}/api/v1/webhooks/github`} />
            </div>
            <p>
              Repository permissions: <span className="text-foreground">Contents (read)</span> and{' '}
              <span className="text-foreground">Metadata (read)</span>. Subscribe to the{' '}
              <span className="text-foreground">push</span> event.
            </p>
            <p>
              After creating the app, generate a private key, then paste the App ID and key below.
            </p>
          </div>
        </details>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Name" error={form.formState.errors.name?.message}>
            {(field) => (
              <Input
                {...field}
                {...form.register('name')}
                placeholder="github"
                invalid={Boolean(form.formState.errors.name)}
              />
            )}
          </Field>

          <Field label="App ID" error={form.formState.errors.app_id?.message}>
            {(field) => (
              <Input
                {...field}
                {...form.register('app_id')}
                inputMode="numeric"
                placeholder="123456"
                invalid={Boolean(form.formState.errors.app_id)}
              />
            )}
          </Field>
        </div>

        <Field
          label="Private key"
          description="The .pem file GitHub generated for the app."
          error={form.formState.errors.private_key?.message}
        >
          {(field) => (
            <Textarea
              {...field}
              {...form.register('private_key')}
              rows={5}
              placeholder="-----BEGIN RSA PRIVATE KEY-----"
              className="font-mono text-xs"
              invalid={Boolean(form.formState.errors.private_key)}
            />
          )}
        </Field>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field
            label="Webhook secret"
            description="Optional. Needed for push auto-deploys."
            error={form.formState.errors.webhook_secret?.message}
          >
            {(field) => (
              <Input
                {...field}
                {...form.register('webhook_secret')}
                type="password"
                autoComplete="off"
                invalid={Boolean(form.formState.errors.webhook_secret)}
              />
            )}
          </Field>

          <Field
            label="Installation ID"
            description="Optional. Add later from the source page."
            error={form.formState.errors.installation_id?.message}
          >
            {(field) => (
              <Input
                {...field}
                {...form.register('installation_id')}
                inputMode="numeric"
                placeholder="12345678"
                invalid={Boolean(form.formState.errors.installation_id)}
              />
            )}
          </Field>
        </div>

        <div className="flex items-center justify-end gap-2 pt-1">
          <Button
            type="button"
            variant="ghost"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={mutation.isPending}>
            Connect app
          </Button>
        </div>
      </form>
    </Modal>
  )
}
