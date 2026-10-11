import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { IconDotsVertical, IconExternalLink, IconGitBranch, IconPlugConnected, IconRefresh, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { Source } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { GitHubIcon } from '@/components/ui/icons'
import { DropdownItem, DropdownMenu, DropdownSeparator } from '@/components/ui/menu'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'

export function SourceCard({ source }: { source: Source }) {
  const queryClient = useQueryClient()
  const [deleteOpen, setDeleteOpen] = useState(false)

  const isGithub = source.type === 'github_app'
  const installed = isGithub && Boolean(source.installation_id)
  const Icon = isGithub ? GitHubIcon : IconGitBranch

  const testMutation = useMutation({
    mutationFn: () => api.testSource(source.uuid),
    onSuccess: (data) => toast.success('Connection works', data.message),
    onError: (error) =>
      toast.error('Connection failed', error instanceof ApiError ? error.message : undefined),
  })

  const deleteMutation = useMutation({
    mutationFn: () => api.deleteSource(source.uuid),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['sources'] })
      toast.success('Source deleted', source.name)
      setDeleteOpen(false)
    },
    onError: (error) =>
      toast.error(
        'Could not delete the source',
        error instanceof ApiError ? error.message : undefined,
      ),
  })

  return (
    <div className="flex flex-col gap-3 rounded-panel border border-border bg-surface p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          <div className="flex size-8 shrink-0 items-center justify-center rounded-lg border border-border bg-surface-raised text-muted">
            <Icon aria-hidden className="size-4" />
          </div>
          <div className="flex min-w-0 flex-col gap-1">
            <div className="flex flex-wrap items-center gap-2">
              <p className="truncate text-sm font-medium text-foreground">{source.name}</p>
              <Badge variant="outline">{isGithub ? 'GitHub App' : 'Git'}</Badge>
              {isGithub ? (
                installed ? (
                  <Badge variant="success">Installed</Badge>
                ) : (
                  <Badge variant="warning">Not installed</Badge>
                )
              ) : null}
            </div>
            <p className="truncate font-mono text-2xs text-muted">
              {isGithub
                ? `${source.app_slug || 'unknown app'} · App ID ${source.app_id}`
                : source.git_url}
            </p>
            {!isGithub && source.auth_type !== 'none' ? (
              <p className="text-2xs text-subtle">
                Authenticated with {source.auth_type === 'token' ? 'an access token' : 'an SSH key'}
              </p>
            ) : null}
          </div>
        </div>

        <DropdownMenu
          trigger={
            <Button variant="ghost" size="icon" aria-label={`Actions for ${source.name}`}>
              <IconDotsVertical aria-hidden className="size-4" />
            </Button>
          }
        >
          <DropdownItem onClick={() => testMutation.mutate()}>
            <IconRefresh aria-hidden className="size-3.5" />
            Test connection
          </DropdownItem>
          {isGithub && source.html_url ? (
            <DropdownItem
              onClick={() => window.open(source.html_url, '_blank', 'noopener,noreferrer')}
            >
              <IconExternalLink aria-hidden className="size-3.5" />
              Open on GitHub
            </DropdownItem>
          ) : null}
          <DropdownSeparator />
          <DropdownItem danger onClick={() => setDeleteOpen(true)}>
            <IconTrash aria-hidden className="size-3.5" />
            Delete source
          </DropdownItem>
        </DropdownMenu>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Link to="/sources/$sourceId" params={{ sourceId: source.uuid }}>
          <Button variant="secondary" size="sm">
            {isGithub ? 'Repositories' : 'Details'}
          </Button>
        </Link>

        {isGithub && !installed && source.app_slug ? (
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
            <IconPlugConnected aria-hidden className="size-3.5" />
            Install app
          </Button>
        ) : null}

        <Button
          variant="ghost"
          size="sm"
          onClick={() => testMutation.mutate()}
          loading={testMutation.isPending}
        >
          Test
        </Button>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={`Delete ${source.name}?`}
        description={
          isGithub
            ? 'The app credentials will be removed from the panel. The GitHub App itself stays on GitHub.'
            : 'Stored credentials for this source will be removed. This cannot be undone.'
        }
        confirmLabel="Delete source"
        danger
        loading={deleteMutation.isPending}
        onConfirm={() => deleteMutation.mutate()}
      />
    </div>
  )
}
