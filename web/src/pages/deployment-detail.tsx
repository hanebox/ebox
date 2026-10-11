import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from '@tanstack/react-router'
import { ArrowLeft, CircleStop } from 'lucide-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { DeploymentStatus } from '@/api/types'
import { DeploymentLog } from '@/components/applications/deployment-log'
import { DeploymentStatusBadge } from '@/components/applications/status'
import { PageHeader } from '@/components/layout/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatDateTime, formatRelative } from '@/lib/format'

export function DeploymentDetailPage() {
  const { deploymentId } = useParams({ strict: false }) as { deploymentId: string }
  const queryClient = useQueryClient()
  const [statusOverride, setStatusOverride] = useState<DeploymentStatus | null>(null)

  const deployment = useQuery({
    queryKey: ['deployment', deploymentId],
    queryFn: () => api.deployment(deploymentId),
    refetchInterval: (query) => {
      const status = query.state.data?.status

      return status && ['queued', 'building', 'deploying'].includes(status) ? 5000 : false
    },
  })

  const cancel = useMutation({
    mutationFn: () => api.cancelDeployment(deploymentId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['deployment', deploymentId] })
      void queryClient.invalidateQueries({ queryKey: ['deployments'] })
      toast.success('Deployment cancelled')
    },
    onError: (error) =>
      toast.error('Could not cancel', error instanceof ApiError ? error.message : undefined),
  })

  const data = deployment.data
  const status = statusOverride ?? data?.status ?? 'queued'
  const active = ['queued', 'building', 'deploying'].includes(status)

  return (
    <div className="flex flex-col gap-6">
      <div>
        <Link
          to="/deployments"
          className="inline-flex items-center gap-1.5 text-xs text-muted transition-colors hover:text-foreground"
        >
          <ArrowLeft aria-hidden className="size-3.5" />
          All deployments
        </Link>
      </div>

      {deployment.isPending ? <Skeleton className="h-20" /> : null}

      {data ? (
        <>
          <PageHeader
            title={data.application_name || 'Deployment'}
            description={`${data.uuid} · triggered by ${data.trigger}`}
            actions={
              <>
                {data.application_uuid ? (
                  <Link
                    to="/applications/$applicationId"
                    params={{ applicationId: data.application_uuid }}
                  >
                    <Button variant="secondary" size="sm">
                      Open application
                    </Button>
                  </Link>
                ) : null}
                {active ? (
                  <Button
                    variant="ghost"
                    size="sm"
                    loading={cancel.isPending}
                    onClick={() => cancel.mutate()}
                  >
                    <CircleStop aria-hidden className="size-3.5 text-danger" />
                    <span className="text-danger">Cancel</span>
                  </Button>
                ) : null}
              </>
            }
          />

          <Card>
            <CardContent className="flex flex-wrap items-center gap-x-6 gap-y-3">
              <DeploymentStatusBadge status={status} />
              <div className="flex flex-col gap-0.5">
                <span className="text-[11px] tracking-wide text-subtle uppercase">Commit</span>
                <span className="font-mono text-xs text-foreground">
                  {data.commit ? data.commit.slice(0, 7) : '—'}
                </span>
              </div>
              <div className="flex flex-col gap-0.5">
                <span className="text-[11px] tracking-wide text-subtle uppercase">Message</span>
                <span className="max-w-md truncate text-xs text-foreground">
                  {data.commit_message || '—'}
                </span>
              </div>
              <div className="flex flex-col gap-0.5">
                <span className="text-[11px] tracking-wide text-subtle uppercase">Image</span>
                <span className="font-mono text-xs text-foreground">{data.image_tag || '—'}</span>
              </div>
              <div className="flex flex-col gap-0.5">
                <span className="text-[11px] tracking-wide text-subtle uppercase">Started</span>
                <span className="text-xs text-foreground">
                  {data.started_at ? formatRelative(data.started_at) : formatDateTime(data.created_at)}
                </span>
              </div>
              <div className="flex flex-col gap-0.5">
                <span className="text-[11px] tracking-wide text-subtle uppercase">Duration</span>
                <span className="text-xs text-foreground">
                  {data.duration_ms ? `${Math.round(data.duration_ms / 1000)}s` : '—'}
                </span>
              </div>
            </CardContent>
          </Card>

          {data.error ? (
            <p className="rounded-lg border border-danger/40 bg-danger-soft p-3 text-xs text-danger">
              {data.error}
            </p>
          ) : null}

          <DeploymentLog
            deploymentId={deploymentId}
            status={status}
            onStatusChange={(next) => {
              setStatusOverride(next)
              void queryClient.invalidateQueries({ queryKey: ['deployment', deploymentId] })
            }}
          />
        </>
      ) : null}
    </div>
  )
}
