import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { IconHistory, IconPlayerPlay, IconPlus } from '@tabler/icons-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { BackupRecord } from '@/api/types'
import { BackupDialog } from '@/components/backups/backup-dialog'
import { BackupRunsTable } from '@/components/backups/runs-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Modal } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatRelative } from '@/lib/format'

// DatabaseBackups lists the schedules that target one database, with history.
export function DatabaseBackups({ databaseId }: { databaseId: string; databaseName?: string }) {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [historyUUID, setHistoryUUID] = useState('')

  const backups = useQuery({ queryKey: ['backups'], queryFn: () => api.backups(), refetchInterval: 20_000 })

  const detail = useQuery({
    queryKey: ['backup', historyUUID],
    queryFn: () => api.backup(historyUUID),
    enabled: historyUUID !== '',
  })

  const run = useMutation({
    mutationFn: (backup: BackupRecord) => api.runBackup(backup.uuid),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['backups'] })
      if (result.status === 'success') {
        toast.success('Backup finished', `${(result.size_bytes / 1024).toFixed(1)} kB in ${(result.duration_ms / 1000).toFixed(1)}s`)
      } else {
        toast.error('Backup failed', result.error)
      }
    },
    onError: (error) =>
      toast.error('Backup failed', error instanceof ApiError ? error.message : undefined),
  })

  const rows = (backups.data ?? []).filter(
    (backup) => backup.target === 'database' && backup.resource_id === databaseId,
  )

  return (
    <div className="flex flex-col gap-3">
      {backups.isPending ? <Skeleton className="h-20" /> : null}

      {backups.isSuccess && rows.length === 0 ? (
        <p className="hatch rounded-control border border-border px-4 py-6 text-center text-xs text-muted">
          No backup schedule targets this database yet.
        </p>
      ) : null}

      {rows.map((backup) => (
        <div
          key={backup.uuid}
          className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border px-3 py-2"
        >
          <div className="flex min-w-0 flex-col">
            <div className="flex items-center gap-2">
              <span className="truncate text-xs font-medium text-foreground">{backup.name}</span>
              {backup.last_status ? (
                <Badge variant={backup.last_status === 'success' ? 'success' : 'danger'}>{backup.last_status}</Badge>
              ) : null}
              <Badge variant={backup.enabled ? 'info' : 'outline'}>{backup.enabled ? 'scheduled' : 'paused'}</Badge>
            </div>
            <span className="text-2xs text-subtle">
              {backup.schedule_hint || backup.schedule} · keep {backup.retention} ·{' '}
              {backup.last_run_at ? `last run ${formatRelative(backup.last_run_at)}` : 'never run'}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="secondary" size="sm" loading={run.isPending} onClick={() => run.mutate(backup)}>
              <IconPlayerPlay aria-hidden className="size-3.5" />
              Run now
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setHistoryUUID(backup.uuid)}>
              <IconHistory aria-hidden className="size-3.5" />
              History
            </Button>
          </div>
        </div>
      ))}

      <div className="flex justify-end">
        <Button variant="secondary" size="sm" onClick={() => setDialogOpen(true)}>
          <IconPlus aria-hidden className="size-3.5" />
          Schedule a backup
        </Button>
      </div>

      <BackupDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        databaseUuid={databaseId}
      />

      <Modal
        open={historyUUID !== ''}
        onOpenChange={(open) => !open && setHistoryUUID('')}
        title="Backup history"
        description="Download an archive or put it back into this database."
        size="lg"
      >
        {detail.isPending ? (
          <Skeleton className="h-40" />
        ) : (
          <BackupRunsTable backupUUID={historyUUID} runs={detail.data?.runs ?? []} />
        )}
      </Modal>
    </div>
  )
}
