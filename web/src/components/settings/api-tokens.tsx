import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { KeyRound, Plus, ShieldAlert } from 'lucide-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { AccessTokenCreated } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardTitle } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Copyable } from '@/components/ui/copyable'
import { Modal } from '@/components/ui/dialog'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatDateTime, formatRelative } from '@/lib/format'

const expiryOptions = [
  { value: '30', label: '30 days' },
  { value: '90', label: '90 days' },
  { value: '365', label: '1 year' },
  { value: '0', label: 'Never expires' },
]

export function ApiTokensSection() {
  const queryClient = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [name, setName] = useState('')
  const [days, setDays] = useState('90')
  const [nameError, setNameError] = useState<string | null>(null)
  const [created, setCreated] = useState<AccessTokenCreated | null>(null)
  const [revokeId, setRevokeId] = useState<number | null>(null)

  const tokens = useQuery({ queryKey: ['tokens'], queryFn: api.tokens })

  const createMutation = useMutation({
    mutationFn: () => api.createToken({ name: name.trim(), expires_in_days: Number(days) }),
    onSuccess: (data) => {
      void queryClient.invalidateQueries({ queryKey: ['tokens'] })
      setCreateOpen(false)
      setName('')
      setDays('90')
      setCreated(data)
    },
    onError: (error) => {
      toast.error(
        'Could not create the token',
        error instanceof ApiError ? error.message : undefined,
      )
    },
  })

  const revokeMutation = useMutation({
    mutationFn: (id: number) => api.deleteToken(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['tokens'] })
      toast.success('Token revoked')
      setRevokeId(null)
    },
    onError: (error) => {
      toast.error('Revoke failed', error instanceof ApiError ? error.message : undefined)
      setRevokeId(null)
    },
  })

  return (
    <Card className="max-w-3xl">
      <div className="flex flex-wrap items-start justify-between gap-3 p-4 pb-2">
        <div className="flex flex-col gap-1">
          <CardTitle>API tokens</CardTitle>
          <CardDescription>
            Bearer tokens for scripts and CI. Use them as{' '}
            <span className="font-mono">Authorization: Bearer …</span> against{' '}
            <span className="font-mono">/api/v1</span>.
          </CardDescription>
        </div>
        <Button variant="secondary" size="sm" onClick={() => setCreateOpen(true)}>
          <Plus aria-hidden className="size-3.5" />
          New token
        </Button>
      </div>

      <div className="p-4 pt-2">
        {tokens.isPending ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-8" />
            <Skeleton className="h-8" />
          </div>
        ) : null}

        {tokens.data && tokens.data.length === 0 ? (
          <div className="flex items-center gap-3 rounded-lg border border-dashed border-border px-4 py-6 text-xs text-muted">
            <KeyRound aria-hidden className="size-4 text-subtle" />
            No tokens yet. Create one for scripts, backups or CI.
          </div>
        ) : null}

        {tokens.data && tokens.data.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Last used</TableHead>
                <TableHead>Expires</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tokens.data.map((token) => (
                <TableRow key={token.id}>
                  <TableCell className="text-xs whitespace-nowrap text-foreground">
                    {token.name}
                  </TableCell>
                  <TableCell className="text-xs whitespace-nowrap text-muted">
                    {formatRelative(token.created_at)}
                  </TableCell>
                  <TableCell className="text-xs whitespace-nowrap text-muted">
                    {token.last_used_at ? formatRelative(token.last_used_at) : 'never'}
                  </TableCell>
                  <TableCell className="text-xs whitespace-nowrap text-muted">
                    {token.expires_at ? formatDateTime(token.expires_at) : 'never'}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-danger"
                      onClick={() => setRevokeId(token.id)}
                    >
                      Revoke
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : null}
      </div>

      <Modal
        open={createOpen}
        onOpenChange={setCreateOpen}
        title="New API token"
        description="The token is shown once after creation."
        size="sm"
      >
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            if (!name.trim()) {
              setNameError('Name is required.')
              return
            }
            setNameError(null)
            createMutation.mutate()
          }}
        >
          <Field label="Name" error={nameError ?? undefined}>
            {(field) => (
              <Input
                {...field}
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="backup-script"
                autoFocus
                invalid={Boolean(nameError)}
              />
            )}
          </Field>
          <Field label="Expiry">
            {(field) => (
              <Select {...field} value={days} onChange={(event) => setDays(event.target.value)}>
                {expiryOptions.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </Select>
            )}
          </Field>
          <div className="flex items-center justify-end gap-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => setCreateOpen(false)}
              disabled={createMutation.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" variant="primary" loading={createMutation.isPending}>
              Create token
            </Button>
          </div>
        </form>
      </Modal>

      <Modal
        open={created !== null}
        onOpenChange={(open) => {
          if (!open) {
            setCreated(null)
          }
        }}
        title="Token created"
        description="Copy it now — it will not be shown again."
        size="md"
      >
        <div className="flex flex-col gap-4">
          <div className="flex items-start gap-3 rounded-lg border border-warning/40 bg-warning-soft p-3">
            <ShieldAlert aria-hidden className="mt-0.5 size-4 shrink-0 text-warning" />
            <p className="text-xs text-foreground">
              Store this token in your CI secrets or password manager. Anyone with it can control the
              panel.
            </p>
          </div>
          {created ? <Copyable value={created.token} className="w-full" /> : null}
          <div className="flex justify-end">
            <Button variant="primary" onClick={() => setCreated(null)}>
              Done
            </Button>
          </div>
        </div>
      </Modal>

      <ConfirmDialog
        open={revokeId !== null}
        onOpenChange={(open) => {
          if (!open) {
            setRevokeId(null)
          }
        }}
        title="Revoke token"
        description="Anything using this token will lose access immediately."
        confirmLabel="Revoke"
        danger
        loading={revokeMutation.isPending}
        onConfirm={() => {
          if (revokeId !== null) {
            revokeMutation.mutate(revokeId)
          }
        }}
      />
    </Card>
  )
}
