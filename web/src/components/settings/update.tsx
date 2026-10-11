import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { IconDownload, IconRefresh, IconRotateClockwise } from '@tabler/icons-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardTitle } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatDateTime } from '@/lib/format'

export function UpdateSection() {
  const queryClient = useQueryClient()
  const [applyOpen, setApplyOpen] = useState(false)
  const [rollbackOpen, setRollbackOpen] = useState(false)

  const status = useQuery({
    queryKey: ['update-status'],
    queryFn: () => api.updateStatus(false),
    refetchInterval: 60_000,
  })

  const check = useMutation({
    mutationFn: () => api.updateStatus(true),
    onSuccess: (result) => {
      queryClient.setQueryData(['update-status'], result)
      if (result.check_error) {
        toast.error('Update check failed', result.check_error)
      } else if (result.update_available) {
        toast.success(`ebox ${result.latest?.version} is available`)
      } else {
        toast.success('The panel is up to date')
      }
    },
    onError: (error) =>
      toast.error('Update check failed', error instanceof ApiError ? error.message : undefined),
  })

  const apply = useMutation({
    mutationFn: () => api.applyUpdate(false),
    onSuccess: (result) => {
      toast.success(result.message, 'This page will reconnect once the panel is back.')
      setApplyOpen(false)
    },
    onError: (error) =>
      toast.error('Update failed', error instanceof ApiError ? error.message : undefined),
  })

  const rollback = useMutation({
    mutationFn: () => api.rollbackUpdate(),
    onSuccess: (result) => {
      toast.success(result.message)
      setRollbackOpen(false)
    },
    onError: (error) =>
      toast.error('Rollback failed', error instanceof ApiError ? error.message : undefined),
  })

  const data = status.data

  return (
    <Card>
      <div className="flex flex-wrap items-start justify-between gap-3 p-4 pb-2">
        <div className="flex flex-col gap-1">
          <CardTitle>Updates</CardTitle>
          <CardDescription>
            The panel can update and restart itself when systemd manages it.
          </CardDescription>
        </div>
        <div className="flex items-center gap-2">
          {data?.update_available ? <Badge variant="info">update available</Badge> : null}
          <Button variant="secondary" size="sm" loading={check.isPending} onClick={() => check.mutate()}>
            <IconRefresh aria-hidden className="size-3.5" />
            Check now
          </Button>
        </div>
      </div>

      <CardContent className="flex flex-col gap-3">
        {status.isPending ? <Skeleton className="h-16" /> : null}

        {data ? (
          <>
            <dl className="grid gap-2 text-xs sm:grid-cols-2">
              <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2">
                <dt className="text-subtle">Installed</dt>
                <dd className="font-mono text-foreground">
                  {data.current_version}
                  <span className="text-subtle"> · {data.current_commit}</span>
                </dd>
              </div>
              <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2">
                <dt className="text-subtle">Newest release</dt>
                <dd className="font-mono text-foreground">
                  {data.latest ? data.latest.version : data.check_error ? 'check failed' : '—'}
                </dd>
              </div>
              <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2">
                <dt className="text-subtle">Built</dt>
                <dd className="text-foreground">{data.build_date === 'unknown' ? '—' : data.build_date}</dd>
              </div>
              <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2">
                <dt className="text-subtle">Asset</dt>
                <dd className="font-mono text-foreground">{data.asset_name}</dd>
              </div>
            </dl>

            {data.check_error ? (
              <p className="rounded-lg border border-warning/40 bg-warning-soft px-3 py-2 text-2xs text-warning">
                {data.check_error}
              </p>
            ) : null}

            {!data.self_updateable && data.reason ? (
              <p className="rounded-lg border border-border px-3 py-2 text-2xs text-subtle">{data.reason}</p>
            ) : null}

            {data.latest?.notes ? (
              <details className="rounded-lg border border-border px-3 py-2">
                <summary className="cursor-pointer text-xs text-foreground">
                  Release notes for {data.latest.version}
                </summary>
                <pre className="mt-2 max-h-56 overflow-auto font-mono text-2xs whitespace-pre-wrap text-muted">
                  {data.latest.notes}
                </pre>
              </details>
            ) : null}

            <div className="flex flex-wrap justify-end gap-2">
              <Button variant="ghost" size="sm" onClick={() => setRollbackOpen(true)}>
                <IconRotateClockwise aria-hidden className="size-3.5" />
                Roll back
              </Button>
              <Button
                variant="primary"
                size="sm"
                disabled={!data.self_updateable || !data.update_available}
                onClick={() => setApplyOpen(true)}
              >
                <IconDownload aria-hidden className="size-3.5" />
                {data.update_available ? `Install ${data.latest?.version ?? ''}` : 'Up to date'}
              </Button>
            </div>

            <p className="text-2xs text-subtle">
              Last checked {formatDateTime(data.checked_at)}
            </p>
          </>
        ) : null}
      </CardContent>

      <ConfirmDialog
        open={applyOpen}
        onOpenChange={setApplyOpen}
        title={`Install ${data?.latest?.version ?? 'the update'}?`}
        description="The new binary is downloaded, verified against the release checksum and swapped in. The panel restarts and comes back in a few seconds."
        confirmLabel="Install and restart"
        loading={apply.isPending}
        onConfirm={() => apply.mutate()}
      />

      <ConfirmDialog
        open={rollbackOpen}
        onOpenChange={setRollbackOpen}
        title="Roll back to the previous version?"
        description="The binary installed before the last update is restored and the panel restarts."
        confirmLabel="Roll back and restart"
        danger
        loading={rollback.isPending}
        onConfirm={() => rollback.mutate()}
      />
    </Card>
  )
}
