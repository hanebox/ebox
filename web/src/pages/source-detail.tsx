import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from '@tanstack/react-router'
import {
  ArrowLeft,
  ExternalLink,
  GitBranch,
  Pencil,
  PlugZap,
  RefreshCw,
  Trash2,
} from 'lucide-react'
import { useState, type ReactNode } from 'react'

import { api } from '@/api/endpoints'
import type { Source } from '@/api/types'
import { PageHeader } from '@/components/layout/page-header'
import { RepositoryList } from '@/components/sources/repository-list'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, Section } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Copyable } from '@/components/ui/copyable'
import { Modal } from '@/components/ui/dialog'
import { EmptyState } from '@/components/ui/empty-state'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Skeleton, Spinner } from '@/components/ui/skeleton'
import { GitHubIcon } from '@/components/ui/icons'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatDateTime } from '@/lib/format'

function InfoRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:gap-4">
      <span className="w-40 shrink-0 text-[11px] tracking-wide text-subtle uppercase">
        {label}
      </span>
      <div className="flex min-w-0 items-center gap-2 text-xs text-foreground">{children}</div>
    </div>
  )
}

function InstallationPanel({ source }: { source: Source }) {
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)

  const installations = useQuery({
    queryKey: ['source', source.uuid, 'installations'],
    queryFn: () => api.githubInstallations(source.uuid),
    enabled: open,
  })

  const attach = useMutation({
    mutationFn: (id: number) => api.setSourceInstallation(source.uuid, String(id)),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['source', source.uuid] })
      void queryClient.invalidateQueries({ queryKey: ['sources'] })
      toast.success('Installation connected')
      setOpen(false)
    },
    onError: (error) =>
      toast.error(
        'Could not attach the installation',
        error instanceof ApiError ? error.message : undefined,
      ),
  })

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-warning/40 bg-warning-soft p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-col gap-0.5">
          <p className="text-sm font-medium text-foreground">No installation connected</p>
          <p className="text-xs text-muted">
            Install the app on GitHub, then attach the installation to browse repositories.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {source.app_slug ? (
            <Button
              variant="primary"
              size="sm"
              onClick={() =>
                window.open(
                  `https://github.com/apps/${source.app_slug}/installations/new`,
                  '_blank',
                  'noopener,noreferrer',
                )
              }
            >
              <PlugZap aria-hidden className="size-3.5" />
              Install on GitHub
            </Button>
          ) : null}
          <Button variant="secondary" size="sm" onClick={() => setOpen((value) => !value)}>
            {open ? 'Hide installations' : 'Find installations'}
          </Button>
        </div>
      </div>

      {open ? (
        <div className="rounded-lg border border-border bg-surface p-2">
          {installations.isPending ? (
            <div className="flex items-center gap-2 p-2 text-xs text-muted">
              <Spinner />
              Looking up installations…
            </div>
          ) : null}
          {installations.isError ? (
            <p className="p-2 text-xs text-danger">
              {installations.error instanceof ApiError
                ? installations.error.message
                : 'Could not list installations.'}
            </p>
          ) : null}
          {installations.data && installations.data.length === 0 ? (
            <p className="p-2 text-xs text-subtle">
              GitHub reports no installations for this app yet. Install it first, then refresh.
            </p>
          ) : null}
          {installations.data && installations.data.length > 0 ? (
            <div className="flex flex-col divide-y divide-border/60">
              {installations.data.map((installation) => (
                <div
                  key={installation.id}
                  className="flex items-center justify-between gap-3 px-2 py-2"
                >
                  <div className="flex flex-col">
                    <span className="text-xs text-foreground">{installation.account}</span>
                    <span className="text-[11px] text-subtle">
                      {installation.type} · ID {installation.id}
                    </span>
                  </div>
                  <Button
                    variant="secondary"
                    size="sm"
                    loading={attach.isPending}
                    onClick={() => attach.mutate(installation.id)}
                  >
                    Attach
                  </Button>
                </div>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

export function SourceDetailPage() {
  const { sourceId } = useParams({ strict: false }) as { sourceId: string }
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [renameOpen, setRenameOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [name, setName] = useState('')

  const source = useQuery({
    queryKey: ['source', sourceId],
    queryFn: () => api.source(sourceId),
  })

  const testMutation = useMutation({
    mutationFn: () => api.testSource(sourceId),
    onSuccess: (data) => toast.success('Connection works', data.message),
    onError: (error) =>
      toast.error('Connection failed', error instanceof ApiError ? error.message : undefined),
  })

  const renameMutation = useMutation({
    mutationFn: () => api.updateSource(sourceId, { name }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['sources'] })
      void queryClient.invalidateQueries({ queryKey: ['source', sourceId] })
      toast.success('Source renamed')
      setRenameOpen(false)
    },
    onError: (error) =>
      toast.error('Could not rename', error instanceof ApiError ? error.message : undefined),
  })

  const deleteMutation = useMutation({
    mutationFn: () => api.deleteSource(sourceId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['sources'] })
      toast.success('Source deleted', source.data?.name)
      void navigate({ to: '/sources' })
    },
    onError: (error) =>
      toast.error(
        'Could not delete the source',
        error instanceof ApiError ? error.message : undefined,
      ),
  })

  const webhookUrl = `${window.location.origin}/api/v1/webhooks/github`

  return (
    <div className="flex flex-col gap-6">
      <div>
        <Link
          to="/sources"
          className="inline-flex items-center gap-1.5 text-xs text-muted transition-colors hover:text-foreground"
        >
          <ArrowLeft aria-hidden className="size-3.5" />
          All sources
        </Link>
      </div>

      {source.isPending ? <Skeleton className="h-20" /> : null}

      {source.isError ? (
        <EmptyState
          icon={<GitBranch aria-hidden />}
          title="Source not found"
          description="It may have been deleted. Head back to the sources list."
          action={
            <Link to="/sources">
              <Button variant="secondary">Back to sources</Button>
            </Link>
          }
        />
      ) : null}

      {source.data ? (
        <>
          <PageHeader
            title={source.data.name}
            description={
              source.data.type === 'github_app'
                ? `GitHub App ${source.data.app_slug || ''}`.trim()
                : undefined
            }
            actions={
              <>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => testMutation.mutate()}
                  loading={testMutation.isPending}
                >
                  <RefreshCw aria-hidden className="size-3.5" />
                  Test connection
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setName(source.data?.name ?? '')
                    setRenameOpen(true)
                  }}
                >
                  <Pencil aria-hidden className="size-3.5" />
                  Rename
                </Button>
                <Button variant="ghost" size="sm" onClick={() => setDeleteOpen(true)}>
                  <Trash2 aria-hidden className="size-3.5 text-danger" />
                  <span className="text-danger">Delete</span>
                </Button>
              </>
            }
          />

          <Card>
            <CardContent className="flex flex-col gap-3">
              <div className="flex items-center gap-2">
                {source.data.type === 'github_app' ? (
                  <GitHubIcon className="size-4 text-muted" />
                ) : (
                  <GitBranch aria-hidden className="size-4 text-muted" />
                )}
                <span className="text-sm font-medium text-foreground">
                  {source.data.type === 'github_app' ? 'GitHub App' : 'Git source'}
                </span>
                {source.data.type === 'github_app' ? (
                  source.data.installation_id ? (
                    <Badge variant="success">Installed</Badge>
                  ) : (
                    <Badge variant="warning">Not installed</Badge>
                  )
                ) : (
                  <Badge variant="outline">
                    {source.data.auth_type === 'none'
                      ? 'public'
                      : source.data.auth_type === 'token'
                        ? 'access token'
                        : 'ssh key'}
                  </Badge>
                )}
              </div>

              {source.data.type === 'github_app' ? (
                <>
                  <InfoRow label="App ID">
                    <span className="font-mono">{source.data.app_id}</span>
                  </InfoRow>
                  <InfoRow label="Slug">
                    <span className="font-mono">{source.data.app_slug || '—'}</span>
                  </InfoRow>
                  <InfoRow label="Installation">
                    <span className="font-mono">{source.data.installation_id || '—'}</span>
                  </InfoRow>
                  <InfoRow label="Webhook URL">
                    <Copyable value={webhookUrl} className="max-w-md" />
                  </InfoRow>
                  {source.data.html_url ? (
                    <InfoRow label="On GitHub">
                      <a
                        href={source.data.html_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1.5 text-accent transition-opacity hover:opacity-80"
                      >
                        {source.data.html_url}
                        <ExternalLink aria-hidden className="size-3" />
                      </a>
                    </InfoRow>
                  ) : null}
                </>
              ) : (
                <>
                  <InfoRow label="Repository">
                    <Copyable value={source.data.git_url} className="max-w-md" />
                  </InfoRow>
                  <InfoRow label="Authentication">
                    <span>
                      {source.data.auth_type === 'none'
                        ? 'Public repository'
                        : source.data.auth_type === 'token'
                          ? 'Access token (encrypted at rest)'
                          : 'SSH private key (encrypted at rest)'}
                    </span>
                  </InfoRow>
                </>
              )}

              <InfoRow label="Added">
                <span>{formatDateTime(source.data.created_at)}</span>
              </InfoRow>
            </CardContent>
          </Card>

          {source.data.type === 'github_app' && !source.data.installation_id ? (
            <InstallationPanel source={source.data} />
          ) : null}

          {source.data.type === 'github_app' ? (
            <Section
              title="Repositories"
              description="Repositories this installation can access. Expand a repository to see its branches."
            >
              <RepositoryList source={source.data} />
            </Section>
          ) : null}

          <Modal
            open={renameOpen}
            onOpenChange={setRenameOpen}
            title="Rename source"
            size="sm"
          >
            <form
              className="flex flex-col gap-4"
              onSubmit={(event) => {
                event.preventDefault()
                renameMutation.mutate()
              }}
            >
              <Field label="Name">
                {(field) => (
                  <Input
                    {...field}
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                    autoFocus
                  />
                )}
              </Field>
              <div className="flex items-center justify-end gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => setRenameOpen(false)}
                  disabled={renameMutation.isPending}
                >
                  Cancel
                </Button>
                <Button type="submit" variant="primary" loading={renameMutation.isPending}>
                  Save
                </Button>
              </div>
            </form>
          </Modal>

          <ConfirmDialog
            open={deleteOpen}
            onOpenChange={setDeleteOpen}
            title={`Delete ${source.data.name}?`}
            description="Stored credentials for this source will be removed. Applications using it must be updated."
            confirmLabel="Delete source"
            danger
            loading={deleteMutation.isPending}
            onConfirm={() => deleteMutation.mutate()}
          />
        </>
      ) : null}
    </div>
  )
}
