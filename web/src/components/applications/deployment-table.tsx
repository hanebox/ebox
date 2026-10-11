import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { IconCircleX, IconExternalLink, IconRocket, IconRotateClockwise } from '@tabler/icons-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { Deployment } from '@/api/types'
import { DeploymentStatusBadge } from '@/components/applications/status'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { DropdownItem, DropdownMenu, DropdownSeparator } from '@/components/ui/menu'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatRelative } from '@/lib/format'
import { cn } from '@/lib/cn'

function formatDuration(milliseconds: number): string {
  if (!milliseconds) {
    return '–'
  }

  const seconds = Math.round(milliseconds / 1000)
  if (seconds < 60) {
    return `${seconds}s`
  }

  return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
}

export interface DeploymentTableProps {
  deployments: Deployment[]
  showApplication?: boolean
  onRollback?: (deployment: Deployment) => void
}

export function DeploymentTable({ deployments, showApplication = false, onRollback }: DeploymentTableProps) {
  const queryClient = useQueryClient()
  const [cancelTarget, setCancelTarget] = useState<Deployment | null>(null)

  const cancel = useMutation({
    mutationFn: (uuid: string) => api.cancelDeployment(uuid),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['deployments'] })
      toast.success('Deployment cancelled')
      setCancelTarget(null)
    },
    onError: (error) => {
      toast.error('Could not cancel', error instanceof ApiError ? error.message : undefined)
      setCancelTarget(null)
    },
  })

  if (deployments.length === 0) {
    return (
      <p className="hatch rounded-control border border-border px-4 py-8 text-center text-xs text-muted">
        No deployments yet.
      </p>
    )
  }

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            {showApplication ? <TableHead>Application</TableHead> : null}
            <TableHead>Status</TableHead>
            <TableHead>Commit</TableHead>
            <TableHead>Trigger</TableHead>
            <TableHead>Duration</TableHead>
            <TableHead>Started</TableHead>
            <TableHead className="text-end">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {deployments.map((deployment) => {
            const active = ['queued', 'building', 'deploying'].includes(deployment.status)

            return (
              <TableRow key={deployment.uuid}>
                {showApplication ? (
                  <TableCell className="text-xs whitespace-nowrap">
                    {deployment.application_uuid ? (
                      <Link
                        to="/applications/$applicationId"
                        params={{ applicationId: deployment.application_uuid }}
                        className="text-foreground hover:underline"
                      >
                        {deployment.application_name || 'application'}
                      </Link>
                    ) : (
                      <span className="text-muted">{deployment.application_name || '–'}</span>
                    )}
                  </TableCell>
                ) : null}
                <TableCell>
                  <DeploymentStatusBadge status={deployment.status} />
                </TableCell>
                <TableCell className="max-w-72 truncate text-xs text-muted">
                  {deployment.commit ? (
                    <span className="font-mono">{deployment.commit.slice(0, 7)}</span>
                  ) : (
                    <span className="text-subtle">–</span>
                  )}
                  {deployment.commit_message ? ` · ${deployment.commit_message}` : ''}
                </TableCell>
                <TableCell className="text-xs whitespace-nowrap text-muted">{deployment.trigger}</TableCell>
                <TableCell className="tabular text-xs whitespace-nowrap text-muted">
                  {formatDuration(deployment.duration_ms)}
                </TableCell>
                <TableCell className="text-xs whitespace-nowrap text-muted">
                  {formatRelative(deployment.created_at)}
                </TableCell>
                <TableCell className="text-end">
                  <div className="flex items-center justify-end gap-1.5">
                    <Link to="/deployments/$deploymentId" params={{ deploymentId: deployment.uuid }}>
                      <Button variant="ghost" size="sm">
                        <IconExternalLink aria-hidden className="size-3.5" />
                        Log
                      </Button>
                    </Link>
                    {active ? (
                      <Button variant="ghost" size="sm" onClick={() => setCancelTarget(deployment)}>
                        <IconCircleX aria-hidden className="size-3.5 text-danger" />
                      </Button>
                    ) : null}
                    {onRollback && deployment.status === 'success' && deployment.image_tag ? (
                      <DropdownMenu
                        trigger={
                          <Button variant="ghost" size="sm" aria-label="More actions">
                            ⋮
                          </Button>
                        }
                      >
                        <DropdownItem onClick={() => onRollback(deployment)}>
                          <IconRotateClockwise aria-hidden className="size-3.5" />
                          Roll back to this version
                        </DropdownItem>
                        <DropdownSeparator />
                        <DropdownItem
                          onClick={() => {
                            if (deployment.application_uuid) {
                              void api.deployApplication(deployment.application_uuid).then(() => {
                                void queryClient.invalidateQueries({ queryKey: ['deployments'] })
                                toast.success('Redeploy queued')
                              })
                            }
                          }}
                        >
                          <IconRocket aria-hidden className="size-3.5" />
                          Redeploy current branch
                        </DropdownItem>
                      </DropdownMenu>
                    ) : null}
                  </div>
                </TableCell>
              </TableRow>
            )
          })}
        </TableBody>
      </Table>

      <ConfirmDialog
        open={cancelTarget !== null}
        onOpenChange={(open) => {
          if (!open) {
            setCancelTarget(null)
          }
        }}
        title="Cancel deployment"
        description="The running deployment stops as soon as the current step finishes."
        confirmLabel="Cancel deployment"
        danger
        loading={cancel.isPending}
        onConfirm={() => {
          if (cancelTarget) {
            cancel.mutate(cancelTarget.uuid)
          }
        }}
      />
    </>
  )
}

export function DeploymentError({ deployment }: { deployment: Deployment }) {
  if (!deployment.error) {
    return null
  }

  return (
    <p className={cn('rounded-lg border border-danger/40 bg-danger-soft p-3 text-xs text-danger')}>
      {deployment.error}
    </p>
  )
}
