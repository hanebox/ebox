import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Download, RotateCcw, Trash2 } from 'lucide-react'
import { useState } from 'react'

import { api, backupDownloadURL } from '@/api/endpoints'
import type { BackupRun } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatDateTime } from '@/lib/format'
import { formatBytes, runStatusVariant } from '@/components/backups/shared'

export function BackupRunsTable({ backupUUID, runs }: { backupUUID: string; runs: BackupRun[] }) {
  const queryClient = useQueryClient()
  const [restoreTarget, setRestoreTarget] = useState<BackupRun | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<BackupRun | null>(null)

  const restore = useMutation({
    mutationFn: (run: BackupRun) => api.restoreBackup(backupUUID, run.id),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['backup', backupUUID] })
      toast.success(result.message)
      setRestoreTarget(null)
    },
    onError: (error) =>
      toast.error('Restore failed', error instanceof ApiError ? error.message : undefined),
  })

  const remove = useMutation({
    mutationFn: (run: BackupRun) => api.deleteBackupRun(backupUUID, run.id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backup', backupUUID] })
      toast.success('Run deleted')
      setDeleteTarget(null)
    },
  })

  if (runs.length === 0) {
    return (
      <p className="rounded-lg border border-dashed border-border px-4 py-6 text-center text-xs text-muted">
        No runs yet. Press Run now to take the first backup.
      </p>
    )
  }

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Started</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Size</TableHead>
            <TableHead>Duration</TableHead>
            <TableHead>File</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {runs.map((run) => (
            <TableRow key={run.id}>
              <TableCell className="text-xs text-foreground">
                {run.started_at ? formatDateTime(run.started_at) : '—'}
              </TableCell>
              <TableCell>
                <Badge variant={runStatusVariant(run.status)}>{run.status}</Badge>
                {run.error ? (
                  <span className="mt-1 block max-w-xs truncate text-[10px] text-danger" title={run.error}>
                    {run.error}
                  </span>
                ) : null}
              </TableCell>
              <TableCell className="font-mono text-[11px] text-muted">{formatBytes(run.size_bytes)}</TableCell>
              <TableCell className="font-mono text-[11px] text-muted">
                {run.duration_ms ? `${(run.duration_ms / 1000).toFixed(1)}s` : '—'}
              </TableCell>
              <TableCell className="max-w-[14rem] truncate font-mono text-[11px] text-subtle">
                {run.file ? run.file.split('/').pop() : '—'}
              </TableCell>
              <TableCell>
                <div className="flex justify-end gap-1">
                  {run.file ? (
                    <>
                      <a href={backupDownloadURL(backupUUID, run.id)} download>
                        <Button variant="ghost" size="sm" aria-label="Download backup">
                          <Download aria-hidden className="size-3.5" />
                        </Button>
                      </a>
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label="Restore backup"
                        onClick={() => setRestoreTarget(run)}
                      >
                        <RotateCcw aria-hidden className="size-3.5" />
                      </Button>
                    </>
                  ) : null}
                  <Button
                    variant="ghost"
                    size="sm"
                    aria-label="Delete run"
                    onClick={() => setDeleteTarget(run)}
                  >
                    <Trash2 aria-hidden className="size-3.5" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <ConfirmDialog
        open={restoreTarget !== null}
        onOpenChange={(open) => !open && setRestoreTarget(null)}
        title="Restore this backup?"
        description="The archive is piped back into the target. Existing data is overwritten where the dump contains it."
        confirmLabel="Restore now"
        loading={restore.isPending}
        onConfirm={() => restoreTarget && restore.mutate(restoreTarget)}
      />

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Delete this run?"
        description="The archive file is removed from disk."
        confirmLabel="Delete run"
        danger
        loading={remove.isPending}
        onConfirm={() => deleteTarget && remove.mutate(deleteTarget)}
      />
    </>
  )
}

