import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { IconArchive, IconHistory, IconPlayerPlay, IconPlus, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { BackupRecord } from '@/api/types'
import { BackupDialog } from '@/components/backups/backup-dialog'
import { BackupRunsTable } from '@/components/backups/runs-table'
import { formatBytes } from '@/components/backups/shared'
import { PageHeader } from '@/components/layout/page-header'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { EmptyState } from '@/components/ui/empty-state'
import { Modal } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatRelative } from '@/lib/format'

function BackupCard({ backup }: { backup: BackupRecord }) {
  const queryClient = useQueryClient()
  const [historyOpen, setHistoryOpen] = useState(false)

  const detail = useQuery({
    queryKey: ['backup', backup.uuid],
    queryFn: () => api.backup(backup.uuid),
    enabled: historyOpen,
  })

  const run = useMutation({
    mutationFn: () => api.runBackup(backup.uuid),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['backups'] })
      void queryClient.invalidateQueries({ queryKey: ['backup', backup.uuid] })
      if (result.status === 'success') {
        toast.success('Backup finished', `${formatBytes(result.size_bytes)} in ${(result.duration_ms / 1000).toFixed(1)}s`)
      } else {
        toast.error('Backup failed', result.error)
      }
    },
    onError: (error) =>
      toast.error('Backup failed', error instanceof ApiError ? error.message : undefined),
  })

  const remove = useMutation({
    mutationFn: () => api.deleteBackup(backup.uuid),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backups'] })
      toast.success('Backup deleted')
    },
  })

  return (
    <Card className="flex flex-col gap-3 p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col">
          <span className="truncate text-sm font-medium text-foreground">{backup.name}</span>
          <span className="truncate text-2xs text-subtle">
            {backup.target === 'database' ? 'Database' : 'Volume'} · {backup.schedule_hint || backup.schedule} · keep{' '}
            {backup.retention}
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          {backup.last_status ? (
            <Badge variant={backup.last_status === 'success' ? 'success' : 'danger'}>{backup.last_status}</Badge>
          ) : null}
          <Badge variant={backup.enabled ? 'info' : 'outline'}>{backup.enabled ? 'scheduled' : 'paused'}</Badge>
        </div>
      </div>

      <div className="flex flex-wrap gap-x-4 gap-y-1 text-2xs text-subtle">
        <span>{backup.last_run_at ? `last run ${formatRelative(backup.last_run_at)}` : 'never run'}</span>
        {backup.next_run_at ? <span>next {formatRelative(backup.next_run_at)}</span> : null}
      </div>

      {backup.last_error ? (
        <p className="max-h-16 overflow-hidden font-mono text-2xs text-danger">{backup.last_error}</p>
      ) : null}

      <div className="flex flex-wrap justify-end gap-2">
        <Button variant="secondary" size="sm" loading={run.isPending} onClick={() => run.mutate()}>
          <IconPlayerPlay aria-hidden className="size-3.5" />
          {run.isPending ? 'Running…' : 'Run now'}
        </Button>
        <Button variant="ghost" size="sm" onClick={() => setHistoryOpen(true)}>
          <IconHistory aria-hidden className="size-3.5" />
          History
        </Button>
        <Button variant="ghost" size="sm" aria-label="Delete backup" onClick={() => remove.mutate()}>
          <IconTrash aria-hidden className="size-3.5" />
        </Button>
      </div>

      <Modal
        open={historyOpen}
        onOpenChange={setHistoryOpen}
        title={`${backup.name} runs`}
        description="Download an archive or put it back into the target."
        size="lg"
      >
        {detail.isPending ? (
          <Skeleton className="h-40" />
        ) : (
          <BackupRunsTable backupUUID={backup.uuid} runs={detail.data?.runs ?? []} />
        )}
      </Modal>
    </Card>
  )
}

export function BackupsPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const backups = useQuery({ queryKey: ['backups'], queryFn: () => api.backups(), refetchInterval: 15_000 })

  const rows = backups.data ?? []

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Backups"
        description="Scheduled dumps of your databases and archives of your volumes."
        actions={
          <Button variant="primary" size="sm" onClick={() => setDialogOpen(true)}>
            <IconPlus aria-hidden className="size-3.5" />
            New backup
          </Button>
        }
      />

      {backups.isPending ? (
        <div className="grid gap-3 lg:grid-cols-2">
          <Skeleton className="h-36" />
          <Skeleton className="h-36" />
        </div>
      ) : null}

      {backups.isSuccess && rows.length === 0 ? (
        <Card className="p-6">
          <EmptyState
            icon={<IconArchive aria-hidden />}
            title="No backups yet"
            description="Schedule a nightly dump for a database, or archive a volume."
            action={
              <Button variant="primary" size="sm" onClick={() => setDialogOpen(true)}>
                <IconPlus aria-hidden className="size-3.5" />
                New backup
              </Button>
            }
          />
        </Card>
      ) : null}

      {rows.length > 0 ? (
        <div className="grid gap-3 lg:grid-cols-2">
          {rows.map((backup) => (
            <BackupCard key={backup.uuid} backup={backup} />
          ))}
        </div>
      ) : null}

      <p className="text-2xs text-subtle">
        Archives live in the panel data directory. Need a database backed up on demand instead?{' '}
        <Link to="/databases" className="text-accent hover:underline">
          Open a database
        </Link>{' '}
        and press Run now from its Backups tab.
      </p>

      <BackupDialog open={dialogOpen} onOpenChange={setDialogOpen} />
    </div>
  )
}
