import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'

import { api } from '@/api/endpoints'
import type { Source } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Modal } from '@/components/ui/dialog'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'

const schema = z
  .object({
    name: z.string().min(1, 'Name is required').max(255),
    git_url: z.string().min(1, 'Repository URL is required').max(512),
    auth_type: z.enum(['none', 'token', 'ssh']),
    token: z.string().optional(),
    ssh_key: z.string().optional(),
  })
  .refine((values) => values.auth_type !== 'token' || Boolean(values.token), {
    path: ['token'],
    message: 'A personal access token is required.',
  })
  .refine((values) => values.auth_type !== 'ssh' || Boolean(values.ssh_key), {
    path: ['ssh_key'],
    message: 'A private key is required.',
  })

type FormValues = z.infer<typeof schema>

export interface GitSourceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated?: (source: Source) => void
}

export function GitSourceDialog({ open, onOpenChange, onCreated }: GitSourceDialogProps) {
  const queryClient = useQueryClient()

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: '', git_url: '', auth_type: 'none', token: '', ssh_key: '' },
  })

  const authType = useWatch({ control: form.control, name: 'auth_type' })

  const mutation = useMutation({
    mutationFn: (values: FormValues) =>
      api.createGitSource({
        name: values.name,
        git_url: values.git_url,
        auth_type: values.auth_type,
        token: values.auth_type === 'token' ? values.token : undefined,
        ssh_key: values.auth_type === 'ssh' ? values.ssh_key : undefined,
      }),
    onSuccess: (source) => {
      void queryClient.invalidateQueries({ queryKey: ['sources'] })
      toast.success('Source added', `${source.name} is reachable.`)
      onOpenChange(false)
      form.reset()
      onCreated?.(source)
    },
    onError: (error) => {
      toast.error(
        'Could not add the source',
        error instanceof ApiError ? error.message : undefined,
      )
    },
  })

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title="Add git source"
      description="Any repository the panel can reach: GitHub, GitLab, Gitea, or a plain git URL."
    >
      <form
        className="flex flex-col gap-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
      >
        <Field label="Name" error={form.formState.errors.name?.message}>
          {(field) => (
            <Input
              {...field}
              {...form.register('name')}
              placeholder="my-repo"
              autoFocus
              invalid={Boolean(form.formState.errors.name)}
            />
          )}
        </Field>

        <Field
          label="Repository URL"
          description="HTTPS or SSH clone URL."
          error={form.formState.errors.git_url?.message}
        >
          {(field) => (
            <Input
              {...field}
              {...form.register('git_url')}
              placeholder="https://github.com/acme/app.git"
              invalid={Boolean(form.formState.errors.git_url)}
            />
          )}
        </Field>

        <Field label="Authentication" error={form.formState.errors.auth_type?.message}>
          {(field) => (
            <Select {...field} {...form.register('auth_type')}>
              <option value="none">Public repository</option>
              <option value="token">Access token</option>
              <option value="ssh">SSH private key</option>
            </Select>
          )}
        </Field>

        {authType === 'token' ? (
          <Field
            label="Access token"
            description="Stored encrypted. Used only for git operations from this panel."
            error={form.formState.errors.token?.message}
          >
            {(field) => (
              <Input
                {...field}
                {...form.register('token')}
                type="password"
                placeholder="ghp_…"
                autoComplete="off"
                invalid={Boolean(form.formState.errors.token)}
              />
            )}
          </Field>
        ) : null}

        {authType === 'ssh' ? (
          <Field
            label="Private key"
            description="PEM or OpenSSH private key. Host keys are accepted on first use."
            error={form.formState.errors.ssh_key?.message}
          >
            {(field) => (
              <Textarea
                {...field}
                {...form.register('ssh_key')}
                rows={5}
                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                className="font-mono text-xs"
                invalid={Boolean(form.formState.errors.ssh_key)}
              />
            )}
          </Field>
        ) : null}

        <p className="text-xs text-subtle">
          The panel verifies the repository is reachable before saving.
        </p>

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
            Add source
          </Button>
        </div>
      </form>
    </Modal>
  )
}
